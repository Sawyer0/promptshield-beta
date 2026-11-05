package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/promptshield/promptshield/internal/version"
)

// Router

func NewMux(opt Options) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(versionHeader("1"))
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

	opt = withDefaults(opt)

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

	// Security Gateway - no complex usage/quota management needed

	mountAdminLifecycle(r, opt)

	// Security Gateway decision endpoints (simple token auth + basic rate limiting)
	r.Group(func(g chi.Router) {
		// Simple token-based user auth
		g.Use(userAuth(opt))
		// Basic per-tenant rate limiting
		if opt.QuotaStore != nil {
			g.Use(tenantQuota(opt))
		}
		g.Post("/check", checkHandlerVersioned(opt))
		g.Post("/scan", scanHandler(opt))
	})

	mountJobs(r, opt)

	// Observability endpoints (admin-protected)
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		mountDecisions(a, opt)
		mountActiveRules(a, opt)
		a.Get("/stats", statsHandler)
		a.Get("/events", eventStreamHandler(opt))
		a.Get("/usage", usageHandler)
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
