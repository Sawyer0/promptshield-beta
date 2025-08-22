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

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed                     bool
	RequestsPerMinuteRemaining  int
	RequestsPerHourRemaining    int
	RetryAfter                  time.Duration
}