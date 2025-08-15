package enforcerhttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/promptshield/promptshield/internal/discovery"
	"github.com/promptshield/promptshield/internal/encoding/jsonx"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

// HTTP server for the PromptShield enforcer.
// Exposes:
// - GET /healthz: liveness probe
// - GET /readyz: readiness probe with rule validation
// - GET /metrics: Prometheus metrics
// - POST /check: enforcement decision endpoint

// NewMux constructs the HTTP handler mux for the enforcer.
var (
	enforcerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ps_enforcer_requests_total", Help: "Total HTTP requests to enforcer"},
		[]string{"path", "code"},
	)
	enforcerDecisions = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ps_enforcer_decisions_total", Help: "Total decisions made by enforcer"},
		[]string{"decision"},
	)
	enforcerReqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "ps_enforcer_request_duration_seconds", Help: "Request duration in seconds", Buckets: prometheus.DefBuckets},
		[]string{"path", "decision"},
	)
)

// Optional performance toggles (env-driven)
func metricsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_DISABLE_METRICS")))
	return !(v == "1" || v == "true" || v == "yes")
}

func tracingEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_DISABLE_TRACING")))
	return !(v == "1" || v == "true" || v == "yes")
}

func fastMode() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_FAST")))
	return v == "1" || v == "true" || v == "yes"
}

func init() {
	// Best-effort registration; ignore panics on duplicate in tests
	prometheus.MustRegister(enforcerRequests, enforcerDecisions, enforcerReqDuration)
}

func NewMux() http.Handler { // backward-compatible wrapper
	adminToken := os.Getenv("PS_ENFORCER_ADMIN_TOKEN")
	return NewMuxWithOptions(api.Options{AdminToken: adminToken})
}

// NewMuxWithOptions constructs the HTTP handler mux with injectable API options.
func NewMuxWithOptions(apiOpt api.Options) http.Handler {
	r := chi.NewRouter()
	// Middlewares: request id, real ip, recoverer, timeout (defense in depth; keep short)
	if !fastMode() {
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(middleware.Recoverer)
		r.Use(middleware.Timeout(10 * time.Second))
	}
	// Global HTTP rate limiter from license entitlements; fallback to unlimited
	// Must be registered before any routes on chi mux
	if ent, ok := license.Entitlement(); ok && ent.MaxRPS > 0 {
		burst := 1
		if b := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS_BURST")); b != "" {
			if n, err := strconv.Atoi(b); err == nil && n > 0 {
				burst = n
			}
		}
		limiter := rate.NewLimiter(rate.Limit(ent.MaxRPS), burst)
		// Basic inflight bytes accounting for billing (optional)
		var inflight int64
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if !limiter.Allow() {
					w.Header().Set("content-type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					_ = json.NewEncoder(w).Encode(map[string]any{"code": "RESOURCE_EXHAUSTED", "message": "rate limit exceeded", "details": map[string]any{"max_rps": ent.MaxRPS}})
					return
				}
				// Track inflight bytes via Content-Length (approximate)
				if cl := req.Header.Get("Content-Length"); cl != "" {
					if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
						atomic.AddInt64(&inflight, n)
						defer atomic.AddInt64(&inflight, -n)
					}
				}
				next.ServeHTTP(w, req)
			})
		})
	}

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// Readiness gate: health plus rulepack/scanner ready
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Basic check: require rulepack env or default pack present
		if os.Getenv("PS_ENFORCER_RULEPACK") == "" {
			if _, err := os.Stat("/rules/basic-security.yaml"); err != nil {
				if _, err := os.Stat("rules/basic-security.yaml"); err != nil {
					http.Error(w, "not ready", http.StatusServiceUnavailable)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	// Prometheus metrics endpoint (optional)
	if metricsEnabled() {
		r.Handle("/metrics", promhttp.Handler())
	}

	// Mount v1 API
	r.Mount("/v1", api.NewMux(apiOpt))

	// Prepare scanner pool and optionally preload rulepacks to reduce per-request overhead
	var (
		preloadPacks []rules.RulePack
		scannerPool  = &sync.Pool{}
	)
	{
		rulepackPath := os.Getenv("PS_ENFORCER_RULEPACK")
		if rulepackPath == "" {
			if _, err := os.Stat("/rules/basic-security.yaml"); err == nil {
				rulepackPath = "/rules/basic-security.yaml"
			} else if _, err := os.Stat("rules/basic-security.yaml"); err == nil {
				rulepackPath = "rules/basic-security.yaml"
			}
		}
		if rulepackPath != "" {
			if packs, e := rules.LoadPacks(rulepackPath); e == nil {
				preloadPacks = packs
			}
		}
		var maxStreamBytes int64
		if v := os.Getenv("PS_ENFORCER_MAX_STREAM_BYTES"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				maxStreamBytes = n
			}
		}
		scannerPool.New = func() any {
			sc := scanner.New(0)
			if maxStreamBytes > 0 {
				sc.SetMaxStreamBytes(maxStreamBytes)
			}
			sc.SetQuarantineOnTimeout(true)
			sc.SetQuarantineOnError(true)
			if len(preloadPacks) > 0 {
				sc.LoadRulePacks(preloadPacks)
			}
			return sc
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Optional bearer token check
		reqToken := os.Getenv("PS_ENFORCER_AUTH_TOKEN")
		if reqToken != "" {
			if !HttpAuthOK(r, reqToken) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				if metricsEnabled() {
					enforcerRequests.WithLabelValues("/check", "401").Inc()
				}
				return
			}
		}
		if !license.IsLicensed() {
			w.Header().Set("X-PromptShield-License", "EVALUATION")
			if !license.AllowEvalRequest() {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("Rate limit exceeded in evaluation mode"))
				if metricsEnabled() {
					enforcerRequests.WithLabelValues("/check", "429").Inc()
				}
				return
			}
		} else {
			w.Header().Set("X-PromptShield-License", "LICENSED")
		}
		// Stream request body to scanner for real-time analysis
		// Enforce configurable body size limit for resource protection
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		start := time.Now()

		// Default 1MB limit; configurable via environment
		maxBytes := int64(1 << 20)
		if v := os.Getenv("PS_ENFORCER_MAX_BODY_BYTES"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				maxBytes = n
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if r.Body != nil {
			defer r.Body.Close()
		}

		sc := scannerPool.Get().(*scanner.Scanner)
		defer scannerPool.Put(sc)
		// Stream scan directly from the request body to avoid temp-file I/O
		res, err := sc.ScanReader(ctx, r.Body, "http:request")
		if err != nil {
			// Map body-size errors to 400; otherwise 500
			msg := err.Error()
			code := http.StatusInternalServerError
			api := map[string]any{"code": "INTERNAL", "message": "scan failed", "details": map[string]any{"error": msg}}
			if strings.Contains(strings.ToLower(msg), "request body too large") {
				code = http.StatusBadRequest
				api = map[string]any{"code": "INVALID_ARGUMENT", "message": "request body too large or invalid", "details": map[string]any{"max_bytes": maxBytes}}
			}
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(code)
			_ = jsonx.NewEncoder(w).Encode(api)
			if metricsEnabled() {
				enforcerRequests.WithLabelValues("/check", strconv.Itoa(code)).Inc()
			}
			return
		}
		// Decision logic: allow if no violations; quarantine/deny based on response actions
		decision := "allow"
		reason := "no_signals"
		total := 0
		// Response-action aware decisioning
		anyQuarantine := false
		anyDeny := false
		firstRule := ""
		// aggregate over struct result
		total = len(res.Violations)
		for _, v := range res.Violations {
			if firstRule == "" {
				firstRule = v.RuleID
			}
			a := v.ResponseAction
			switch a {
			case "deny", "block":
				anyDeny = true
			case "quarantine":
				anyQuarantine = true
			}
		}
		if anyDeny {
			decision = "deny"
			reason = firstNonEmpty(firstRule, "response_action")
		} else if anyQuarantine || total > 0 {
			decision = "quarantine"
			reason = firstNonEmpty(firstRule, "signals_detected")
		}
		// Request correlation id
		reqID := middleware.GetReqID(r.Context())
		if reqID == "" {
			reqID = generateRequestID()
		}
		// Map decision to HTTP status for use with Envoy ext_authz (HTTP service).
		// Honor enforcement mode: observe -> 200 always; enforce/quarantine -> 403 on violations.
		mode := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_MODE")))
		if mode == "" {
			mode = strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_ENFORCEMENT_MODE")))
		}
		statusCode := http.StatusOK
		if (mode == "enforce" || mode == "quarantine") && decision != "allow" {
			statusCode = http.StatusForbidden
		}

		// Trace correlation header
		if span := trace.SpanFromContext(r.Context()); span != nil {
			sc := span.SpanContext()
			if sc.IsValid() {
				w.Header().Set("x-ps-trace-id", sc.TraceID().String())
			}
		}
		w.Header().Set("x-ps-request-id", reqID)
		w.Header().Set("x-ps-decision", decision)
		w.Header().Set("x-ps-reason", reason)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(statusCode)
		_ = jsonx.NewEncoder(w).Encode(map[string]any{"decision": decision, "reason": reason, "violations": total, "request_id": reqID})
		if metricsEnabled() {
			enforcerDecisions.WithLabelValues(decision).Inc()
			enforcerRequests.WithLabelValues("/check", strconv.Itoa(statusCode)).Inc()
			enforcerReqDuration.WithLabelValues("/check", decision).Observe(time.Since(start).Seconds())
		}
		_ = discovery.ErrNoInputFiles // keep imported until multi-path support
	}
	// Handle all /check paths for ext_authz path_prefix behavior
	r.Route("/check", func(checkRouter chi.Router) {
		checkRouter.HandleFunc("/*", handler) // /check/*
		checkRouter.HandleFunc("/", handler)  // /check
	})
	// Filter noisy endpoints from tracing
	if tracingEnabled() {
		return otelhttp.NewHandler(r, "ps_enforcer_http", otelhttp.WithFilter(func(r *http.Request) bool {
			p := r.URL.Path
			if p == "/healthz" || p == "/metrics" {
				return false
			}
			return true
		}))
	}
	return r
}

// Serve starts an HTTP server on addr with sane timeouts.
func Serve(addr string) *http.Server {
	var srv *http.Server
	// Inject shutdown hooks into API mux
	adminToken := os.Getenv("PS_ENFORCER_ADMIN_TOKEN")
	apiMux := NewMuxWithOptions(api.Options{
		AdminToken: adminToken,
		OnDrain:    func(ctx context.Context) error { return nil },
		OnShutdown: func(ctx context.Context, delay time.Duration) error {
			if delay > 0 {
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			if srv != nil {
				return srv.Shutdown(ctx)
			}
			return nil
		},
	})
	srv = &http.Server{
		Addr:              addr,
		Handler:           apiMux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Optional TLS and mTLS
	certFile := os.Getenv("PS_ENFORCER_TLS_CERT")
	keyFile := os.Getenv("PS_ENFORCER_TLS_KEY")
	clientCA := os.Getenv("PS_ENFORCER_TLS_CLIENT_CA")
	if certFile != "" && keyFile != "" {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		if clientCA != "" {
			if caPEM, err := os.ReadFile(clientCA); err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caPEM) {
					tlsCfg.ClientCAs = pool
					tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
				}
			}
		}
		srv.TLSConfig = tlsCfg
		go func() {
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Printf("enforcer https server error: %v", err)
			}
		}()
	} else {
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("enforcer http server error: %v", err)
			}
		}()
	}
	return srv
}

func generateRequestID() string {
	return uuid.NewString()
}

// exported for reuse in api
func HttpAuthOK(r *http.Request, want string) bool {
	if want == "" {
		return true
	}
	// Authorization: Bearer <token>
	if v := r.Header.Get("Authorization"); v != "" {
		if len(v) >= 7 && (strings.HasPrefix(v, "Bearer ") || strings.HasPrefix(v, "bearer ")) {
			if v[7:] == want {
				return true
			}
		}
		if v == want {
			return true
		}
	}
	if v := r.Header.Get("X-PS-Token"); v != "" {
		if v == want {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
