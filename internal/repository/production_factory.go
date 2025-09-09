package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/promptshield/promptshield/internal/contracts"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	sharedcontracts "github.com/promptshield/promptshield/internal/shared/contracts"
)

// ProductionRepositoryFactory provides PostgreSQL-backed repositories with Redis caching
type ProductionRepositoryFactory struct {
	config            *RepositoryConfig
	connectionManager *ConnectionManager
	logger            *slog.Logger
	monitor           *RepositoryMonitor

	// Repository instances (cached)
	tenantRepo     domain.TenantRepository
	auditRepo      domain.AuditRepository
	assignmentRepo domain.RulepackAssignmentRepository
	apiTokenRepo   domain.APITokenRepository
	settingsRepo   domain.SettingsRepository
	rulepackRepo   contracts.RulepackRepository
	policyRepo     sharedcontracts.PolicyRepository
}

// FactoryStats provides statistics about the factory and its repositories
type FactoryStats struct {
	FactoryType     string             `json:"factory_type"`
	HasRedisCache   bool               `json:"has_redis_cache"`
	Connections     *ConnectionStats   `json:"connections"`
	RepositoryCount int                `json:"repository_count"`
	CacheConfig     *CacheConfig       `json:"cache_config,omitempty"`
}

// CacheConfig shows the cache configuration
type CacheConfig struct {
	TenantCacheTTL     string `json:"tenant_cache_ttl"`
	AssignmentCacheTTL string `json:"assignment_cache_ttl"`
	TokenCacheTTL      string `json:"token_cache_ttl"`
}

// NewProductionRepositoryFactory creates a new production repository factory
func NewProductionRepositoryFactory(config *RepositoryConfig, cm *ConnectionManager) (*ProductionRepositoryFactory, error) {
	logger := slog.With("component", "production-factory")
	
	factory := &ProductionRepositoryFactory{
		config:            config,
		connectionManager: cm,
		logger:            logger,
		monitor:           NewRepositoryMonitor(logger),
	}

	// Validate that we have the required connections for production
	if !cm.HasPostgres() {
		err := fmt.Errorf("production factory requires PostgreSQL connection")
		logger.Error("Production factory validation failed", "error", err)
		return nil, FactoryError("production", "validation", err)
	}

	logger.Info("Initializing production repository factory", 
		"has_redis", cm.HasRedis(),
		"environment", config.Environment)

	// Initialize repositories with Redis caching
	if err := factory.initializeRepositories(); err != nil {
		logger.Error("Failed to initialize repositories", "error", err)
		return nil, FactoryError("production", "initialization", err)
	}

	logger.Info("Production repository factory initialized successfully")
	return factory, nil
}

// initializeRepositories sets up all repository instances with appropriate caching
func (f *ProductionRepositoryFactory) initializeRepositories() error {
	return f.monitor.RecordOperation("initialize_repositories", func() error {
		pool := f.connectionManager.PostgresPool()
		
		f.logger.Debug("Creating base PostgreSQL repositories")
		
		// Create base PostgreSQL repositories
		pgTenantRepo := postgres.TenantRepo(pool)
		pgAssignmentRepo := postgres.RulepackAssignmentRepo(pool)
		pgAuditRepo := postgres.AuditRepo(pool)
		pgAPITokenRepo := postgres.APITokenRepo(pool)
		pgSettingsRepo := postgres.NewSettingsRepository(pool)
		pgRulepackRepo := postgres.RulepackRepo(pool)
		
		// For now, use in-memory policy repository (TODO: implement PostgreSQL version)
		pgPolicyRepo := memory.NewPolicyRepository()

		// Apply Redis caching if available
		if f.connectionManager.HasRedis() {
			redisClient := f.connectionManager.RedisClient()
			
			f.logger.Info("Applying Redis caching to hot-path repositories",
				"tenant_ttl", f.config.TenantCacheTTL,
				"assignment_ttl", f.config.AssignmentCacheTTL,
				"token_ttl", f.config.TokenCacheTTL)
			
			// Cache hot-path repositories
			f.tenantRepo = postgres.NewRedisTenantRepository(pgTenantRepo, redisClient, f.config.TenantCacheTTL)
			f.assignmentRepo = postgres.NewRedisRulepackAssignmentRepository(pgAssignmentRepo, redisClient, f.config.AssignmentCacheTTL)
			f.apiTokenRepo = postgres.NewRedisAPITokenRepository(pgAPITokenRepo, redisClient, f.config.TokenCacheTTL)
		} else {
			f.logger.Info("Redis not available, using direct PostgreSQL repositories")
			
			// Use direct PostgreSQL repositories without caching
			f.tenantRepo = pgTenantRepo
			f.assignmentRepo = pgAssignmentRepo
			f.apiTokenRepo = pgAPITokenRepo
		}

		// These repositories typically don't benefit from caching
		f.auditRepo = pgAuditRepo      // Write-heavy, read-light
		f.settingsRepo = pgSettingsRepo // Infrequently accessed
		f.rulepackRepo = pgRulepackRepo // Has its own caching strategy
		f.policyRepo = pgPolicyRepo    // In-memory for now

		f.logger.Debug("Repository initialization completed successfully")
		return nil
	})
}

// Tenant returns the tenant repository
func (f *ProductionRepositoryFactory) Tenant() domain.TenantRepository {
	return f.tenantRepo
}

// Audit returns the audit repository
func (f *ProductionRepositoryFactory) Audit() domain.AuditRepository {
	return f.auditRepo
}

// RulepackAssignment returns the rulepack assignment repository
func (f *ProductionRepositoryFactory) RulepackAssignment() domain.RulepackAssignmentRepository {
	return f.assignmentRepo
}

// APIToken returns the API token repository
func (f *ProductionRepositoryFactory) APIToken() domain.APITokenRepository {
	return f.apiTokenRepo
}

// Settings returns the settings repository
func (f *ProductionRepositoryFactory) Settings() domain.SettingsRepository {
	return f.settingsRepo
}

// Rulepack returns the rulepack repository
func (f *ProductionRepositoryFactory) Rulepack() contracts.RulepackRepository {
	return f.rulepackRepo
}

// Policy returns the policy repository
func (f *ProductionRepositoryFactory) Policy() sharedcontracts.PolicyRepository {
	return f.policyRepo
}

// Close closes all repository connections with proper lifecycle management
func (f *ProductionRepositoryFactory) Close() error {
	f.logger.Info("Closing production repository factory")
	
	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	var errors []error
	
	// Stop background monitoring first
	f.logger.Debug("Stopping background monitoring")
	// Note: monitoring goroutines will stop when context is cancelled
	
	// Flush any pending operations
	if err := f.flushPendingOperations(ctx); err != nil {
		f.logger.Error("Failed to flush pending operations", "error", err)
		errors = append(errors, fmt.Errorf("flush pending operations: %w", err))
	}
	
	// Log final metrics before closing
	f.monitor.LogMetricsSummary()
	
	// Close connection manager
	if err := f.connectionManager.Close(); err != nil {
		f.logger.Error("Failed to close connection manager", "error", err)
		errors = append(errors, fmt.Errorf("connection manager close: %w", err))
	}
	
	// Clear repository references to help with garbage collection
	f.tenantRepo = nil
	f.auditRepo = nil
	f.assignmentRepo = nil
	f.apiTokenRepo = nil
	f.settingsRepo = nil
	f.rulepackRepo = nil
	f.policyRepo = nil
	
	if len(errors) > 0 {
		return fmt.Errorf("production factory close errors: %v", errors)
	}
	
	f.logger.Info("Production repository factory closed successfully")
	return nil
}

// flushPendingOperations ensures all pending operations are completed
func (f *ProductionRepositoryFactory) flushPendingOperations(ctx context.Context) error {
	// For now, this is a placeholder for any pending operations
	// In the future, this could flush caches, complete transactions, etc.
	f.logger.Debug("Flushing pending operations")
	
	// Give a small grace period for any ongoing operations
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}
	
	return nil
}

// HealthCheck verifies repository health
func (f *ProductionRepositoryFactory) HealthCheck(ctx context.Context) error {
	return f.monitor.RecordOperation("factory_health_check", func() error {
		return f.connectionManager.HealthCheck(ctx)
	})
}

// GetStats returns detailed statistics about the factory
func (f *ProductionRepositoryFactory) GetStats(ctx context.Context) *FactoryStats {
	stats := &FactoryStats{
		FactoryType:     "production",
		HasRedisCache:   f.connectionManager.HasRedis(),
		Connections:     f.connectionManager.GetConnectionStats(ctx),
		RepositoryCount: f.getRepositoryCount(),
	}

	if f.connectionManager.HasRedis() {
		stats.CacheConfig = &CacheConfig{
			TenantCacheTTL:     f.config.TenantCacheTTL.String(),
			AssignmentCacheTTL: f.config.AssignmentCacheTTL.String(),
			TokenCacheTTL:      f.config.TokenCacheTTL.String(),
		}
	}

	return stats
}

// getRepositoryCount returns the number of initialized repositories
func (f *ProductionRepositoryFactory) getRepositoryCount() int {
	count := 0
	if f.tenantRepo != nil {
		count++
	}
	if f.auditRepo != nil {
		count++
	}
	if f.assignmentRepo != nil {
		count++
	}
	if f.apiTokenRepo != nil {
		count++
	}
	if f.settingsRepo != nil {
		count++
	}
	if f.rulepackRepo != nil {
		count++
	}
	return count
}

// ValidateRepositories checks that all repositories are properly initialized
func (f *ProductionRepositoryFactory) ValidateRepositories(ctx context.Context) error {
	var errors []error

	// Check that all repositories are initialized
	if f.tenantRepo == nil {
		errors = append(errors, fmt.Errorf("tenant repository not initialized"))
	}
	if f.auditRepo == nil {
		errors = append(errors, fmt.Errorf("audit repository not initialized"))
	}
	if f.assignmentRepo == nil {
		errors = append(errors, fmt.Errorf("assignment repository not initialized"))
	}
	if f.apiTokenRepo == nil {
		errors = append(errors, fmt.Errorf("API token repository not initialized"))
	}
	if f.settingsRepo == nil {
		errors = append(errors, fmt.Errorf("settings repository not initialized"))
	}
	if f.rulepackRepo == nil {
		errors = append(errors, fmt.Errorf("rulepack repository not initialized"))
	}

	if len(errors) > 0 {
		return fmt.Errorf("repository validation failed: %v", errors)
	}

	return nil
}

// Reconnect attempts to reconnect any failed connections
func (f *ProductionRepositoryFactory) Reconnect(ctx context.Context) error {
	// Reconnect underlying connections
	if err := f.connectionManager.Reconnect(ctx); err != nil {
		return fmt.Errorf("connection manager reconnect failed: %w", err)
	}

	// Reinitialize repositories if connections were restored
	if err := f.initializeRepositories(); err != nil {
		return fmt.Errorf("repository reinitialization failed: %w", err)
	}

	return nil
}

// GetMonitor returns the repository monitor for metrics access
func (f *ProductionRepositoryFactory) GetMonitor() *RepositoryMonitor {
	return f.monitor
}

// StartMonitoring begins background monitoring for the factory
func (f *ProductionRepositoryFactory) StartMonitoring(ctx context.Context) {
	f.logger.Info("Starting production factory monitoring")
	go f.monitor.StartMonitoring(ctx)
	
	// Also start connection manager monitoring
	f.connectionManager.StartMonitoring(ctx)
}