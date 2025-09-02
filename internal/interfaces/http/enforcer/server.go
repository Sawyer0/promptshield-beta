package enforcerhttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/audit"
	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/infrastructure/planstate"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	"github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/security/paths"
	semopenai "github.com/promptshield/promptshield/internal/semantic/openai"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
	"github.com/promptshield/promptshield/internal/usage"
	redis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTP server for the PromptShield enforcer.
// Exposes:
// - GET /healthz: liveness probe
// - GET /readyz: readiness probe with rule validation
// - GET /metrics: Prometheus metrics
// - POST /check: enforcement decision endpoint

// NewMux constructs the HTTP handler mux for the enforcer.

// Optional performance toggles (env-driven)
func metricsEnabled() bool {
	return metrics.Enabled()
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

func NewMux() http.Handler { // backward-compatible wrapper
	return NewMuxWithOptions(getAPIOptions())
}

// getAPIOptions constructs API options from environment variables
func getAPIOptions() api.Options {
	return getAPIOptionsWithDB(nil)
}

// getAPIOptionsWithDB constructs API options with optional database pool
func getAPIOptionsWithDB(dbPool *pg.Pool) api.Options {
	adminToken := os.Getenv("PS_ENFORCER_ADMIN_TOKEN")

	// Optional insecure mode toggle for local dev / demos
	allowInsecure := false
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_ALLOW_INSECURE_ADMIN"))); v != "" {
		switch v {
		case "1", "true", "yes", "on":
			allowInsecure = true
		}
	}

	// Initialize services with database if available
	var rulepackService *services.RulepackService
	var policyService contracts.PolicyService
	var tenantRepo domain.TenantRepository
	var assignmentRepo domain.RulepackAssignmentRepository
	var auditRepo domain.AuditRepository

	if dbPool != nil {
		// Use PostgreSQL repositories for enterprise features
		rulepackRepo := pg.RulepackRepo(dbPool)
		rulepackService = services.RulepackServiceCstor(rulepackRepo, nil)

		// Initialize enterprise repositories
		tenantRepo = pg.TenantRepo(dbPool)
		assignmentRepo = pg.RulepackAssignmentRepo(dbPool)
		auditRepo = pg.AuditRepo(dbPool)

		// Use in-memory policy service for fast enforcement
		// Policies are persisted in frontend and synced via API
		policyService = initializePolicyService()
	} else {
		// Use in-memory implementations for high-performance enforcement
		// Policies are persisted in frontend database and synced via API

		// Create in-memory rulepack repository
		rulepackRepo := memory.NewRulepackRepository()
		rulepackService = services.RulepackServiceCstor(rulepackRepo, nil)

		// Use in-memory policy repository
		policyService = initializePolicyService()

		// Enterprise repositories will be nil (endpoints will return NOT_IMPLEMENTED)
	}

	// Create scanner manager for event-driven real-time enforcement
	scannerManager := NewScannerManagerWithRulepackService(rulepackService, dbPool)

	// Build API options with database-backed repositories
	options := api.Options{
		AdminToken:         adminToken,
		AllowInsecureAdmin: allowInsecure,
		PolicyService:      policyService,
		RulepackService:    rulepackService,
		ScannerManager:     scannerManager,
	}

	// Wire enterprise repositories when database is available
	if tenantRepo != nil {
		options.TenantRepository = tenantRepo
	}
	if assignmentRepo != nil {
		options.AssignmentRepository = assignmentRepo
	}
	if auditRepo != nil {
		options.AuditRepository = auditRepo
	}
	if dbPool != nil {
		options.SettingsRepository = pg.NewSettingsRepository(dbPool)
		options.DB = dbPool
	}

	return options
}

// initializePolicyService creates and configures the policy management service
func initializePolicyService() contracts.PolicyService {
	// Initialize policy dependencies with bootstrap
	policyDeps := bootstrap.InitializePolicyDependencies(
		nil, // ruleCompiler - will be nil for now, add later when integrating with scanner
		nil, // scanEngine - will be nil for now, add later when integrating with scanner
		nil, // auditLogger - will be nil for now, add later when integrating with audit
	)

	return policyDeps.Service
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
		sc := scanner.ScanEngineCstor(0)
		if maxStreamBytes > 0 {
			sc.SetMaxStreamBytes(maxStreamBytes)
		}
		sc.SetQuarantineOnTimeout(true)
		sc.SetQuarantineOnError(true)
		if len(preloadPacks) > 0 {
			sc.LoadRulePacks(preloadPacks)
		}

		// Initialize semantic analyzer if enabled
		if os.Getenv("PS_SEMANTIC_ENABLED") == "true" {
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey != "" {
				analyzer := semopenai.New(semopenai.Options{
					APIKey:            apiKey,
					MaxConcurrency:    2,
					CacheSize:         1000,
					CacheTTL:          15 * time.Minute,
					RequestsPerSecond: 10,
					BurstSize:         20,
				})
				sc.SetSemanticAnalyzer(analyzer)
				if logger := slog.With("component", "semantic"); logger != nil {
					logger.Info("OpenAI semantic analyzer initialized")
				}
			}
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

	// Back-compat aliases for gateway tests expecting root-level paths
	r.Group(func(g chi.Router) {
		// Minimal adapter to forward to API mux for /check and /scan at root
		am := api.NewMux(apiOpt)
		g.Post("/check", func(w http.ResponseWriter, r *http.Request) {
			// Forward to API mux native path
			r = r.Clone(r.Context())
			if r.Header.Get("X-PS-Tenant-ID") == "" {
				r.Header.Set("X-PS-Tenant-ID", "00000000-0000-0000-0000-000000000001")
			}
			r.URL.Path = "/check"
			am.ServeHTTP(w, r)
		})
		g.Post("/scan", func(w http.ResponseWriter, r *http.Request) {
			r = r.Clone(r.Context())
			if r.Header.Get("X-PS-Tenant-ID") == "" {
				r.Header.Set("X-PS-Tenant-ID", "00000000-0000-0000-0000-000000000001")
			}
			r.URL.Path = "/scan"
			am.ServeHTTP(w, r)
		})
	})

	// Prepare scanner pool and optionally preload rulepacks to reduce per-request overhead
	{
		var maxStreamBytes int64
		if v := os.Getenv("PS_ENFORCER_MAX_STREAM_BYTES"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				maxStreamBytes = n
			}
		}
		scannerPool.New = func() any {
			sc := scanner.ScanEngineCstor(0)
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
	// Initialize database pool if configured
	var dbPool *pg.Pool
	if dsn := os.Getenv("PS_PG_DSN"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if pool, err := pg.NewPool(ctx, dsn); err == nil {
			pingCtx, cancelPing := context.WithTimeout(ctx, 2*time.Second)
			if _, err := pool.Raw().Exec(pingCtx, "SELECT 1"); err == nil {
				dbPool = pool
			} else {
				pool.Close()
			}
			cancelPing()
		}
	}
	// Inject shutdown hooks into API mux with database pool
	options := getAPIOptionsWithDB(dbPool)
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

	// Wire Redis PlanState for agent plan/lane TTL
	if options.PlanState == nil {
		if addr := strings.TrimSpace(os.Getenv("PS_PLANSTATE_REDIS_ADDR")); addr != "" {
			prefix := strings.TrimSpace(os.Getenv("PS_PLANSTATE_PREFIX"))
			if prefix == "" {
				prefix = "ps"
			}
			rdb := redis.NewClient(&redis.Options{Addr: addr})
			options.PlanState = planstate.NewRedisPlanState(rdb, prefix)
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
		if dbPool != nil {
			dbPool.Close()
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
			logger := slog.With("component", "enforcer-http")
			if err := paths.ValidateCAFilePath(clientCA); err != nil {
				logger.Error("invalid client CA file path", "error", err)
			} else if caPEM, err := os.ReadFile(clientCA); err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caPEM) {
					tlsCfg.ClientCAs = pool
					tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
				} else {
					logger.Error("failed to parse client CA certificate")
				}
			} else {
				logger.Error("failed to read client CA file", "error", err)
			}
		}
		srv.TLSConfig = tlsCfg
		go func() {
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				logger := slog.With("component", "enforcer-http")
				logger.Error("enforcer https server error", "error", err)
			}
		}()
	}

	switch mode {
	case "disable":
		// Explicit insecure mode
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger := slog.With("component", "enforcer-http")
				logger.Error("enforcer http server error", "error", err)
			}
		}()
		return srv
	case "require":
		if !havePair {
			logger := slog.With("component", "enforcer-http")
			logger.Error("TLS required but no certificate configured", "address", addr)
			os.Exit(1)
		}
		startHTTPS(certFile, keyFile)
		return srv
	default: // auto
		if nonLoop {
			if !havePair {
				logger := slog.With("component", "enforcer-http")
				logger.Error("Refusing to listen on non-loopback address without TLS", "address", addr)
				os.Exit(1)
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
					logger := slog.With("component", "enforcer-http")
					logger.Error("enforcer http server error", "error", err)
				}
			}()
		}
	}
	return srv
}

func generateRequestID() string {
	return uuid.NewString()
}

// auditAdapter adapts internal/audit.Logger to contracts.AuditLogger
type auditAdapter struct{ inner audit.Logger }

func (a auditAdapter) LogWithContext(ctx context.Context, ev types.AuditEvent) error {
	return a.inner.Log(audit.Event{Type: ev.Action, Data: ev.Metadata, Hash: "", PrevHash: ""})
}

func (a auditAdapter) Flush() error {
	return nil
}

func (a auditAdapter) Close() error {
	return nil
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
