package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/promptshield/promptshield/internal/jobs"
	"github.com/promptshield/promptshield/internal/usage"
)

// Options configures the API mux.
type Options struct {
	AdminToken         string
	AllowInsecureAdmin bool
	ConfigStore        *RuntimeConfigStore
	RulepackManager    *RulepackManager
	UsageStore         usage.UsageStore
	// JobManager handles asynchronous job processing. When nil, a default manager is created.
	JobManager *jobs.Manager
	// Events provides a simple in-memory broadcaster for SSE and hooks.
	// When nil, a default hub is created by the router.
	Events *EventHub
	// OnDrain is called when /v1/admin/drain is invoked. Optional.
	OnDrain func(ctx context.Context) error
	// OnShutdown is called when /v1/admin/shutdown is invoked. Optional.
	// Implementations should gracefully stop the server after the given delay.
	OnShutdown func(ctx context.Context, delay time.Duration) error
	// QuotaStore enables per-tenant rate limiting when set.
	QuotaStore usage.QuotaStore
	// OIDC enables JWT validation when configured.
	OIDC OIDCConfig
	// oidcVerifier is initialized when OIDC is enabled; stored as any to avoid import here.
	oidcVerifier any
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, msg string, details map[string]any) {
	w.Header().Set("content-type", "application/json")
	w.Header().Set("X-PS-API-Version", "1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{Code: code, Message: msg, Details: details})
}

func versionHeader(v string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-PS-API-Version", v)
			next.ServeHTTP(w, r)
		})
	}
}
