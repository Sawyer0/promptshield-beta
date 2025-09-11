package repository

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// ConfigFromEnvironment creates a repository configuration from environment variables
func ConfigFromEnvironment() *RepositoryConfig {
	config := DefaultConfig()

	// Database configuration
	if dbURL := os.Getenv("PS_PG_DSN"); dbURL != "" {
		config.DatabaseURL = dbURL
	}

	// Redis configuration
	if redisAddr := os.Getenv("PS_REDIS_ADDR"); redisAddr != "" {
		config.RedisAddr = redisAddr
	}
	if redisPassword := os.Getenv("PS_REDIS_PASSWORD"); redisPassword != "" {
		config.RedisPassword = redisPassword
	}
	if redisDB := os.Getenv("PS_REDIS_DB"); redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			config.RedisDB = db
		}
	}

	// Environment detection
	if env := os.Getenv("PS_ENVIRONMENT"); env != "" {
		config.Environment = env
	} else if env := os.Getenv("GO_ENV"); env != "" {
		config.Environment = env
	} else {
		// Auto-detect based on other environment variables
		if config.RedisAddr != "" && config.DatabaseURL != "" {
			config.Environment = "production"
		} else if config.DatabaseURL != "" {
			config.Environment = "development"
		} else {
			config.Environment = "test"
		}
	}

	// Test mode detection
	if testMode := os.Getenv("PS_TEST_MODE"); testMode != "" {
		config.TestMode = strings.ToLower(testMode) == "true"
	}
	// Also check for common test environment indicators
	if os.Getenv("CI") != "" || os.Getenv("TESTING") != "" || strings.Contains(os.Args[0], "test") {
		config.TestMode = true
		config.Environment = "test"
	}

	// Cache TTL configuration
	if ttl := os.Getenv("PS_TENANT_CACHE_TTL"); ttl != "" {
		if duration, err := time.ParseDuration(ttl); err == nil {
			config.TenantCacheTTL = duration
		}
	}
	if ttl := os.Getenv("PS_ASSIGNMENT_CACHE_TTL"); ttl != "" {
		if duration, err := time.ParseDuration(ttl); err == nil {
			config.AssignmentCacheTTL = duration
		}
	}
	if ttl := os.Getenv("PS_TOKEN_CACHE_TTL"); ttl != "" {
		if duration, err := time.ParseDuration(ttl); err == nil {
			config.TokenCacheTTL = duration
		}
	}

	// Connection pool configuration
	if maxConn := os.Getenv("PS_MAX_CONNECTIONS"); maxConn != "" {
		if max, err := strconv.Atoi(maxConn); err == nil && max > 0 {
			config.MaxConnections = max
		}
	}
	if maxIdle := os.Getenv("PS_MAX_IDLE_CONNECTIONS"); maxIdle != "" {
		if max, err := strconv.Atoi(maxIdle); err == nil && max > 0 {
			config.MaxIdleConnections = max
		}
	}
	if timeout := os.Getenv("PS_CONNECTION_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.ConnectionTimeout = duration
		}
	}

	return config
}