package enforcerhttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/promptshield/promptshield/internal/audit"
	"github.com/promptshield/promptshield/internal/discovery"
	"github.com/promptshield/promptshield/internal/encoding/jsonx"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/security/paths"
	"github.com/promptshield/promptshield/internal/usage"
	redis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
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
	policyBypass = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ps_policy_bypass_total", Help: "Total requests served in policy bypass mode"},
		[]string{"reason"},
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

// envBool returns true when the provided env var is set to 1, true, yes, or on (case-insensitive).
func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func init() {
	// Best-effort registration; ignore panics on duplicate in tests
	prometheus.MustRegister(enforcerRequests, enforcerDecisions, enforcerReqDuration, policyBypass)
}

func NewMux() http.Handler { // backward-compatible wrapper
	return NewMuxWithOptions(getAPIOptions())
}

// getAPIOptions constructs API options from environment variables
func getAPIOptions() api.Options {
	adminToken := os.Getenv("PS_ENFORCER_ADMIN_TOKEN")

	// Optional insecure mode toggle for local dev / demos
	allowInsecure := false
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_ALLOW_INSECURE_ADMIN"))); v != "" {
		switch v {
		case "1", "true", "yes", "on":
			allowInsecure = true
		}
	}

	// OIDC configuration
	oidcConfig := api.OIDCConfig{
		Issuer:   os.Getenv("PS_ENFORCER_OIDC_ISSUER"),
		Audience: os.Getenv("PS_ENFORCER_OIDC_AUDIENCE"),
	}

	return api.Options{
		AdminToken:         adminToken,
		AllowInsecureAdmin: allowInsecure,
		OIDC:               oidcConfig,
	}
}

// NewMuxWithOptions constructs the HTTP handler mux with injectable API options.
func NewMuxWithOptions(apiOpt api.Options) http.Handler {
	r := chi.NewRouter()
	// Perform startup DB health check (optional). When DB is unreachable, we
	// downgrade enforcement mode to "observe" and mark readiness probe
	// unhealthy, but still allow the service to start (fail-open) so traffic
	// is not blocked during outages.
	dbHealthy := true
	if dsn := os.Getenv("PS_PG_DSN"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if pool, err := pg.NewPool(ctx, dsn); err == nil {
			pingCtx, cancelPing := context.WithTimeout(ctx, 2*time.Second)
			if _, err := pool.Raw().Exec(pingCtx, "SELECT 1"); err != nil {
				dbHealthy = false
			}
			cancelPing()
			pool.Close()
		} else {
			dbHealthy = false
		}
		if !dbHealthy {
			// Switch to observe mode to fail-open.
			os.Setenv("PS_ENFORCER_MODE", "observe")
			log.Printf("[WARN] Database unreachable at startup (%s); entering OBSERVE fail-open mode", dsn)
		}
	}

	// Middlewares: request id, real ip, recoverer, timeout (defense in depth; keep short)
	if !fastMode() {
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(middleware.Recoverer)
		r.Use(middleware.Timeout(10 * time.Second))
	}
	// Global HTTP rate limiter from license entitlements; fallback to unlimited
	// Must be registered before any routes on chi mux
	// preloadPacks declared early for readiness probe access; initialized later.
	var preloadPacks []rules.RulePack
	var scannerPool = &sync.Pool{}
	// Preload rulepacks early so readiness probe has accurate state.
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

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// Readiness gate: health plus rulepack/scanner ready
	requireAtStartup := envBool("PS_REQUIRE_RULEPACK_AT_STARTUP")
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		hasActivePack := len(preloadPacks) > 0
		ready := dbHealthy && (hasActivePack || !requireAtStartup)
		if !ready {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
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
	{
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

		// Emit policy bypass metric when serving with no active rulepack or when configured bypass is in effect.
		if len(preloadPacks) == 0 {
			if metricsEnabled() {
				policyBypass.WithLabelValues("no_rules").Inc()
			}
		} else if !requireAtStartup {
			if metricsEnabled() {
				policyBypass.WithLabelValues("config").Inc()
			}
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
	options := getAPIOptions()
	// Wire Redis-backed UsageStore when configured
	if options.UsageStore == nil {
		if addr := strings.TrimSpace(os.Getenv("PS_USAGE_REDIS_ADDR")); addr != "" {
			// Normalize prefix: default to PS_REGION or "ps"
			prefix := strings.TrimSpace(os.Getenv("PS_USAGE_PREFIX"))
			if prefix == "" {
				prefix = strings.TrimSpace(os.Getenv("PS_REGION"))
			}
			if prefix == "" {
				prefix = "ps"
			}
			// Optional TTL in days
			ttlDays := 35
			if v := strings.TrimSpace(os.Getenv("PS_USAGE_TTL_DAYS")); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					ttlDays = n
				}
			}
			rdb := redis.NewClient(&redis.Options{Addr: addr})
			options.UsageStore = usage.NewRedisUsageStore(rdb, prefix, time.Duration(ttlDays)*24*time.Hour)
		}
	}
	// Construct durable audit logger from environment and adapt to API interface
	var auditClose func() error
	if lgr, closeFn, err := audit.NewLoggerFromEnv(); err == nil && lgr != nil {
		options.AuditLogger = auditAdapter{inner: lgr}
		auditClose = closeFn
	}
	options.OnDrain = func(ctx context.Context) error { return nil }
	options.OnShutdown = func(ctx context.Context, delay time.Duration) error {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if auditClose != nil {
			_ = auditClose()
		}
		if options.UsageStore != nil {
			_ = options.UsageStore.Close(ctx)
		}
		if srv != nil {
			return srv.Shutdown(ctx)
		}
		return nil
	}
	apiMux := NewMuxWithOptions(options)
	srv = &http.Server{
		Addr:              addr,
		Handler:           apiMux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// TLS policy: require on non-loopback unless explicitly disabled; auto-detect certs when not provided
	mode := tlsMode()
	certFile, keyFile, havePair := findTLSPair()
	clientCA := os.Getenv("PS_ENFORCER_TLS_CLIENT_CA")
	nonLoop := !isLoopbackAddr(addr)

	startHTTPS := func(certFile, keyFile string) {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		if clientCA != "" {
			if err := paths.ValidateCAFilePath(clientCA); err != nil {
				log.Printf("invalid client CA file path: %v", err)
			} else if caPEM, err := os.ReadFile(clientCA); err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caPEM) {
					tlsCfg.ClientCAs = pool
					tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
				} else {
					log.Printf("failed to parse client CA certificate")
				}
			} else {
				log.Printf("failed to read client CA file: %v", err)
			}
		}
		srv.TLSConfig = tlsCfg
		go func() {
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Printf("enforcer https server error: %v", err)
			}
		}()
	}

	switch mode {
	case "disable":
		// Explicit insecure mode
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("enforcer http server error: %v", err)
			}
		}()
		return srv
	case "require":
		if !havePair {
			log.Fatalf("TLS required but no certificate configured for %s; set PS_ENFORCER_TLS_CERT and PS_ENFORCER_TLS_KEY or mount /tls/server.crt and /tls/server.key (set PS_ENFORCER_TLS_MODE=disable for local dev)", addr)
		}
		startHTTPS(certFile, keyFile)
		return srv
	default: // auto
		if nonLoop {
			if !havePair {
				log.Fatalf("Refusing to listen on non-loopback address %s without TLS; set PS_ENFORCER_TLS_CERT and PS_ENFORCER_TLS_KEY or mount /tls/server.crt and /tls/server.key, or set PS_ENFORCER_TLS_MODE=disable to allow insecure", addr)
			}
			startHTTPS(certFile, keyFile)
			return srv
		}
		// loopback: prefer TLS if available, else plain HTTP
		if havePair {
			startHTTPS(certFile, keyFile)
		} else {
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("enforcer http server error: %v", err)
				}
			}()
		}
	}
	return srv
}

func generateRequestID() string {
	return uuid.NewString()
}

// auditAdapter adapts internal/audit.Logger to api.AuditLogger
type auditAdapter struct{ inner audit.Logger }

func (a auditAdapter) Log(ev api.AuditEvent) error {
	return a.inner.Log(audit.Event{Type: ev.Type, Data: ev.Data, Hash: ev.Hash, PrevHash: ev.PrevHash})
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

// tlsMode returns one of: auto (default), require, disable
func tlsMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_TLS_MODE")))
	if mode == "require" || mode == "disable" {
		return mode
	}
	return "auto"
}

// isLoopbackAddr returns true if the provided listen address binds only to loopback
func isLoopbackAddr(addr string) bool {
	host := addr
	if strings.HasPrefix(host, ":") {
		// :port means all interfaces
		return false
	}
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	// Unknown hostname: assume non-loopback to be safe
	return false
}

// findTLSPair returns cert and key file paths if configured or found in default mounts
func findTLSPair() (string, string, bool) {
	certFile := strings.TrimSpace(os.Getenv("PS_ENFORCER_TLS_CERT"))
	keyFile := strings.TrimSpace(os.Getenv("PS_ENFORCER_TLS_KEY"))
	if certFile != "" && keyFile != "" {
		return certFile, keyFile, true
	}
	// Auto-detect common mount locations
	defaults := [][2]string{
		{"/tls/server.crt", "/tls/server.key"},
		{"tls/server.crt", "tls/server.key"},
	}
	for _, p := range defaults {
		if _, err1 := os.Stat(p[0]); err1 == nil {
			if _, err2 := os.Stat(p[1]); err2 == nil {
				return p[0], p[1], true
			}
		}
	}
	return "", "", false
}
