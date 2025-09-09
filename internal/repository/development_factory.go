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

// DevelopmentRepositoryFactory provides PostgreSQL-backed repositories without caching
type DevelopmentRepositoryFactory struct {
	config            *RepositoryConfig
	connectionManager *ConnectionManager
	logger            *slog.Logger

	// Repository instances
	tenantRepo     domain.TenantRepository
	auditRepo      domain.AuditRepository
	assignmentRepo domain.RulepackAssignmentRepository
	apiTokenRepo   domain.APITokenRepository
	settingsRepo   domain.SettingsRepository
	rulepackRepo   contracts.RulepackRepository
	policyRepo     sharedcontracts.PolicyRepository
}

// DevelopmentFactoryStats provides statistics about the development factory
type DevelopmentFactoryStats struct {
	FactoryType     string           `json:"factory_type"`
	HasRedisCache   bool             `json:"has_redis_cache"`
	Connections     *ConnectionStats `json:"connections"`
	RepositoryCount int              `json:"repository_count"`
	DebugMode       bool             `json:"debug_mode"`
	DatabaseURL     string           `json:"database_url,omitempty"`
}

// NewDevelopmentRepositoryFactory creates a new development repository factory
func NewDevelopmentRepositoryFactory(config *RepositoryConfig, cm *ConnectionManager) (*DevelopmentRepositoryFactory, error) {
	factory := &DevelopmentRepositoryFactory{
		config:            config,
		connectionManager: cm,
		logger:            slog.With("component", "development-factory"),
	}

	// Initialize repositories with detailed logging for debugging
	if err := factory.initializeRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize development repositories: %w", err)
	}

	return factory, nil
}

// initializeRepositories sets up all repository instances without caching for simpler debugging
func (f *DevelopmentRepositoryFactory) initializeRepositories() error {
	if !f.connectionManager.HasPostgres() {
		return fmt.Errorf("development factory requires PostgreSQL connection")
	}

	pool := f.connectionManager.PostgresPool()
	
	// Create direct PostgreSQL repositories without caching for easier debugging
	f.tenantRepo = postgres.TenantRepo(pool)
	f.assignmentRepo = postgres.RulepackAssignmentRepo(pool)
	f.auditRepo = postgres.AuditRepo(pool)
	f.apiTokenRepo = postgres.APITokenRepo(pool)
	f.settingsRepo = postgres.NewSettingsRepository(pool)
	f.rulepackRepo = postgres.RulepackRepo(pool)
	f.policyRepo = memory.NewPolicyRepository() // In-memory for now

	// Log repository initialization for debugging
	fmt.Printf("Development factory: Initialized %d repositories without caching\n", f.getRepositoryCount())
	fmt.Printf("Development factory: PostgreSQL connection available, Redis caching disabled for debugging\n")

	// Check if Redis was configured but not used
	if f.config.RedisAddr != "" && !f.connectionManager.HasRedis() {
		fmt.Printf("Development factory: Redis configured (%s) but connection failed - continuing without cache\n", f.config.RedisAddr)
	}

	return nil
}

// Tenant returns the tenant repository
func (f *DevelopmentRepositoryFactory) Tenant() domain.TenantRepository {
	return f.tenantRepo
}

// Audit returns the audit repository
func (f *DevelopmentRepositoryFactory) Audit() domain.AuditRepository {
	return f.auditRepo
}

// RulepackAssignment returns the rulepack assignment repository
func (f *DevelopmentRepositoryFactory) RulepackAssignment() domain.RulepackAssignmentRepository {
	return f.assignmentRepo
}

// APIToken returns the API token repository
func (f *DevelopmentRepositoryFactory) APIToken() domain.APITokenRepository {
	return f.apiTokenRepo
}

// Settings returns the settings repository
func (f *DevelopmentRepositoryFactory) Settings() domain.SettingsRepository {
	return f.settingsRepo
}

// Rulepack returns the rulepack repository
func (f *DevelopmentRepositoryFactory) Rulepack() contracts.RulepackRepository {
	return f.rulepackRepo
}

// Policy returns the policy repository
func (f *DevelopmentRepositoryFactory) Policy() sharedcontracts.PolicyRepository {
	return f.policyRepo
}

// Close closes all repository connections with proper lifecycle management
func (f *DevelopmentRepositoryFactory) Close() error {
	f.logger.Info("Closing development repository factory")
	
	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	var errors []error
	
	// Flush any pending operations
	if err := f.flushPendingOperations(ctx); err != nil {
		f.logger.Error("Failed to flush pending operations", "error", err)
		errors = append(errors, fmt.Errorf("flush pending operations: %w", err))
	}
	
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
		return fmt.Errorf("development factory close errors: %v", errors)
	}
	
	f.logger.Info("Development repository factory closed successfully")
	return nil
}

// flushPendingOperations ensures all pending operations are completed
func (f *DevelopmentRepositoryFactory) flushPendingOperations(ctx context.Context) error {
	f.logger.Debug("Flushing pending operations")
	
	// Give a small grace period for any ongoing operations
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}
	
	return nil
}

// HealthCheck verifies repository health
func (f *DevelopmentRepositoryFactory) HealthCheck(ctx context.Context) error {
	return f.connectionManager.HealthCheck(ctx)
}

// GetStats returns detailed statistics about the development factory
func (f *DevelopmentRepositoryFactory) GetStats(ctx context.Context) *DevelopmentFactoryStats {
	stats := &DevelopmentFactoryStats{
		FactoryType:     "development",
		HasRedisCache:   false, // Development factory never uses Redis cache
		Connections:     f.connectionManager.GetConnectionStats(ctx),
		RepositoryCount: f.getRepositoryCount(),
		DebugMode:       true,
	}

	// Include database URL for debugging (but mask sensitive parts)
	if f.config.DatabaseURL != "" {
		stats.DatabaseURL = f.maskDatabaseURL(f.config.DatabaseURL)
	}

	return stats
}

// getRepositoryCount returns the number of initialized repositories
func (f *DevelopmentRepositoryFactory) getRepositoryCount() int {
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
	if f.policyRepo != nil {
		count++
	}
	return count
}

// maskDatabaseURL masks sensitive information in database URL for logging
func (f *DevelopmentRepositoryFactory) maskDatabaseURL(url string) string {
	// Simple masking - in a real implementation you'd want more sophisticated masking
	if len(url) > 20 {
		return url[:10] + "***" + url[len(url)-7:]
	}
	return "***"
}

// ValidateRepositories checks that all repositories are properly initialized
func (f *DevelopmentRepositoryFactory) ValidateRepositories(ctx context.Context) error {
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
	if f.policyRepo == nil {
		errors = append(errors, fmt.Errorf("policy repository not initialized"))
	}

	if len(errors) > 0 {
		fmt.Printf("Development factory validation failed: %v\n", errors)
		return fmt.Errorf("repository validation failed: %v", errors)
	}

	fmt.Printf("Development factory validation passed: all %d repositories initialized\n", f.getRepositoryCount())
	return nil
}

// Reconnect attempts to reconnect any failed connections with detailed logging
func (f *DevelopmentRepositoryFactory) Reconnect(ctx context.Context) error {
	fmt.Printf("Development factory: Attempting to reconnect failed connections...\n")

	// Reconnect underlying connections
	if err := f.connectionManager.Reconnect(ctx); err != nil {
		fmt.Printf("Development factory: Connection manager reconnect failed: %v\n", err)
		return fmt.Errorf("connection manager reconnect failed: %w", err)
	}

	// Reinitialize repositories if connections were restored
	if err := f.initializeRepositories(); err != nil {
		fmt.Printf("Development factory: Repository reinitialization failed: %v\n", err)
		return fmt.Errorf("repository reinitialization failed: %w", err)
	}

	fmt.Printf("Development factory: Reconnection successful\n")
	return nil
}

// LogRepositoryUsage logs detailed information about repository usage for debugging
func (f *DevelopmentRepositoryFactory) LogRepositoryUsage(ctx context.Context) {
	fmt.Printf("=== Development Factory Repository Usage ===\n")
	fmt.Printf("Factory Type: %s\n", "development")
	fmt.Printf("Caching Enabled: %t\n", false)
	fmt.Printf("Repository Count: %d\n", f.getRepositoryCount())
	
	// Log connection stats
	stats := f.connectionManager.GetConnectionStats(ctx)
	if stats.PostgreSQL != nil {
		fmt.Printf("PostgreSQL Connected: %t\n", stats.PostgreSQL.Connected)
		if stats.PostgreSQL.Error != "" {
			fmt.Printf("PostgreSQL Error: %s\n", stats.PostgreSQL.Error)
		}
	}
	
	if stats.Redis != nil {
		fmt.Printf("Redis Connected: %t\n", stats.Redis.Connected)
		if stats.Redis.Error != "" {
			fmt.Printf("Redis Error: %s\n", stats.Redis.Error)
		}
	}
	
	fmt.Printf("=== End Repository Usage ===\n")
}

// EnableVerboseLogging enables detailed logging for debugging
func (f *DevelopmentRepositoryFactory) EnableVerboseLogging() {
	fmt.Printf("Development factory: Verbose logging enabled\n")
	fmt.Printf("Development factory: Configuration - DB: %s, Redis: %s\n", 
		f.maskDatabaseURL(f.config.DatabaseURL), f.config.RedisAddr)
	fmt.Printf("Development factory: Connection limits - Max: %d, Idle: %d, Timeout: %v\n",
		f.config.MaxConnections, f.config.MaxIdleConnections, f.config.ConnectionTimeout)
}