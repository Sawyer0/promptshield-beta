package api

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/jobs"
	"github.com/promptshield/promptshield/internal/usage"
)

// Options configures the API mux.
type Options struct {
	AdminToken         string
	AllowInsecureAdmin bool
	ConfigStore        *RuntimeConfigStore
	RulepackService    *services.RulepackService
	UsageStore         usage.UsageStore
	// AuditLogger, when set, receives durable audit trail events
	AuditLogger AuditLogger
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
	
	// Enterprise repositories for tenant management
	TenantRepository     domain.TenantRepository
	AssignmentRepository domain.PolicyAssignmentRepository
	AuditRepository      domain.AuditRepository
	ProviderKeyStore     domain.ProviderKeyRepository
	QuotaRepository      domain.QuotaRepository
	
	
	// Provider API keys management (deprecated - use ProviderKeyStore)
	ProviderKeys map[string]string // provider -> encrypted key
}


// AuditLogger is a narrow interface to avoid importing audit package here.
type AuditLogger interface {
	Log(event AuditEvent) error
}

// AuditEvent mirrors internal/audit.Event without importing the package to avoid cycle.
type AuditEvent struct {
	Timestamp time.Time      `json:"timestamp"`
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
	Hash      string         `json:"hash"`
	PrevHash  string         `json:"prev_hash"`
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Deprecated: Use writeErrorJSON instead
func writeError(w http.ResponseWriter, status int, code, msg string, details map[string]any) {
	writeErrorJSON(w, status, code, msg, details, nil)
}

// QuotaRepository defines operations for quota management and rate limiting
type QuotaRepository interface {
	Create(ctx context.Context, quota *domain.Quota) error
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.Quota, error)
	Update(ctx context.Context, quota *domain.Quota) error
	Delete(ctx context.Context, tenantID uuid.UUID) error
	CheckRateLimit(ctx context.Context, tenantID uuid.UUID) (*domain.RateLimitResult, error)
	IncrementUsage(ctx context.Context, tenantID uuid.UUID, tokens int64) error
}

// APITokenRepository defines operations for API token management
type APITokenRepository interface {
	Create(ctx context.Context, token *domain.APIToken) error
	Get(ctx context.Context, id uuid.UUID) (*domain.APIToken, error)
	GetByHash(ctx context.Context, hashedToken string) (*domain.APIToken, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.APIToken, error)
	Update(ctx context.Context, token *domain.APIToken) error
	Delete(ctx context.Context, id uuid.UUID) error
	Rotate(ctx context.Context, id uuid.UUID) (string, error)
}

func versionHeader(v string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-PS-API-Version", v)
			next.ServeHTTP(w, r)
		})
	}
}
