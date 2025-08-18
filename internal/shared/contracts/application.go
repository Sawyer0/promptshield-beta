package contracts

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// RulepackService defines the interface for rulepack business logic
type RulepackService interface {
	// Create creates a new rulepack
	Create(ctx context.Context, tenantID uuid.UUID, name, description string) (uuid.UUID, error)

	// Upload validates and creates a new rulepack version, optionally activating it
	Upload(ctx context.Context, tenantID uuid.UUID, packID uuid.UUID, version int, dsl json.RawMessage, activate bool) (uuid.UUID, error)

	// CreateVersionActivate stores a new version and immediately activates it
	CreateVersionActivate(ctx context.Context, tenantID uuid.UUID, packID uuid.UUID, version int, dsl json.RawMessage) error

	// SetActive activates a specific rulepack version
	SetActive(ctx context.Context, tenantID uuid.UUID, packID, versionID uuid.UUID) error

	// ActivateLatest activates the latest version of a rulepack
	ActivateLatest(ctx context.Context, tenantID uuid.UUID, packID uuid.UUID) error

	// GetActive returns the active version DSL and version number for a rulepack
	GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error)

	// GetVersion returns the DSL and status for a specific version
	GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error)

	// ValidateDSL validates a rulepack DSL payload (JSON or YAML)
	ValidateDSL(data []byte) (bool, []string, []string)

	// ParseDSL parses and validates DSL, returning the RulePack struct
	ParseDSL(data []byte) (rules.RulePack, error)

	// List returns all rulepacks for a tenant
	List(ctx context.Context, tenantID uuid.UUID) ([]types.RulepackInfo, error)

	// Delete removes a rulepack and all its versions
	Delete(ctx context.Context, tenantID uuid.UUID, packID uuid.UUID) error
}

// RuntimeConfigStore defines the interface for runtime configuration management
type RuntimeConfigStore interface {
	// Get retrieves a configuration value
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a configuration value
	Set(ctx context.Context, key string, value interface{}) error

	// GetAll retrieves all configuration values
	GetAll(ctx context.Context) (map[string]interface{}, error)

	// Delete removes a configuration value
	Delete(ctx context.Context, key string) error

	// Watch watches for configuration changes
	Watch(ctx context.Context, key string) (<-chan interface{}, error)

	// GetVersion returns the current configuration version
	GetVersion(ctx context.Context) (string, error)

	// Reload reloads configuration from source
	Reload(ctx context.Context) error

	// Validate validates the current configuration
	Validate(ctx context.Context) (*types.ValidationResult, error)

	// Export exports configuration to a format
	Export(ctx context.Context, format string) ([]byte, error)

	// Import imports configuration from data
	Import(ctx context.Context, data []byte, format string) error

	// GetMetadata returns configuration metadata
	GetMetadata(ctx context.Context) (*types.RuntimeConfigStore, error)

	// Backup creates a configuration backup
	Backup(ctx context.Context) ([]byte, error)

	// Restore restores configuration from backup
	Restore(ctx context.Context, backup []byte) error
}

// EventHub defines the interface for event broadcasting and SSE
type EventHub interface {
	// Subscribe subscribes to events with a filter
	Subscribe(ctx context.Context, filter types.EventFilter) (<-chan *types.Event, error)

	// Publish publishes an event to all matching subscribers
	Publish(ctx context.Context, event *types.Event) error

	// Unsubscribe removes a subscription
	Unsubscribe(subscriptionID string) error

	// GetStats returns event hub statistics
	GetStats(ctx context.Context) (*types.EventHubStats, error)

	// GetActiveConnections returns the number of active connections
	GetActiveConnections() int

	// Broadcast broadcasts an event to all subscribers
	Broadcast(ctx context.Context, eventType string, data map[string]interface{}) error

	// BroadcastToTenant broadcasts an event to a specific tenant's subscribers
	BroadcastToTenant(ctx context.Context, tenantID string, eventType string, data map[string]interface{}) error

	// Close closes the event hub and all connections
	Close() error
}

// RoutingService defines the interface for request routing logic
type RoutingService interface {
	// Route determines the target for a request based on routing rules
	Route(ctx context.Context, request *types.RequestContext) (*types.RoutingTarget, error)

	// CreateRule creates a new routing rule
	CreateRule(ctx context.Context, rule *types.RoutingRule) error

	// UpdateRule updates an existing routing rule
	UpdateRule(ctx context.Context, ruleID string, rule *types.RoutingRule) error

	// DeleteRule deletes a routing rule
	DeleteRule(ctx context.Context, ruleID string) error

	// GetRule retrieves a routing rule by ID
	GetRule(ctx context.Context, ruleID string) (*types.RoutingRule, error)

	// ListRules lists all routing rules
	ListRules(ctx context.Context) ([]*types.RoutingRule, error)

	// EnableRule enables a routing rule
	EnableRule(ctx context.Context, ruleID string) error

	// DisableRule disables a routing rule
	DisableRule(ctx context.Context, ruleID string) error

	// TestRule tests a routing rule against a request
	TestRule(ctx context.Context, rule *types.RoutingRule, request *types.RequestContext) (bool, error)

	// GetSupportedProviders returns list of supported providers
	GetSupportedProviders(ctx context.Context) ([]*types.SupportedProvider, error)
}

// ValidationService defines the interface for validation operations
type ValidationService interface {
	// ValidateRulepack validates a rulepack configuration
	ValidateRulepack(ctx context.Context, data []byte) (*types.ValidationResult, error)

	// ValidateProvider validates a provider configuration
	ValidateProvider(ctx context.Context, config *types.ProviderConfig) (*types.ValidationResult, error)

	// ValidatePolicy validates a policy configuration
	ValidatePolicy(ctx context.Context, policy *types.Policy) (*types.ValidationResult, error)

	// ValidateQuota validates a quota configuration
	ValidateQuota(ctx context.Context, quota *types.Quota) (*types.ValidationResult, error)

	// ValidateAPIToken validates an API token format
	ValidateAPIToken(ctx context.Context, token string) (*types.ValidationResult, error)

	// ValidateJWT validates a JWT token
	ValidateJWT(ctx context.Context, token string) (*types.ValidationResult, error)

	// ValidateConfig validates runtime configuration
	ValidateConfig(ctx context.Context, config map[string]interface{}) (*types.ValidationResult, error)

	// ValidateSchema validates data against a schema
	ValidateSchema(ctx context.Context, data interface{}, schema string) (*types.ValidationResult, error)
}

// ServiceRegistry defines the interface for service registration and discovery
type ServiceRegistry interface {
	// RegisterService registers a service instance
	RegisterService(ctx context.Context, metadata *types.ServiceMetadata) error

	// DeregisterService deregisters a service instance
	DeregisterService(ctx context.Context, serviceName string) error

	// GetService retrieves service metadata
	GetService(ctx context.Context, serviceName string) (*types.ServiceMetadata, error)

	// ListServices lists all registered services
	ListServices(ctx context.Context) ([]*types.ServiceMetadata, error)

	// UpdateServiceStatus updates service status
	UpdateServiceStatus(ctx context.Context, serviceName string, status string) error

	// HealthCheck performs health check on a service
	HealthCheck(ctx context.Context, serviceName string) (*types.HealthStatus, error)

	// Watch watches for service changes
	Watch(ctx context.Context, serviceName string) (<-chan *types.ServiceMetadata, error)

	// GetDependencies returns service dependencies
	GetDependencies(ctx context.Context, serviceName string) ([]string, error)

	// CheckDependencies checks if all dependencies are healthy
	CheckDependencies(ctx context.Context, serviceName string) (map[string]*types.HealthStatus, error)
}

// AsyncProcessor defines the interface for asynchronous processing
type AsyncProcessor interface {
	// ProcessAsync processes a task asynchronously
	ProcessAsync(ctx context.Context, task interface{}) error

	// ProcessBatch processes multiple tasks in a batch
	ProcessBatch(ctx context.Context, tasks []interface{}) error

	// Schedule schedules a task for future execution
	Schedule(ctx context.Context, task interface{}, delay time.Duration) error

	// GetStatus returns processing status
	GetStatus(ctx context.Context, taskID string) (string, error)

	// Cancel cancels a scheduled or running task
	Cancel(ctx context.Context, taskID string) error

	// GetStats returns processor statistics
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// RegisterHandler registers a task handler
	RegisterHandler(taskType string, handler TaskHandler) error

	// UnregisterHandler unregisters a task handler
	UnregisterHandler(taskType string) error
}

// TaskHandler defines the interface for handling asynchronous tasks
type TaskHandler interface {
	Handle(ctx context.Context, task interface{}) error
}

// CacheManager defines the interface for cache management operations
type CacheManager interface {
	// GetCache returns a cache instance by name
	GetCache(name string) (Cache, error)

	// CreateCache creates a new cache instance
	CreateCache(name string, config *types.CacheConfig) (Cache, error)

	// DeleteCache removes a cache instance
	DeleteCache(name string) error

	// ListCaches lists all cache instances
	ListCaches() []string

	// GetCacheStats returns statistics for all caches
	GetCacheStats() (map[string]*types.CacheStats, error)

	// FlushAll flushes all caches
	FlushAll(ctx context.Context) error

	// Close closes all cache connections
	Close() error
}
