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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	appscan "github.com/promptshield/promptshield/internal/application/scan"
	"github.com/promptshield/promptshield/internal/discovery"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Simple HTTP server stub for the PromptShield enforcer.
// Exposes:
// - GET /healthz: liveness probe
// - POST /check: minimal allow/deny stub (currently always allow)

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

func init() {
	// Best-effort registration; ignore panics on duplicate in tests
	prometheus.MustRegister(enforcerRequests, enforcerDecisions, enforcerReqDuration)
}

func NewMux() http.Handler {
	r := chi.NewRouter()
	// Middlewares: request id, real ip, recoverer, timeout (defense in depth; keep short)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

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
	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Optional bearer token check
		reqToken := os.Getenv("PS_ENFORCER_AUTH_TOKEN")
		if reqToken != "" {
			if !httpAuthOK(r, reqToken) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				enforcerRequests.WithLabelValues("/check", "401").Inc()
				return
			}
		}
		if !license.IsLicensed() {
			w.Header().Set("X-PromptShield-License", "EVALUATION")
			if !license.AllowEvalRequest() {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("Rate limit exceeded in evaluation mode"))
				enforcerRequests.WithLabelValues("/check", "429").Inc()
				return
			}
		} else {
			w.Header().Set("X-PromptShield-License", "LICENSED")
		}
		// Minimal PoC: read a small body to a temp file (or stdin), scan via existing orchestrator
		// For safety, enforce a small max size to avoid abuse in the stub
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		start := time.Now()

		tmp, err := os.CreateTemp("", "ps-check-*.txt")
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			enforcerRequests.WithLabelValues("/check", "500").Inc()
			return
		}
		defer func() { _ = os.Remove(tmp.Name()) }()

		// Cap read at 1MB for stub; allow override via env
		maxBytes := int64(1 << 20)
		if v := os.Getenv("PS_ENFORCER_MAX_BODY_BYTES"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				maxBytes = n
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if r.Body != nil {
			defer r.Body.Close()
			if _, err := tmp.ReadFrom(r.Body); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				enforcerRequests.WithLabelValues("/check", "400").Inc()
				return
			}
		}
		_ = tmp.Close()

		sc := scanner.New(0)
		// Load rulepack from env if set; fallback to common mount paths
		rulepack := os.Getenv("PS_ENFORCER_RULEPACK")
		if rulepack == "" {
			if _, err := os.Stat("/rules/basic-security.yaml"); err == nil {
				rulepack = "/rules/basic-security.yaml"
			} else if _, err := os.Stat("rules/basic-security.yaml"); err == nil {
				rulepack = "rules/basic-security.yaml"
			}
		}
		if rulepack != "" {
			if packs, e := rules.LoadPacks(rulepack); e == nil {
				sc.LoadRulePacks(packs)
			}
		}
		svc := appscan.NewService(sc)
		res, err := svc.Scan(ctx, []string{tmp.Name()}, appscan.Options{Workers: 1, PendingWindow: 32})
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			enforcerRequests.WithLabelValues("/check", "500").Inc()
			return
		}
		// Decision: allow if no violations; quarantine otherwise (stub)
		decision := "allow"
		reason := "no_signals"
		total := 0
		// Response-action aware decisioning
		anyQuarantine := false
		anyDeny := false
		firstRule := ""
		for _, r := range res {
			total += len(r.Violations)
			for _, v := range r.Violations {
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
		_ = json.NewEncoder(w).Encode(map[string]any{"decision": decision, "reason": reason, "violations": total, "request_id": reqID})
		enforcerDecisions.WithLabelValues(decision).Inc()
		enforcerRequests.WithLabelValues("/check", strconv.Itoa(statusCode)).Inc()
		enforcerReqDuration.WithLabelValues("/check", decision).Observe(time.Since(start).Seconds())
		_ = discovery.ErrNoInputFiles // keep imported until multi-path support
	}
	// Handle all /check paths for ext_authz path_prefix behavior
	r.Route("/check", func(checkRouter chi.Router) {
		checkRouter.HandleFunc("/*", handler) // /check/*
		checkRouter.HandleFunc("/", handler)  // /check
	})
	// Filter noisy endpoints from tracing
	return otelhttp.NewHandler(r, "ps_enforcer_http", otelhttp.WithFilter(func(r *http.Request) bool {
		p := r.URL.Path
		if p == "/healthz" || p == "/metrics" {
			return false
		}
		return true
	}))
}

// Serve starts an HTTP server on addr with sane timeouts.
func Serve(addr string) *http.Server {
	srv := &http.Server{
		Addr:              addr,
		Handler:           NewMux(),
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

func httpAuthOK(r *http.Request, want string) bool {
	if want == "" {
		return true
	}
	// Authorization: Bearer <token>
	if v := r.Header.Get("Authorization"); v != "" {
		if len(v) >= 7 && (v[:7] == "Bearer " || v[:7] == "bearer ") {
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
