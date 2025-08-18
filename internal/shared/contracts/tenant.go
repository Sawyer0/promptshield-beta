package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// TenantManager defines the interface for tenant management operations
type TenantManager interface {
	// CreateTenant creates a new tenant
	CreateTenant(ctx context.Context, name string, metadata map[string]interface{}) (types.TenantID, error)
	
	// GetTenant retrieves a tenant by ID
	GetTenant(ctx context.Context, tenantID types.TenantID) (*types.TenantContext, error)
	
	// UpdateTenant updates tenant information
	UpdateTenant(ctx context.Context, tenantID types.TenantID, updates map[string]interface{}) error
	
	// DeleteTenant deletes a tenant
	DeleteTenant(ctx context.Context, tenantID types.TenantID) error
	
	// ListTenants lists all tenants
	ListTenants(ctx context.Context) ([]*types.TenantContext, error)
	
	// ActivateTenant activates a tenant
	ActivateTenant(ctx context.Context, tenantID types.TenantID) error
	
	// DeactivateTenant deactivates a tenant
	DeactivateTenant(ctx context.Context, tenantID types.TenantID) error
	
	// GetTenantStatus returns the status of a tenant
	GetTenantStatus(ctx context.Context, tenantID types.TenantID) (string, error)
	
	// ValidateTenant validates tenant configuration
	ValidateTenant(ctx context.Context, tenantID types.TenantID) error
}

// TenantQuotaManager defines the interface for tenant quota management
type TenantQuotaManager interface {
	// SetQuota sets quota limits for a tenant
	SetQuota(ctx context.Context, quota *types.TenantQuota) error
	
	// GetQuota retrieves quota configuration for a tenant
	GetQuota(ctx context.Context, tenantID types.TenantID) (*types.TenantQuota, error)
	
	// UpdateQuota updates quota limits for a tenant
	UpdateQuota(ctx context.Context, tenantID types.TenantID, updates *types.TenantQuota) error
	
	// DeleteQuota removes quota limits for a tenant
	DeleteQuota(ctx context.Context, tenantID types.TenantID) error
	
	// CheckRateLimit checks if a request is within rate limits
	CheckRateLimit(ctx context.Context, tenantID types.TenantID, requestType string) (*types.RateLimitResult, error)
	
	// IncrementUsage increments usage counters for a tenant
	IncrementUsage(ctx context.Context, tenantID types.TenantID, requests int, tokens int64) error
	
	// GetUsage returns current usage for a tenant
	GetUsage(ctx context.Context, tenantID types.TenantID, timeWindow time.Duration) (map[string]interface{}, error)
	
	// ResetUsage resets usage counters for a tenant
	ResetUsage(ctx context.Context, tenantID types.TenantID) error
	
	// GetQuotaStatus returns quota status and remaining limits
	GetQuotaStatus(ctx context.Context, tenantID types.TenantID) (map[string]interface{}, error)
}

// TenantContextProvider defines the interface for tenant context operations
type TenantContextProvider interface {
	// ExtractTenantID extracts tenant ID from request context
	ExtractTenantID(ctx context.Context) (types.TenantID, error)
	
	// EnrichContext enriches context with tenant information
	EnrichContext(ctx context.Context, tenantID types.TenantID) (context.Context, error)
	
	// ValidateContext validates tenant context
	ValidateContext(ctx context.Context) error
	
	// GetTenantFromAPIKey extracts tenant ID from API key
	GetTenantFromAPIKey(ctx context.Context, apiKey string) (types.TenantID, error)
	
	// GetTenantFromJWT extracts tenant ID from JWT token
	GetTenantFromJWT(ctx context.Context, token string) (types.TenantID, error)
	
	// CreateTenantContext creates a tenant context
	CreateTenantContext(tenantID types.TenantID, name string, metadata map[string]interface{}) *types.TenantContext
}

// MultiTenantService defines the interface for multi-tenant operations
type MultiTenantService interface {
	// IsolateData ensures data isolation between tenants
	IsolateData(ctx context.Context, tenantID types.TenantID, operation func(ctx context.Context) error) error
	
	// RouteToTenant routes requests to tenant-specific resources
	RouteToTenant(ctx context.Context, tenantID types.TenantID, resourceType string) (string, error)
	
	// GetTenantConfiguration returns tenant-specific configuration
	GetTenantConfiguration(ctx context.Context, tenantID types.TenantID) (map[string]interface{}, error)
	
	// SetTenantConfiguration sets tenant-specific configuration
	SetTenantConfiguration(ctx context.Context, tenantID types.TenantID, config map[string]interface{}) error
	
	// MigrateTenantData migrates data for a tenant
	MigrateTenantData(ctx context.Context, tenantID types.TenantID, targetVersion string) error
	
	// BackupTenantData creates a backup of tenant data
	BackupTenantData(ctx context.Context, tenantID types.TenantID) ([]byte, error)
	
	// RestoreTenantData restores tenant data from backup
	RestoreTenantData(ctx context.Context, tenantID types.TenantID, backupData []byte) error
	
	// GetTenantMetrics returns metrics for a specific tenant
	GetTenantMetrics(ctx context.Context, tenantID types.TenantID, timeRange types.TimeRange) (map[string]interface{}, error)
}