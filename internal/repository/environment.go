package repository

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvironmentDetector provides advanced environment detection capabilities
type EnvironmentDetector struct {
	config *RepositoryConfig
}

// NewEnvironmentDetector creates a new environment detector
func NewEnvironmentDetector(config *RepositoryConfig) *EnvironmentDetector {
	return &EnvironmentDetector{config: config}
}

// DetectEnvironment automatically detects the runtime environment based on available dependencies and configuration
func (ed *EnvironmentDetector) DetectEnvironment(ctx context.Context) (string, error) {
	// First check explicit environment variables
	if env := ed.getExplicitEnvironment(); env != "" {
		return env, nil
	}

	// Check for explicit test mode indicators (these override dependency detection)
	if ed.isExplicitTestMode() {
		return "test", nil
	}

	// Auto-detect based on available dependencies
	hasDatabase := ed.hasDatabaseConnection(ctx)
	hasRedis := ed.hasRedisConnection(ctx)

	switch {
	case hasDatabase && hasRedis:
		return "production", nil
	case hasDatabase && !hasRedis:
		return "development", nil
	case !hasDatabase && !hasRedis:
		return "test", nil
	default:
		// Redis without database is unusual, default to development
		return "development", nil
	}
}

// ValidateEnvironmentConfiguration validates that the detected environment has the required dependencies
func (ed *EnvironmentDetector) ValidateEnvironmentConfiguration(ctx context.Context, environment string) error {
	switch environment {
	case "production":
		return ed.validateProductionEnvironment(ctx)
	case "development":
		return ed.validateDevelopmentEnvironment(ctx)
	case "test":
		return ed.validateTestEnvironment(ctx)
	default:
		return fmt.Errorf("unknown environment: %s", environment)
	}
}

// GetFactoryRecommendation recommends the best factory type based on available dependencies
func (ed *EnvironmentDetector) GetFactoryRecommendation(ctx context.Context) (string, string, error) {
	hasDatabase := ed.hasDatabaseConnection(ctx)
	hasRedis := ed.hasRedisConnection(ctx)

	switch {
	case hasDatabase && hasRedis:
		return "production", "PostgreSQL with Redis caching for optimal performance", nil
	case hasDatabase && !hasRedis:
		return "development", "PostgreSQL without caching for simpler debugging", nil
	case !hasDatabase && !hasRedis:
		return "test", "In-memory repositories for testing", nil
	case !hasDatabase && hasRedis:
		return "test", "Redis available but no database - using test factory", nil
	default:
		return "development", "Default fallback to development factory", nil
	}
}

// getExplicitEnvironment checks for explicitly set environment variables
func (ed *EnvironmentDetector) getExplicitEnvironment() string {
	// Check PromptShield-specific environment variable first
	if env := os.Getenv("PS_ENVIRONMENT"); env != "" {
		return strings.ToLower(env)
	}

	// Check common environment variables
	if env := os.Getenv("GO_ENV"); env != "" {
		return strings.ToLower(env)
	}

	if env := os.Getenv("NODE_ENV"); env != "" {
		return strings.ToLower(env)
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return strings.ToLower(env)
	}

	return ""
}

// isExplicitTestMode checks for explicit test mode indicators (not including binary/flag detection)
func (ed *EnvironmentDetector) isExplicitTestMode() bool {
	// Check explicit test mode in config
	if ed.config.TestMode {
		return true
	}

	// Check test mode environment variable
	if testMode := os.Getenv("PS_TEST_MODE"); strings.ToLower(testMode) == "true" {
		return true
	}

	// Check CI/CD environment indicators
	if os.Getenv("CI") != "" || os.Getenv("TESTING") != "" {
		return true
	}

	return false
}

// isTestEnvironment checks for all test environment indicators (including binary/flag detection)
func (ed *EnvironmentDetector) isTestEnvironment() bool {
	// Check explicit test mode first
	if ed.isExplicitTestMode() {
		return true
	}

	// Check if running in test binary
	if strings.Contains(os.Args[0], "test") || strings.HasSuffix(os.Args[0], ".test") {
		return true
	}

	// Check for Go test flags
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}

	return false
}

// hasDatabaseConnection checks if a database connection is available
func (ed *EnvironmentDetector) hasDatabaseConnection(ctx context.Context) bool {
	if ed.config.DatabaseURL == "" {
		return false
	}

	// Try to parse the database URL to validate format
	if !ed.isValidDatabaseURL(ed.config.DatabaseURL) {
		return false
	}

	// For quick detection, we don't actually connect, just check if URL is provided and valid
	// Actual connection testing is done during factory creation
	return true
}

// hasRedisConnection checks if a Redis connection is available
func (ed *EnvironmentDetector) hasRedisConnection(ctx context.Context) bool {
	if ed.config.RedisAddr == "" {
		return false
	}

	// Try to parse the Redis address to validate format
	if !ed.isValidRedisAddr(ed.config.RedisAddr) {
		return false
	}

	// For quick detection, we don't actually connect, just check if address is provided and valid
	// Actual connection testing is done during factory creation
	return true
}

// validateProductionEnvironment validates production environment requirements
func (ed *EnvironmentDetector) validateProductionEnvironment(ctx context.Context) error {
	var errors []string

	// Production requires database
	if ed.config.DatabaseURL == "" {
		errors = append(errors, "database URL is required for production environment (set PS_PG_DSN)")
	} else if !ed.isValidDatabaseURL(ed.config.DatabaseURL) {
		errors = append(errors, "invalid database URL format")
	}

	// Production should have Redis for optimal performance
	if ed.config.RedisAddr == "" {
		errors = append(errors, "Redis address is recommended for production environment (set PS_REDIS_ADDR)")
	} else if !ed.isValidRedisAddr(ed.config.RedisAddr) {
		errors = append(errors, "invalid Redis address format")
	}

	// Validate cache TTL settings
	if ed.config.TenantCacheTTL <= 0 {
		errors = append(errors, "tenant cache TTL must be positive for production")
	}

	if len(errors) > 0 {
		return fmt.Errorf("production environment validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateDevelopmentEnvironment validates development environment requirements
func (ed *EnvironmentDetector) validateDevelopmentEnvironment(ctx context.Context) error {
	var errors []string

	// Development requires database
	if ed.config.DatabaseURL == "" {
		errors = append(errors, "database URL is required for development environment (set PS_PG_DSN)")
	} else if !ed.isValidDatabaseURL(ed.config.DatabaseURL) {
		errors = append(errors, "invalid database URL format")
	}

	// Redis is optional for development
	if ed.config.RedisAddr != "" && !ed.isValidRedisAddr(ed.config.RedisAddr) {
		errors = append(errors, "invalid Redis address format")
	}

	if len(errors) > 0 {
		return fmt.Errorf("development environment validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateTestEnvironment validates test environment requirements
func (ed *EnvironmentDetector) validateTestEnvironment(ctx context.Context) error {
	// Test environment is very flexible - no strict requirements
	// Database and Redis are optional for testing

	if ed.config.DatabaseURL != "" && !ed.isValidDatabaseURL(ed.config.DatabaseURL) {
		return fmt.Errorf("test environment validation failed: invalid database URL format")
	}

	if ed.config.RedisAddr != "" && !ed.isValidRedisAddr(ed.config.RedisAddr) {
		return fmt.Errorf("test environment validation failed: invalid Redis address format")
	}

	return nil
}

// isValidDatabaseURL performs basic validation of database URL format
func (ed *EnvironmentDetector) isValidDatabaseURL(url string) bool {
	if url == "" {
		return false
	}

	// Check for common database URL patterns
	validPrefixes := []string{"postgres://", "postgresql://", "sqlite://", "mysql://"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}

	return false
}

// isValidRedisAddr performs basic validation of Redis address format
func (ed *EnvironmentDetector) isValidRedisAddr(addr string) bool {
	if addr == "" {
		return false
	}

	// Try to parse as host:port
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	// Validate that host is not empty
	if host == "" {
		return false
	}

	// Validate that port is a valid number
	if _, err := strconv.Atoi(port); err != nil {
		return false
	}

	return true
}

// TestDatabaseConnection attempts to test the database connection
func (ed *EnvironmentDetector) TestDatabaseConnection(ctx context.Context) error {
	if ed.config.DatabaseURL == "" {
		return fmt.Errorf("no database URL configured")
	}

	// Create a temporary connection manager to test the connection
	tempConfig := &RepositoryConfig{
		DatabaseURL:       ed.config.DatabaseURL,
		ConnectionTimeout: 5 * time.Second, // Short timeout for testing
		MaxConnections:    1,                // Minimal connection for testing
	}

	cm, err := NewConnectionManager(ctx, tempConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection manager: %w", err)
	}
	defer cm.Close()

	// Test the connection with health check
	return cm.HealthCheck(ctx)
}

// TestRedisConnection attempts to test the Redis connection
func (ed *EnvironmentDetector) TestRedisConnection(ctx context.Context) error {
	if ed.config.RedisAddr == "" {
		return fmt.Errorf("no Redis address configured")
	}

	// Create a temporary connection manager to test the connection
	tempConfig := &RepositoryConfig{
		RedisAddr:         ed.config.RedisAddr,
		RedisPassword:     ed.config.RedisPassword,
		RedisDB:           ed.config.RedisDB,
		ConnectionTimeout: 5 * time.Second, // Short timeout for testing
	}

	cm, err := NewConnectionManager(ctx, tempConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection manager: %w", err)
	}
	defer cm.Close()

	// Test the connection with health check
	return cm.HealthCheck(ctx)
}