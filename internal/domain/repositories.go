package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TenantRepository defines operations for tenant management
type TenantRepository interface {
	Create(ctx context.Context, tenant *Tenant) error
	Get(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetByName(ctx context.Context, name string) (*Tenant, error)
	List(ctx context.Context, offset, limit int) ([]*Tenant, int, error)
	Update(ctx context.Context, tenant *Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
	// External organization mapping helpers (e.g., Clerk org -> tenant)
	GetByExternalOrg(ctx context.Context, provider string, externalOrgID string) (*Tenant, error)
	LinkExternalOrg(ctx context.Context, provider string, externalOrgID string, tenantID uuid.UUID) error
}

// AuditRepository defines operations for audit trail management
type AuditRepository interface {
	Create(ctx context.Context, entry *AuditEntry) error
	Get(ctx context.Context, id uuid.UUID) (*AuditEntry, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*AuditEntry, int, error)
	ListByObject(ctx context.Context, objectType string, objectID uuid.UUID, offset, limit int) ([]*AuditEntry, int, error)
	ListByAction(ctx context.Context, action string, offset, limit int) ([]*AuditEntry, int, error)
}

// RulepackAssignmentRepository defines operations for rulepack assignments
type RulepackAssignmentRepository interface {
	Create(ctx context.Context, assignment *RulepackAssignment) error
	Get(ctx context.Context, id uuid.UUID) (*RulepackAssignment, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*RulepackAssignment, error)
	ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*RulepackAssignment, error)
	ListByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]*RulepackAssignment, error)
	Update(ctx context.Context, assignment *RulepackAssignment) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByTenantAndPolicy(ctx context.Context, tenantID, policyID uuid.UUID) error
}

// Security Gateway - no provider key management needed

// Security Gateway uses simple environment-based rate limiting
// No complex per-tenant quota management needed

// APITokenRepository defines operations for API token management
type APITokenRepository interface {
	Create(ctx context.Context, token *APIToken) error
	Get(ctx context.Context, id uuid.UUID) (*APIToken, error)
	GetByHash(ctx context.Context, hashedToken string) (*APIToken, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*APIToken, error)
	Update(ctx context.Context, token *APIToken) error
	Delete(ctx context.Context, id uuid.UUID) error
	Rotate(ctx context.Context, id uuid.UUID) (string, error)
}

// SettingsRepository defines operations for platform settings management
type SettingsRepository interface {
	Get(ctx context.Context) (*PlatformSettings, error)
	Update(ctx context.Context, settings interface{}) (*PlatformSettings, error)
	GetHistory(ctx context.Context, limit int, offset int) ([]*PlatformSettings, int, error)
	Delete(ctx context.Context) error
	Backup(ctx context.Context) ([]byte, error)
	Restore(ctx context.Context, backupData []byte) error
	ValidateConnection(ctx context.Context) error
}

// PlatformSettings represents the platform configuration stored in database
type PlatformSettings struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Settings  []byte    `json:"settings" db:"settings"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	UpdatedBy string    `json:"updated_by" db:"updated_by"`
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed                    bool
	RequestsPerMinuteRemaining int
	RequestsPerHourRemaining   int
	RetryAfter                 time.Duration
}
