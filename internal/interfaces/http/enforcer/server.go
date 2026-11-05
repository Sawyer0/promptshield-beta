package enforcerhttp

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/promptshield/promptshield/internal/application/decisions"
	"github.com/promptshield/promptshield/internal/application/rulesets"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	"github.com/promptshield/promptshield/internal/interfaces/http/dashboard"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTP server for the PromptShield enforcer.
// Exposes:
// - GET /healthz: liveness probe
// - GET /readyz: readiness probe with rule validation
// - GET /metrics: Prometheus metrics
// - POST /check: enforcement decision endpoint (also /check/* for ext_authz)
// - /v1/*: versioned Gateway API (see internal/interfaces/http/api)
// - GET /dashboard/: operator dashboard (disable with PS_DASHBOARD_DISABLED=true)

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

func init() {
	// Best-effort registration; ignore panics on duplicate in tests
	prometheus.MustRegister(enforcerRequests, enforcerDecisions, enforcerReqDuration, policyBypass)
}

// NewMux constructs the enforcer HTTP handler using options from the environment.
func NewMux() http.Handler {
	return NewMuxWithOptions(getAPIOptions())
}

// NewMuxWithOptions constructs the HTTP handler mux with injectable API options.
func NewMuxWithOptions(apiOpt api.Options) http.Handler {
	r := chi.NewRouter()
	// Initialize database pool if configured
	var dbPool *pg.Pool
	dbHealthy := true
	if dsn := os.Getenv("PS_PG_DSN"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		pool, err := pg.NewPool(ctx, dsn)
		if err == nil {
			pingCtx, cancelPing := context.WithTimeout(ctx, 2*time.Second)
			if _, err := pool.Raw().Exec(pingCtx, "SELECT 1"); err != nil {
				dbHealthy = false
				pool.Close()
				pool = nil
			} else {
				dbPool = pool
			}
			cancelPing()
		} else {
			dbHealthy = false
		}
		if !dbHealthy {
			// Switch to observe mode to fail-open.
			os.Setenv("PS_ENFORCER_MODE", "observe")
			logger := slog.With("component", "enforcer-http")
			logger.Warn("Database unreachable at startup; entering OBSERVE fail-open mode", "dsn", dsn)
		}
	}

	// If API options don't have services configured, initialize them with DB
	if apiOpt.RulepackService == nil {
		apiOpt = getAPIOptionsWithDB(dbPool)
	}

	// Middlewares: request id, real ip, recoverer, timeout (defense in depth; keep short)
	if !fastMode() {
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(middleware.Recoverer)
		r.Use(api.RequestTimeout(10 * time.Second)) // /v1/events is exempt
	}
	// Active RulePacks, shared by /check, /v1/check and the dashboard, and
	// reloadable through POST /v1/rules/reload.
	if apiOpt.ActiveRules == nil {
		set, err := rulesets.NewSet(os.Getenv("PS_ENFORCER_RULEPACK"))
		if err != nil {
			slog.Error("failed to load rulepack", "path", os.Getenv("PS_ENFORCER_RULEPACK"), "error", err)
		}
		apiOpt.ActiveRules = set
	}
	// Share one event hub and decision log between /check and the v1 API.
	if apiOpt.Events == nil {
		apiOpt.Events = api.NewEventHub()
	}
	if apiOpt.Decisions == nil {
		apiOpt.Decisions = decisions.NewLog(decisions.DefaultCapacity)
	}
	var maxStreamBytes int64
	if v := os.Getenv("PS_ENFORCER_MAX_STREAM_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxStreamBytes = n
		}
	}
	scanners := newScannerPool(apiOpt.ActiveRules, maxStreamBytes)
	hasRules := func() bool { return len(apiOpt.ActiveRules.Packs()) > 0 }

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// Readiness gate: health plus rulepack/scanner ready
	requireAtStartup := envBool("PS_REQUIRE_RULEPACK_AT_STARTUP")
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ready := dbHealthy && (hasRules() || !requireAtStartup)
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

	// Operator dashboard: static assets only; its API calls need the admin token.
	if !envBool("PS_DASHBOARD_DISABLED") {
		r.Mount("/dashboard", http.StripPrefix("/dashboard", dashboard.Handler()))
	}

	// Mount v1 API
	r.Mount("/v1", api.NewMux(apiOpt))

	// POST /check (and /check/* for Envoy ext_authz path_prefix forwarding)
	check := newCheckHandler(checkHandlerConfig{
		scanners:     scanners,
		hasRules:     hasRules,
		requireRules: requireAtStartup,
		record:       func(rec decisions.Record) { api.RecordDecision(apiOpt, rec) },
	})
	r.Route("/check", func(cr chi.Router) {
		cr.HandleFunc("/", check)
		cr.HandleFunc("/*", check)
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
