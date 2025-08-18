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

// ProviderKeyRepository defines operations for managing provider API keys
type ProviderKeyRepository interface {
	Create(ctx context.Context, key *types.ProviderKey) error
	Get(ctx context.Context, id uuid.UUID) (*types.ProviderKey, error)
	GetByAlias(ctx context.Context, tenantID uuid.UUID, provider types.Provider, alias string) (*types.ProviderKey, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*types.ProviderKey, error)
	ListByProvider(ctx context.Context, tenantID uuid.UUID, provider types.Provider) ([]*types.ProviderKey, error)
	Update(ctx context.Context, key *types.ProviderKey) error
	Delete(ctx context.Context, id uuid.UUID) error
	Rotate(ctx context.Context, id uuid.UUID, newEncryptedKey string) error
	SetDefault(ctx context.Context, tenantID uuid.UUID, provider types.Provider, keyID uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// QuotaRepository defines operations for quota management and rate limiting
type QuotaRepository interface {
	Create(ctx context.Context, quota *types.Quota) error
	Get(ctx context.Context, tenantID uuid.UUID) (*types.Quota, error)
	Update(ctx context.Context, quota *types.Quota) error
	Delete(ctx context.Context, tenantID uuid.UUID) error
	CheckRateLimit(ctx context.Context, tenantID uuid.UUID) (*types.RateLimitResult, error)
	IncrementUsage(ctx context.Context, tenantID uuid.UUID, tokens int64) error
}

// APITokenRepository defines operations for API token management
type APITokenRepository interface {
	Create(ctx context.Context, token *types.APIToken) error
	Get(ctx context.Context, id uuid.UUID) (*types.APIToken, error)
	GetByHash(ctx context.Context, hashedToken string) (*types.APIToken, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*types.APIToken, error)
	Update(ctx context.Context, token *types.APIToken) error
	Delete(ctx context.Context, id uuid.UUID) error
	Rotate(ctx context.Context, id uuid.UUID) (string, error)
}

// UsageRepository defines operations for usage tracking and analytics
type UsageRepository interface {
	// Record a single usage event
	RecordUsage(ctx context.Context, record *types.UsageRecord) error

	// Query usage data with filters and aggregation
	QueryUsage(ctx context.Context, query *types.UsageQuery) (*types.UsageResult, error)

	// Get aggregated metrics for a time window
	GetMetrics(ctx context.Context, tenantID uuid.UUID, window types.TimeWindow, start, end time.Time) ([]*types.UsageMetric, error)

	// Get usage summary for a tenant
	GetUsageSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (map[string]interface{}, error)

	// Delete old usage records (for data retention)
	DeleteOldRecords(ctx context.Context, before time.Time) error
}

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
