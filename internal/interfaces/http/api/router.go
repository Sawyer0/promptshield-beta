package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/usage"
	"github.com/promptshield/promptshield/internal/version"
)

// Router

func NewMux(opt Options) http.Handler {
	r := chi.NewRouter()
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
	r.Use(tenantContextMiddleware)
	r.Use(requestLoggerMiddleware)
	// bytes in/out accounting
	r.Use(captureBytesMiddleware)

	// Multi-tenant validation (tenant routing only - auth handled by frontend)
	if opt.DB != nil {
		r.Use(tenantValidationMiddleware(opt.DB))
	}
	// Note: User authentication handled by frontend - backend only validates tenants

	// Ensure defaults for optional dependencies
	if opt.ConfigStore == nil {
		opt.ConfigStore = NewRuntimeConfigStoreFromEnv()
	}
	// RulepackService is required and must be provided by caller
	if opt.RulepackService == nil {
		panic("RulepackService is required")
	}
	if opt.Events == nil {
		opt.Events = NewEventHub()
	}
	// usage store is optional; if unset, usage endpoint will report zeroes
	// Quota store default: derive from entitlements/env if not provided
	if opt.QuotaStore == nil {
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
			opt.QuotaStore = usage.NewInMemoryQuota(rps, burst)
		}
	}

	// Health & info
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Ready regardless of rulepack presence; no built-in defaults
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	// Version
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version":    version.Version,
			"commit":     version.Commit,
			"build_date": version.BuildDate,
		})
	})

	// Rulepacks
	mountRulepacks(r, opt)

	// Config
	mountConfig(r, opt)

	// Enterprise / platform features (registrations are no-ops when corresponding
	// repositories are nil, so safe to mount unconditionally).
	registerTenantHandlers(r, opt)
	registerAssignmentHandlers(r, opt)
	// registerQuotaHandlers(r, opt) - removed for Security Gateway
	registerAuditHandlers(r, opt)
	registerPolicyHandlers(r, opt)
	registerSystemHandlers(r, opt)
	registerServiceControlHandlers(r, opt)
	registerSettingsHandlers(r, opt)
	registerBusinessMetricsHandlers(r, opt)

	// Security Gateway - no complex usage/quota management needed

	// Admin endpoints (simple token auth)
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Post("/admin/drain", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			if opt.OnDrain != nil {
				go func() { _ = opt.OnDrain(r.Context()) }()
			}
		})
		a.Post("/admin/shutdown", func(w http.ResponseWriter, r *http.Request) {
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
		})
	})

	// License (GET is public; POST rotates license and is admin-protected)
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
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Post("/license", func(w http.ResponseWriter, r *http.Request) {
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
		})
	})

	// Security Gateway decision endpoints (tenant-based + rate limiting)
	r.Group(func(g chi.Router) {
		// Tenant-based rate limiting (auth handled by frontend)
		if opt.QuotaStore != nil {
			g.Use(tenantQuota(opt))
		}
		g.Post("/check", checkHandlerVersioned(opt))
		g.Post("/scan", scanHandler(opt))
	})

	// Observability endpoints (admin-protected)
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))

		a.Get("/events", func(w http.ResponseWriter, r *http.Request) {
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
			// initial
			_, _ = w.Write([]byte("event: ready\ndata: {\"status\":\"ok\"}\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
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
		})

	})

	// Expose usage store via context for handlers that may record usage
	if opt.UsageStore != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := withUsageStore(req.Context(), opt.UsageStore)
			r.ServeHTTP(w, req.WithContext(ctx))
		})
	}
	return r
}

// registerPolicyHandlers registers policy management endpoints
func registerPolicyHandlers(r chi.Router, opt Options) {
	if opt.PolicyService == nil {
		// Policy management not configured - skip registration
		// Debug output
		println("WARNING: PolicyService is nil, skipping policy endpoint registration")
		return
	}

	// Create policy handlers with the policy service
	handlers := NewPolicyHandlers(opt.PolicyService)

	// Register routes
	handlers.RegisterRoutes(r, opt)
	println("INFO: Policy endpoints registered at /admin/policies")
}

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
