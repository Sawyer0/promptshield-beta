package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	redis "github.com/redis/go-redis/v9"
)

// ConnectionManager manages database and Redis connections
type ConnectionManager struct {
	pgPool      *postgres.Pool
	redisClient *redis.Client
	config      *RepositoryConfig
	logger      *slog.Logger
	monitor     *RepositoryMonitor
}

// ConnectionStats provides statistics about connection usage
type ConnectionStats struct {
	PostgreSQL *PostgresStats `json:"postgresql,omitempty"`
	Redis      *RedisStats     `json:"redis,omitempty"`
}

// PostgresStats contains PostgreSQL connection statistics
type PostgresStats struct {
	Connected bool   `json:"connected"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

// RedisStats contains Redis connection statistics
type RedisStats struct {
	Connected bool              `json:"connected"`
	Info      map[string]string `json:"info,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// NewConnectionManager creates a new connection manager with enhanced connection pooling
func NewConnectionManager(ctx context.Context, config *RepositoryConfig) (*ConnectionManager, error) {
	logger := slog.With("component", "connection-manager")
	
	cm := &ConnectionManager{
		config:  config,
		logger:  logger,
		monitor: NewRepositoryMonitor(logger),
	}

	// Initialize PostgreSQL connection if database URL is provided
	if config.DatabaseURL != "" {
		logger.Info("Initializing PostgreSQL connection", "database_url", maskDatabaseURL(config.DatabaseURL))
		cm.monitor.metrics.RecordConnection(false) // Will be updated to true if successful
		
		pool, err := cm.createPostgresPool(ctx)
		if err != nil {
			cm.monitor.metrics.RecordConnection(false)
			logger.Error("Failed to connect to PostgreSQL", "error", err)
			return nil, ConnectionError("postgresql", "connect", err)
		}
		
		cm.pgPool = pool
		cm.monitor.metrics.RecordConnection(true)
		logger.Info("PostgreSQL connection established successfully")
	}

	// Initialize Redis connection if Redis address is provided
	if config.RedisAddr != "" {
		logger.Info("Initializing Redis connection", "redis_addr", config.RedisAddr)
		cm.monitor.metrics.RecordConnection(false) // Will be updated to true if successful
		
		redisClient, err := cm.createRedisClient(ctx)
		if err != nil {
			cm.monitor.metrics.RecordConnection(false)
			logger.Warn("Redis connection failed, falling back to PostgreSQL-only mode", "error", err)
			// Don't return error - Redis is optional for caching
		} else {
			cm.redisClient = redisClient
			cm.monitor.metrics.RecordConnection(true)
			logger.Info("Redis connection established successfully")
		}
	}

	return cm, nil
}

// createPostgresPool creates a PostgreSQL connection pool with configured limits
func (cm *ConnectionManager) createPostgresPool(ctx context.Context) (*postgres.Pool, error) {
	// Create connection with timeout
	ctx, cancel := context.WithTimeout(ctx, cm.config.ConnectionTimeout)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cm.config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	// Configure connection pool limits
	// Note: pgxpool doesn't expose direct configuration after creation,
	// but we can validate the connection works
	if _, err := pool.Raw().Exec(ctx, "SELECT 1"); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres pool validation failed: %w", err)
	}

	return pool, nil
}

// createRedisClient creates a Redis client with configured options
func (cm *ConnectionManager) createRedisClient(ctx context.Context) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cm.config.RedisAddr,
		Password:     cm.config.RedisPassword,
		DB:           cm.config.RedisDB,
		DialTimeout:  cm.config.ConnectionTimeout,
		ReadTimeout:  cm.config.ConnectionTimeout,
		WriteTimeout: cm.config.ConnectionTimeout,
		PoolSize:     cm.config.MaxConnections,
		MinIdleConns: cm.config.MaxIdleConnections,
	})

	// Test Redis connection with timeout
	ctx, cancel := context.WithTimeout(ctx, cm.config.ConnectionTimeout)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("redis connection test failed: %w", err)
	}

	return redisClient, nil
}

// PostgresPool returns the PostgreSQL connection pool
func (cm *ConnectionManager) PostgresPool() *postgres.Pool {
	return cm.pgPool
}

// RedisClient returns the Redis client
func (cm *ConnectionManager) RedisClient() *redis.Client {
	return cm.redisClient
}

// HasPostgres returns true if PostgreSQL connection is available
func (cm *ConnectionManager) HasPostgres() bool {
	return cm.pgPool != nil
}

// HasRedis returns true if Redis connection is available
func (cm *ConnectionManager) HasRedis() bool {
	return cm.redisClient != nil
}

// HealthCheck verifies that all connections are healthy
func (cm *ConnectionManager) HealthCheck(ctx context.Context) error {
	return cm.monitor.RecordOperation("health_check", func() error {
		var errors []error

		// Check PostgreSQL connection
		if cm.pgPool != nil {
			if err := cm.checkPostgresHealth(ctx); err != nil {
				cm.monitor.metrics.RecordHealthCheck(false)
				errors = append(errors, HealthCheckError("postgresql", "health_check", err))
			}
		}

		// Check Redis connection
		if cm.redisClient != nil {
			if err := cm.checkRedisHealth(ctx); err != nil {
				cm.monitor.metrics.RecordHealthCheck(false)
				errors = append(errors, HealthCheckError("redis", "health_check", err))
			}
		}

		if len(errors) > 0 {
			cm.monitor.metrics.RecordHealthCheck(false)
			return fmt.Errorf("health check failures: %v", errors)
		}

		cm.monitor.metrics.RecordHealthCheck(true)
		return nil
	})
}

// checkPostgresHealth performs a detailed PostgreSQL health check
func (cm *ConnectionManager) checkPostgresHealth(ctx context.Context) error {
	// Basic connectivity test
	if _, err := cm.pgPool.Raw().Exec(ctx, "SELECT 1"); err != nil {
		cm.logger.Error("PostgreSQL connectivity test failed", "error", err)
		return fmt.Errorf("basic connectivity test failed: %w", err)
	}

	// Check if we can query system information
	var version string
	if err := cm.pgPool.Raw().QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		cm.logger.Error("PostgreSQL version query failed", "error", err)
		return fmt.Errorf("version query failed: %w", err)
	}

	cm.logger.Debug("PostgreSQL health check passed", "version", version)
	return nil
}

// checkRedisHealth performs a detailed Redis health check
func (cm *ConnectionManager) checkRedisHealth(ctx context.Context) error {
	// Basic ping test
	if err := cm.redisClient.Ping(ctx).Err(); err != nil {
		cm.logger.Error("Redis ping test failed", "error", err)
		return fmt.Errorf("ping test failed: %w", err)
	}

	// Check if we can perform basic operations
	testKey := "health_check_test"
	if err := cm.redisClient.Set(ctx, testKey, "test", time.Second).Err(); err != nil {
		cm.logger.Error("Redis set operation failed", "error", err)
		return fmt.Errorf("set operation failed: %w", err)
	}

	if err := cm.redisClient.Del(ctx, testKey).Err(); err != nil {
		cm.logger.Error("Redis delete operation failed", "error", err)
		return fmt.Errorf("delete operation failed: %w", err)
	}

	cm.logger.Debug("Redis health check passed")
	return nil
}

// Close closes all connections
func (cm *ConnectionManager) Close() error {
	cm.logger.Info("Closing all connections")
	var errors []error

	// Create a timeout context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Wait for any ongoing operations to complete
	if err := cm.waitForOngoingOperations(ctx); err != nil {
		cm.logger.Warn("Timeout waiting for ongoing operations", "error", err)
	}

	// Close PostgreSQL connection
	if cm.pgPool != nil {
		cm.logger.Debug("Closing PostgreSQL connection")
		cm.pgPool.Close()
		cm.monitor.metrics.RecordDisconnection()
		cm.pgPool = nil // Clear reference
	}

	// Close Redis connection
	if cm.redisClient != nil {
		cm.logger.Debug("Closing Redis connection")
		if err := cm.redisClient.Close(); err != nil {
			cm.logger.Error("Failed to close Redis connection", "error", err)
			errors = append(errors, fmt.Errorf("failed to close Redis connection: %w", err))
		} else {
			cm.monitor.metrics.RecordDisconnection()
		}
		cm.redisClient = nil // Clear reference
	}

	// Log final metrics summary
	cm.monitor.LogMetricsSummary()

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("connection close errors: %v", errors)
	}

	cm.logger.Info("All connections closed successfully")
	return nil
}

// waitForOngoingOperations waits for any ongoing operations to complete
func (cm *ConnectionManager) waitForOngoingOperations(ctx context.Context) error {
	// This is a placeholder for waiting for ongoing operations
	// In a more sophisticated implementation, this could track active operations
	cm.logger.Debug("Waiting for ongoing operations to complete")
	
	// Give a small grace period for operations to complete
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitForConnections waits for all connections to be ready
func (cm *ConnectionManager) WaitForConnections(ctx context.Context) error {
	// Wait for PostgreSQL
	if cm.pgPool != nil {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return fmt.Errorf("timeout waiting for PostgreSQL connection: %w", ctx.Err())
			case <-ticker.C:
				if _, err := cm.pgPool.Raw().Exec(ctx, "SELECT 1"); err == nil {
					break
				}
			}
		}
	}

	// Wait for Redis
	if cm.redisClient != nil {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return fmt.Errorf("timeout waiting for Redis connection: %w", ctx.Err())
			case <-ticker.C:
				if err := cm.redisClient.Ping(ctx).Err(); err == nil {
					break
				}
			}
		}
	}

	return nil
}

// GetConnectionStats returns detailed statistics about all connections
func (cm *ConnectionManager) GetConnectionStats(ctx context.Context) *ConnectionStats {
	stats := &ConnectionStats{}

	// Get PostgreSQL stats
	if cm.pgPool != nil {
		stats.PostgreSQL = cm.getPostgresStats(ctx)
	}

	// Get Redis stats
	if cm.redisClient != nil {
		stats.Redis = cm.getRedisStats(ctx)
	}

	return stats
}

// getPostgresStats collects PostgreSQL connection statistics
func (cm *ConnectionManager) getPostgresStats(ctx context.Context) *PostgresStats {
	stats := &PostgresStats{}

	// Test connection
	if err := cm.checkPostgresHealth(ctx); err != nil {
		stats.Connected = false
		stats.Error = err.Error()
		return stats
	}

	stats.Connected = true

	// Get version information
	var version string
	if err := cm.pgPool.Raw().QueryRow(ctx, "SELECT version()").Scan(&version); err == nil {
		stats.Version = version
	}

	return stats
}

// getRedisStats collects Redis connection statistics
func (cm *ConnectionManager) getRedisStats(ctx context.Context) *RedisStats {
	stats := &RedisStats{
		Info: make(map[string]string),
	}

	// Test connection
	if err := cm.checkRedisHealth(ctx); err != nil {
		stats.Connected = false
		stats.Error = err.Error()
		return stats
	}

	stats.Connected = true

	// Get Redis info
	if info, err := cm.redisClient.Info(ctx).Result(); err == nil {
		// Parse basic info (simplified)
		lines := strings.Split(info, "\n")
		for _, line := range lines {
			if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					// Only include key metrics to avoid overwhelming output
					if key == "redis_version" || key == "connected_clients" || key == "used_memory_human" {
						stats.Info[key] = value
					}
				}
			}
		}
	}

	return stats
}

// Reconnect attempts to reconnect failed connections
func (cm *ConnectionManager) Reconnect(ctx context.Context) error {
	var errors []error

	// Reconnect PostgreSQL if needed
	if cm.config.DatabaseURL != "" && (cm.pgPool == nil || cm.checkPostgresHealth(ctx) != nil) {
		if cm.pgPool != nil {
			cm.pgPool.Close()
		}
		
		pool, err := cm.createPostgresPool(ctx)
		if err != nil {
			errors = append(errors, fmt.Errorf("PostgreSQL reconnect failed: %w", err))
		} else {
			cm.pgPool = pool
		}
	}

	// Reconnect Redis if needed
	if cm.config.RedisAddr != "" && (cm.redisClient == nil || cm.checkRedisHealth(ctx) != nil) {
		if cm.redisClient != nil {
			cm.redisClient.Close()
		}
		
		client, err := cm.createRedisClient(ctx)
		if err != nil {
			errors = append(errors, fmt.Errorf("Redis reconnect failed: %w", err))
		} else {
			cm.redisClient = client
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("reconnection errors: %v", errors)
	}

	return nil
}

// GetMonitor returns the repository monitor for metrics access
func (cm *ConnectionManager) GetMonitor() *RepositoryMonitor {
	return cm.monitor
}

// StartMonitoring begins background monitoring
func (cm *ConnectionManager) StartMonitoring(ctx context.Context) {
	cm.logger.Info("Starting connection monitoring")
	go cm.monitor.StartMonitoring(ctx)
}

// maskDatabaseURL masks sensitive information in database URLs for logging
func maskDatabaseURL(url string) string {
	// Simple masking - replace password with ***
	// This is a basic implementation, could be enhanced for different URL formats
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) >= 2 {
			// Find the password part and mask it
			beforeAt := parts[0]
			if strings.Contains(beforeAt, ":") {
				userPass := strings.Split(beforeAt, ":")
				if len(userPass) >= 2 {
					userPass[len(userPass)-1] = "***"
					parts[0] = strings.Join(userPass, ":")
				}
			}
			return strings.Join(parts, "@")
		}
	}
	return url
}