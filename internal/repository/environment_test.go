package repository

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestEnvironmentDetector_DetectEnvironment(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{"PS_ENVIRONMENT", "GO_ENV", "NODE_ENV", "ENVIRONMENT", "PS_TEST_MODE", "CI", "TESTING"}
	
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	
	// Restore environment after test
	defer func() {
		for _, env := range envVars {
			if val, exists := originalEnv[env]; exists && val != "" {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	tests := []struct {
		name           string
		config         *RepositoryConfig
		envVars        map[string]string
		expectedEnv    string
		expectError    bool
	}{
		{
			name: "explicit production environment",
			config: &RepositoryConfig{
				DatabaseURL: "postgres://localhost:5432/test",
				RedisAddr:   "localhost:6379",
			},
			envVars: map[string]string{
				"PS_ENVIRONMENT": "production",
			},
			expectedEnv: "production",
			expectError: false,
		},
		{
			name: "explicit test environment",
			config: &RepositoryConfig{},
			envVars: map[string]string{
				"PS_ENVIRONMENT": "test",
			},
			expectedEnv: "test",
			expectError: false,
		},
		{
			name: "auto-detect production (database + redis)",
			config: &RepositoryConfig{
				DatabaseURL: "postgres://localhost:5432/test",
				RedisAddr:   "localhost:6379",
			},
			envVars:     map[string]string{},
			expectedEnv: "production",
			expectError: false,
		},
		{
			name: "auto-detect development (database only)",
			config: &RepositoryConfig{
				DatabaseURL: "postgres://localhost:5432/test",
			},
			envVars:     map[string]string{},
			expectedEnv: "development",
			expectError: false,
		},
		{
			name: "auto-detect test (no dependencies)",
			config: &RepositoryConfig{},
			envVars:     map[string]string{},
			expectedEnv: "test",
			expectError: false,
		},
		{
			name: "test mode via environment variable",
			config: &RepositoryConfig{
				DatabaseURL: "postgres://localhost:5432/test",
			},
			envVars: map[string]string{
				"PS_TEST_MODE": "true",
			},
			expectedEnv: "test",
			expectError: false,
		},
		{
			name: "test mode via CI environment",
			config: &RepositoryConfig{
				DatabaseURL: "postgres://localhost:5432/test",
			},
			envVars: map[string]string{
				"CI": "true",
			},
			expectedEnv: "test",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			detector := NewEnvironmentDetector(tt.config)
			env, err := detector.DetectEnvironment(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if env != tt.expectedEnv {
				t.Errorf("Expected environment %s, got %s", tt.expectedEnv, env)
			}

			// Clean up environment variables for next test
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestEnvironmentDetector_ValidateEnvironmentConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      *RepositoryConfig
		environment string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid production config",
			config: &RepositoryConfig{
				DatabaseURL:    "postgres://localhost:5432/test",
				RedisAddr:      "localhost:6379",
				TenantCacheTTL: 15 * time.Minute,
			},
			environment: "production",
			expectError: false,
		},
		{
			name: "production missing database",
			config: &RepositoryConfig{
				RedisAddr: "localhost:6379",
			},
			environment: "production",
			expectError: true,
			errorMsg:    "database URL is required",
		},
		{
			name: "production missing redis",
			config: &RepositoryConfig{
				DatabaseURL:    "postgres://localhost:5432/test",
				TenantCacheTTL: 15 * time.Minute,
			},
			environment: "production",
			expectError: true,
			errorMsg:    "Redis address is recommended",
		},
		{
			name: "valid development config",
			config: &RepositoryConfig{
				DatabaseURL: "postgres://localhost:5432/test",
			},
			environment: "development",
			expectError: false,
		},
		{
			name: "development missing database",
			config: &RepositoryConfig{},
			environment: "development",
			expectError: true,
			errorMsg:    "database URL is required",
		},
		{
			name: "valid test config (minimal)",
			config: &RepositoryConfig{},
			environment: "test",
			expectError: false,
		},
		{
			name: "test config with invalid database URL",
			config: &RepositoryConfig{
				DatabaseURL: "invalid-url",
			},
			environment: "test",
			expectError: true,
			errorMsg:    "invalid database URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewEnvironmentDetector(tt.config)
			err := detector.ValidateEnvironmentConfiguration(context.Background(), tt.environment)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestEnvironmentDetector_GetFactoryRecommendation(t *testing.T) {
	tests := []struct {
		name             string
		config           *RepositoryConfig
		expectedFactory  string
		expectedContains string
	}{
		{
			name: "database and redis available",
			config: &RepositoryConfig{
				DatabaseURL: "postgres://localhost:5432/test",
				RedisAddr:   "localhost:6379",
			},
			expectedFactory:  "production",
			expectedContains: "Redis caching",
		},
		{
			name: "database only",
			config: &RepositoryConfig{
				DatabaseURL: "postgres://localhost:5432/test",
			},
			expectedFactory:  "development",
			expectedContains: "without caching",
		},
		{
			name: "no dependencies",
			config: &RepositoryConfig{},
			expectedFactory:  "test",
			expectedContains: "In-memory",
		},
		{
			name: "redis only (unusual case)",
			config: &RepositoryConfig{
				RedisAddr: "localhost:6379",
			},
			expectedFactory:  "test",
			expectedContains: "no database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewEnvironmentDetector(tt.config)
			factoryType, reason, err := detector.GetFactoryRecommendation(context.Background())

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if factoryType != tt.expectedFactory {
				t.Errorf("Expected factory type %s, got %s", tt.expectedFactory, factoryType)
			}

			if !containsString(reason, tt.expectedContains) {
				t.Errorf("Expected reason to contain '%s', got '%s'", tt.expectedContains, reason)
			}
		})
	}
}

func TestEnvironmentDetector_URLValidation(t *testing.T) {
	detector := NewEnvironmentDetector(&RepositoryConfig{})

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"valid postgres URL", "postgres://localhost:5432/test", true},
		{"valid postgresql URL", "postgresql://localhost:5432/test", true},
		{"valid sqlite URL", "sqlite://test.db", true},
		{"valid mysql URL", "mysql://localhost:3306/test", true},
		{"invalid URL", "invalid-url", false},
		{"empty URL", "", false},
		{"http URL", "http://localhost", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isValidDatabaseURL(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for URL %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestEnvironmentDetector_RedisAddrValidation(t *testing.T) {
	detector := NewEnvironmentDetector(&RepositoryConfig{})

	tests := []struct {
		name     string
		addr     string
		expected bool
	}{
		{"valid host:port", "localhost:6379", true},
		{"valid IP:port", "127.0.0.1:6379", true},
		{"valid hostname:port", "redis.example.com:6379", true},
		{"invalid format", "localhost", false},
		{"invalid port", "localhost:abc", false},
		{"empty address", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isValidRedisAddr(tt.addr)
			if result != tt.expected {
				t.Errorf("Expected %v for address %s, got %v", tt.expected, tt.addr, result)
			}
		})
	}
}

func TestEnvironmentDetector_TestEnvironmentDetection(t *testing.T) {
	// Save original args and environment
	originalArgs := os.Args
	originalEnv := make(map[string]string)
	envVars := []string{"PS_TEST_MODE", "CI", "TESTING"}
	
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	
	defer func() {
		os.Args = originalArgs
		for _, env := range envVars {
			if val, exists := originalEnv[env]; exists && val != "" {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	tests := []struct {
		name     string
		config   *RepositoryConfig
		args     []string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "explicit test mode in config",
			config:   &RepositoryConfig{TestMode: true},
			expected: true,
		},
		{
			name:   "test mode environment variable",
			config: &RepositoryConfig{},
			envVars: map[string]string{
				"PS_TEST_MODE": "true",
			},
			expected: true,
		},
		{
			name:   "CI environment",
			config: &RepositoryConfig{},
			envVars: map[string]string{
				"CI": "true",
			},
			expected: true,
		},
		{
			name:   "TESTING environment",
			config: &RepositoryConfig{},
			envVars: map[string]string{
				"TESTING": "1",
			},
			expected: true,
		},
		{
			name:     "test binary name",
			config:   &RepositoryConfig{},
			args:     []string{"myapp.test"},
			expected: true,
		},
		{
			name:     "go test flag",
			config:   &RepositoryConfig{},
			args:     []string{"myapp", "-test.v"},
			expected: true,
		},
		{
			name:     "normal execution",
			config:   &RepositoryConfig{},
			args:     []string{"myapp"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Set args if provided
			if tt.args != nil {
				os.Args = tt.args
			}

			detector := NewEnvironmentDetector(tt.config)
			result := detector.isTestEnvironment()

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}

			// Clean up environment variables
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}