package contracts

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// TenantRepository defines operations for tenant management
type TenantRepository interface {
	Create(ctx context.Context, tenant *types.Tenant) error
	Get(ctx context.Context, id uuid.UUID) (*types.Tenant, error)
	GetByName(ctx context.Context, name string) (*types.Tenant, error)
	List(ctx context.Context, offset, limit int) ([]*types.Tenant, int, error)
	Update(ctx context.Context, tenant *types.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AuditRepository defines operations for audit trail management
type AuditRepository interface {
	Create(ctx context.Context, entry *types.AuditEntry) error
	Get(ctx context.Context, id uuid.UUID) (*types.AuditEntry, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*types.AuditEntry, int, error)
	ListByObject(ctx context.Context, objectType string, objectID uuid.UUID, offset, limit int) ([]*types.AuditEntry, int, error)
	ListByAction(ctx context.Context, action string, offset, limit int) ([]*types.AuditEntry, int, error)
}

// PolicyRepository defines operations for policy management
type PolicyRepository interface {
	Create(ctx context.Context, policy *types.Policy) error
	Get(ctx context.Context, id uuid.UUID) (*types.Policy, error)
	GetByName(ctx context.Context, name string) (*types.Policy, error)
	List(ctx context.Context, policyType *types.PolicyType, offset, limit int) ([]*types.Policy, int, error)
	Update(ctx context.Context, policy *types.Policy) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetLatestVersion(ctx context.Context, name string) (*types.Policy, error)
	// New methods for policy management
	GetActive(ctx context.Context) ([]*types.Policy, error)
	ListWithFilter(ctx context.Context, filter map[string]interface{}) ([]*types.Policy, int, error)
}

// PolicyAssignmentRepository defines operations for policy assignments
type PolicyAssignmentRepository interface {
	Create(ctx context.Context, assignment *types.PolicyAssignment) error
	Get(ctx context.Context, id uuid.UUID) (*types.PolicyAssignment, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*types.PolicyAssignment, error)
	ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*types.PolicyAssignment, error)
	ListByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]*types.PolicyAssignment, error)
	Update(ctx context.Context, assignment *types.PolicyAssignment) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByTenantAndPolicy(ctx context.Context, tenantID, policyID uuid.UUID) error
}

// Security Gateway - no provider key management needed

// Security Gateway uses simple environment-based rate limiting
// No complex per-tenant quota management needed

// Security Gateway - no API token or usage tracking repositories needed
// Simple token auth via environment variables only

// CacheRepository defines operations for caching frequently accessed data
type CacheRepository interface {
	// Get a cached value by key
	Get(ctx context.Context, key string) ([]byte, error)

	// Set a cached value with optional TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete a cached value
	Delete(ctx context.Context, key string) error

	// Check if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// Delete all keys matching a pattern
	DeletePattern(ctx context.Context, pattern string) error

	// Get multiple keys at once
	GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)

	// Set multiple keys at once
	SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error
}

// TransactionManager defines operations for database transactions
type TransactionManager interface {
	// Execute a function within a transaction
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// Begin a new transaction (for manual transaction management)
	BeginTx(ctx context.Context) (context.Context, error)

	// Commit the current transaction
	CommitTx(ctx context.Context) error

	// Rollback the current transaction
	RollbackTx(ctx context.Context) error
}

// Migrator defines operations for database schema management
type Migrator interface {
	// Apply pending migrations
	Migrate(ctx context.Context) error

	// Get migration status
	GetMigrationStatus(ctx context.Context) ([]map[string]interface{}, error)

	// Rollback to a specific migration version
	RollbackTo(ctx context.Context, version string) error
}

