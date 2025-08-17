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
}

// AuditRepository defines operations for audit trail management
type AuditRepository interface {
	Create(ctx context.Context, entry *AuditEntry) error
	Get(ctx context.Context, id uuid.UUID) (*AuditEntry, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*AuditEntry, int, error)
	ListByObject(ctx context.Context, objectType string, objectID uuid.UUID, offset, limit int) ([]*AuditEntry, int, error)
	ListByAction(ctx context.Context, action string, offset, limit int) ([]*AuditEntry, int, error)
}

// PolicyAssignmentRepository defines operations for policy assignments
type PolicyAssignmentRepository interface {
	Create(ctx context.Context, assignment *PolicyAssignment) error
	Get(ctx context.Context, id uuid.UUID) (*PolicyAssignment, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*PolicyAssignment, error)
	ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*PolicyAssignment, error)
	ListByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]*PolicyAssignment, error)
	Update(ctx context.Context, assignment *PolicyAssignment) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByTenantAndPolicy(ctx context.Context, tenantID, policyID uuid.UUID) error
}

// ProviderKeyRepository defines operations for managing provider API keys
type ProviderKeyRepository interface {
	Create(ctx context.Context, key *ProviderKey) error
	Get(ctx context.Context, id uuid.UUID) (*ProviderKey, error)
	GetByAlias(ctx context.Context, tenantID uuid.UUID, provider string, alias string) (*ProviderKey, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ProviderKey, error)
	ListByProvider(ctx context.Context, tenantID uuid.UUID, provider string) ([]*ProviderKey, error)
	Update(ctx context.Context, key *ProviderKey) error
	Delete(ctx context.Context, id uuid.UUID) error
	Rotate(ctx context.Context, id uuid.UUID, newEncryptedKey string) error
	SetDefault(ctx context.Context, tenantID uuid.UUID, provider string, keyID uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// QuotaRepository defines operations for quota management and rate limiting
type QuotaRepository interface {
	Create(ctx context.Context, quota *Quota) error
	Get(ctx context.Context, tenantID uuid.UUID) (*Quota, error)
	Update(ctx context.Context, quota *Quota) error
	Delete(ctx context.Context, tenantID uuid.UUID) error
	CheckRateLimit(ctx context.Context, tenantID uuid.UUID) (*RateLimitResult, error)
	IncrementUsage(ctx context.Context, tenantID uuid.UUID, tokens int64) error
}

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

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed                     bool
	RequestsPerMinuteRemaining  int
	RequestsPerHourRemaining    int
	RetryAfter                  time.Duration
}