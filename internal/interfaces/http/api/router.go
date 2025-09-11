package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
    "sync"
    "fmt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
    rnats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
    pmetrics "github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/usage"
	"github.com/promptshield/promptshield/internal/version"
	"github.com/promptshield/promptshield/internal/pdp"
)

// Router

var startFlushSubscriberOnce sync.Once

// applyStandardMiddleware applies the common middleware chain to a router
func applyStandardMiddleware(r chi.Router, opt Options) {
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(versionHeader("1"))
	// CORS middleware for frontend access
	r.Use(corsMiddleware)
	// Error recovery and structured error handling
	r.Use(errorRecoveryMiddleware)
	// Distributed tracing and header propagation
	if opt.Telemetry != nil {
		r.Use(tracingMiddleware)
	}
	// Request logging and correlation
	r.Use(correlationIDMiddleware)
	// BFF JWT auth: trusts only tokens minted by the frontend service
	r.Use(jwtAuthMiddleware)
	r.Use(tenantContextMiddleware)
	// Attach external PDP client (if configured) for authorization decisions
	r.Use(pdpMiddleware)
	r.Use(requestLoggerMiddleware)
	// Enforce agent policies when tool calls are declared
	r.Use(agentEnforcementMiddleware(opt))
	// bytes in/out accounting
	r.Use(captureBytesMiddleware)
}

// validateAndSetDefaults ensures required options are set and provides defaults
func validateAndSetDefaults(opt *Options) {
	// Required dependencies
	if opt.RulepackService == nil {
		panic("RulepackService is required")
	}
	
	// Optional dependencies with defaults
	if opt.ConfigStore == nil {
		opt.ConfigStore = NewRuntimeConfigStoreFromEnv()
	}
	if opt.Events == nil {
		opt.Events = NewEventHub()
	}
	
	// Quota store from environment if not provided
	if opt.QuotaStore == nil {
		opt.QuotaStore = createQuotaStoreFromEnv()
	}
}

// createQuotaStoreFromEnv creates a quota store from environment variables
func createQuotaStoreFromEnv() usage.QuotaStore {
	var rps float64
	var burst int
	
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			rps = f
		}
	}
	
	if rps > 0 {
		if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS_BURST")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				burst = n
			}
		}
		return usage.NewInMemoryQuota(rps, burst)
	}
	
	return nil
}

// registerAllHandlers registers all API endpoints in a logical order
func registerAllHandlers(r chi.Router, opt Options) {
	// Core functionality
	mountRulepacks(r, opt)
	mountConfig(r, opt)
	
	// Tool and agent functionality
	r.Post("/api/tools/exec", toolExecHandler(opt))
	registerToolHandlers(r, opt)
	registerAgentHandlers(r, opt)
	registerPresetHandlers(r, opt)
	
	// Enterprise features (safe to mount even if repositories are nil)
	registerTenantHandlers(r, opt)
	registerAssignmentHandlers(r, opt)
	registerAuditHandlers(r, opt)
	registerUserHandlers(r, opt)
	
	// System and monitoring
	registerSystemHandlers(r, opt)
	registerSettingsHandlers(r, opt)
	registerBusinessMetricsHandlers(r, opt)
}

// registerStandardEndpoints registers health, readiness, and version endpoints
func registerStandardEndpoints(r chi.Router) {
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version":    version.Version,
			"commit":     version.Commit,
			"build_date": version.BuildDate,
		})
	})
}

// registerAdminEndpoints registers administrative endpoints with token auth

// pdpEpochHandler updates the PDP cache epoch at runtime
func pdpEpochHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Epoch string `json:"epoch"` }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.TrimSpace(body.Epoch) == "" { writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "epoch is required", nil, r); return }
		pdp.SetPolicyEpoch(body.Epoch)
		writeJSON(w, http.StatusNoContent, nil, r)
	}
}

// pdpReloadHandler rebuilds the PDP client from current env (refresh mode/endpoint/policy)
func pdpReloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Build a new client and swap it in
		c := buildPDPClientForAdmin()
		if c == nil { writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to initialize PDP", nil, r); return }
		pdpClient = c
		// Auto-bump epoch
		e := fmt.Sprintf("%d", time.Now().UnixNano())
		pdp.SetPolicyEpoch(e)
		writeJSON(w, http.StatusNoContent, nil, r)
	}
}
func registerAdminEndpoints(r chi.Router, opt Options) {
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		
		// System control
		a.Post("/admin/drain", drainHandler(opt))
		a.Post("/admin/shutdown", shutdownHandler(opt))
		a.Post("/admin/tool-policies/flush", toolPolicyFlushHandler(opt))
		// PDP admin
		a.Post("/admin/pdp/epoch", pdpEpochHandler())
		a.Post("/admin/pdp/reload", pdpReloadHandler())
	})
}

// drainHandler handles graceful drain requests
func drainHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		if opt.OnDrain != nil {
			go func() { _ = opt.OnDrain(r.Context()) }()
		}
	}
}

// shutdownHandler handles graceful shutdown requests
func shutdownHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		if opt.OnShutdown != nil {
			// optional delay in seconds
			delay := 0 * time.Second
			if v := r.URL.Query().Get("delay"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					delay = time.Duration(n) * time.Second
				}
			}
			go func() { _ = opt.OnShutdown(r.Context(), delay) }()
		}
	}
}

// toolPolicyFlushHandler handles tool policy cache flush requests
func toolPolicyFlushHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Optional tenant-scoped flush (JSON body: {"tenant_id":"..."})
		var req struct{ TenantID string `json:"tenant_id"` }
		_ = json.NewDecoder(r.Body).Decode(&req)
		
		// Immediate local flush
		tenantID := strings.TrimSpace(req.TenantID)
		if tenantID != "" {
			flushToolPolicyCacheTenant(tenantID)
		} else {
			flushToolPolicyCache()
		}
		
		// Bump global epoch for cluster-wide invalidation
		updateToolPolicyEpoch(r, opt)
		
		// Publish push invalidation event via NATS (if configured)
		publishToolPolicyFlush(tenantID)
		
		w.WriteHeader(http.StatusNoContent)
	}
}

// updateToolPolicyEpoch increments the tool policy epoch in settings
func updateToolPolicyEpoch(r *http.Request, opt Options) {
	if opt.SettingsRepository == nil {
		return
	}
	
	s, err := opt.SettingsRepository.Get(r.Context())
	if err != nil || s == nil {
		return
	}
	
	var raw map[string]any
	_ = json.NewDecoder(strings.NewReader(string(s.Settings))).Decode(&raw)
	if raw == nil {
		raw = map[string]any{}
	}
	
	if v, ok := raw["tool_policy_epoch"].(float64); ok {
		raw["tool_policy_epoch"] = int64(v) + 1
	} else {
		raw["tool_policy_epoch"] = 1
	}
	
	raw["updated_by"] = getUserFromContext(r.Context())
	raw["updated_at"] = time.Now().UTC()
	_, _ = opt.SettingsRepository.Update(r.Context(), raw)
}

// publishToolPolicyFlush publishes a tool policy flush event via NATS
func publishToolPolicyFlush(tenantID string) {
	addr := strings.TrimSpace(os.Getenv("PS_NATS_URL"))
	if addr == "" {
		return
	}
	
	ev := rnats.ToolPolicyFlush{
		TenantID: tenantID,
		At:       time.Now().UTC(),
		Reason:   "admin_flush",
	}
	
	_ = rnats.PublishToolPolicyFlush(addr, ev)
	
	scope := "global"
	if ev.TenantID != "" {
		scope = "tenant"
	}
	
	logger := slog.With("component", "policy-flush-publisher")
	logger.Info("published tool policy flush", "scope", scope, "tenant_id", ev.TenantID)
}

// registerLicenseEndpoints registers license management endpoints
func registerLicenseEndpoints(r chi.Router, opt Options) {
	// Public license info
	r.Get("/license", func(w http.ResponseWriter, r *http.Request) {
		l := license.Info()
		ent, _ := license.Entitlement()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"org":          l.Organization,
			"tier":         l.Tier,
			"expires_at":   l.ExpiresAt,
			"licensed":     license.IsLicensed(),
			"entitlements": ent,
		})
	})
	
	// Admin-only license rotation
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Post("/license", licenseUpdateHandler())
	})
}

// licenseUpdateHandler handles license key updates
func licenseUpdateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.FormValue("key")
		if key == "" {
			var body struct {
				Key string `json:"key"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			key = body.Key
		}
		if key == "" {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing key", nil)
			return
		}
		_ = os.Setenv("PROMPTSHIELD_LICENSE_KEY", key)
		w.WriteHeader(http.StatusNoContent)
	}
}

// registerSecurityEndpoints registers security gateway decision endpoints
func registerSecurityEndpoints(r chi.Router, opt Options) {
	r.Group(func(g chi.Router) {
		// Tenant-based rate limiting (auth handled by frontend)
		if opt.QuotaStore != nil {
			g.Use(tenantQuota(opt))
		}
		
		g.Post("/check", checkHandlerVersioned(opt))
	})
}


// registerObservabilityEndpoints registers observability and monitoring endpoints
func registerObservabilityEndpoints(r chi.Router, opt Options) {
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Get("/events", eventsStreamHandler(opt))
	})
}

// eventsStreamHandler handles server-sent events streaming
func eventsStreamHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var flusher http.Flusher
		if f, ok := w.(http.Flusher); ok {
			flusher = f
		}
		
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		
		ch := opt.Events.Subscribe(r.Context().Done())
		defer opt.Events.Unsubscribe(ch)
		
		// Optional type filtering
		var filters map[string]struct{}
		if v := strings.TrimSpace(r.URL.Query().Get("types")); v != "" {
			filters = make(map[string]struct{})
			for _, t := range strings.Split(v, ",") {
				if t = strings.TrimSpace(t); t != "" {
					filters[t] = struct{}{}
				}
			}
		}
		
		// Send initial ready event
		_, _ = w.Write([]byte("event: ready\ndata: {\"status\":\"ok\"}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		
		// Stream events
		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case ev := <-ch:
				if ev == nil {
					return
				}
				if len(filters) > 0 {
					if _, ok := filters[ev.Type]; !ok {
						continue
					}
				}
				_, _ = w.Write(ev.SSE())
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
	}
}

func NewMux(opt Options) http.Handler {
	r := chi.NewRouter()
	
	// Reset PDP client between mux instances when PDP endpoint is not configured.
	// This improves test isolation so a prior test enabling PDP does not affect others.
	if strings.TrimSpace(os.Getenv("PS_PDP_ENDPOINT")) == "" && strings.ToLower(strings.TrimSpace(os.Getenv("PS_PDP_MODE"))) != "inprocess" {
		pdpClient = nil
		pdpOnce = sync.Once{}
	}
	
	// Apply standard middleware chain
	applyStandardMiddleware(r, opt)
	
	// Multi-tenant validation (tenant routing only - auth handled by frontend)
	if opt.DB != nil {
		r.Use(tenantValidationMiddleware(opt.DB))
	}
	// Note: User authentication handled by frontend - backend only validates tenants

	// Validate and set option defaults
	validateAndSetDefaults(&opt)

	// Standard endpoints
	registerStandardEndpoints(r)

	// Register all API endpoints
	registerAllHandlers(r, opt)

	// Security Gateway - no complex usage/quota management needed

	// Start Redis pub/sub subscriber for tool policy flush if configured
    if addr := strings.TrimSpace(os.Getenv("PS_NATS_URL")); addr != "" {
        _ = addr // silence unused in certain builds
        startFlushSubscriberOnce.Do(func(){
            slog.With("component","policy-flush-subscriber").Info("starting subscriber", "addr", addr)
            pmetrics.PolicyFlushSubscriberUp.Set(1)
            _ = rnats.StartToolPolicyFlushSubscriber(addr, func(ev rnats.ToolPolicyFlush){
                // Metrics: latency and counters
                scope := "global"; if ev.TenantID != "" { scope = "tenant" }
                if !ev.At.IsZero() {
                    if d := time.Since(ev.At); d >= 0 {
                        pmetrics.PolicyFlushLatencySeconds.WithLabelValues(scope).Observe(d.Seconds())
                    }
                }
                pmetrics.PolicyFlushEventsTotal.WithLabelValues("subscriber", scope).Inc()
                // Log
                logger := slog.With("component", "policy-flush-subscriber")
                logger.Info("received tool policy flush", "scope", scope, "tenant_id", ev.TenantID)
                pmetrics.PolicyFlushLastReceiveUnixSeconds.Set(float64(time.Now().Unix()))
                // Apply flush
                if ev.TenantID != "" { flushToolPolicyCacheTenant(ev.TenantID) } else { flushToolPolicyCache() }
            })
        })
    }
    if strings.TrimSpace(os.Getenv("PS_NATS_URL")) == "" {
        slog.With("component","policy-flush-subscriber").Info("subscriber disabled; PS_NATS_URL empty")
        pmetrics.PolicyFlushSubscriberUp.Set(0)
    }

	// Admin endpoints
	registerAdminEndpoints(r, opt)

	// License endpoints
	registerLicenseEndpoints(r, opt)

	// Security Gateway endpoints
	registerSecurityEndpoints(r, opt)

	// Observability endpoints
	registerObservabilityEndpoints(r, opt)

	// Debug endpoints for authentication troubleshooting (opt-in via PS_ENABLE_DEBUG_ENDPOINTS)
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PS_ENABLE_DEBUG_ENDPOINTS")), "true") {
		registerDebugEndpoints(r, opt)
	}

	// Expose usage store via context for handlers that may record usage
	if opt.UsageStore != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := withUsageStore(req.Context(), opt.UsageStore)
			r.ServeHTTP(w, req.WithContext(ctx))
		})
	}
	return r
}

// (removed) registerPolicyHandlers — policy endpoints are registered elsewhere

// registerServiceControlHandlers registers service control endpoints
func registerServiceControlHandlers(r chi.Router, opt Options) {
	if opt.DB == nil {
		// Service control requires database connection
		println("WARNING: Service control disabled - no database connection")
		return
	}

	// Create mock service manager for now (replace with real implementation)
	serviceManager := NewMockServiceManager()

	// Create service control handlers
	handlers := NewServiceControlHandlers(opt.DB, serviceManager, opt.Events)

	// Register routes
	handlers.RegisterServiceRoutes(r, opt)
	println("INFO: Service control endpoints registered at /api/v1/services")
}
