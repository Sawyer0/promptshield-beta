package repository

import (
	"context"
	"os"
	"testing"
)

func TestNewFactoryBuilder(t *testing.T) {
	builder := NewFactoryBuilder()

	if builder == nil {
		t.Fatal("NewFactoryBuilder() returned nil")
	}

	if builder.config == nil {
		t.Fatal("Builder config is nil")
	}

	if builder.customRepositories == nil {
		t.Fatal("Builder customRepositories is nil")
	}

	// Check default config values
	if builder.config.Environment != "development" {
		t.Errorf("Expected default environment 'development', got %s", builder.config.Environment)
	}
}

func TestFactoryBuilderWithConfig(t *testing.T) {
	customConfig := &RepositoryConfig{
		Environment:   "production",
		DatabaseURL:   "postgres://test",
		RedisAddr:     "localhost:6379",
		MaxConnections: 50,
	}

	builder := NewFactoryBuilder().WithConfig(customConfig)

	if builder.config != customConfig {
		t.Error("WithConfig() did not set the custom config")
	}

	// Test with nil config (should not change existing config)
	originalConfig := builder.config
	builder.WithConfig(nil)
	if builder.config != originalConfig {
		t.Error("WithConfig(nil) should not change existing config")
	}
}

func TestFactoryBuilderFluentInterface(t *testing.T) {
	builder := NewFactoryBuilder().
		WithDatabaseURL("postgres://test").
		WithRedis("localhost:6379", "password", 1).
		WithEnvironment("production").
		WithTestMode(true).
		WithCustomRepository("test", "mock")

	// Check database URL
	if builder.config.DatabaseURL != "postgres://test" {
		t.Errorf("Expected database URL 'postgres://test', got %s", builder.config.DatabaseURL)
	}

	// Check Redis settings
	if builder.config.RedisAddr != "localhost:6379" {
		t.Errorf("Expected Redis addr 'localhost:6379', got %s", builder.config.RedisAddr)
	}
	if builder.config.RedisPassword != "password" {
		t.Errorf("Expected Redis password 'password', got %s", builder.config.RedisPassword)
	}
	if builder.config.RedisDB != 1 {
		t.Errorf("Expected Redis DB 1, got %d", builder.config.RedisDB)
	}

	// Check environment
	if builder.config.Environment != "production" {
		t.Errorf("Expected environment 'production', got %s", builder.config.Environment)
	}

	// Check test mode
	if !builder.config.TestMode {
		t.Error("Expected test mode to be true")
	}

	// Check custom repository
	if repo, exists := builder.customRepositories["test"]; !exists || repo != "mock" {
		t.Error("Custom repository was not set correctly")
	}
}

func TestFactoryBuilderBuildValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		setupFunc func() *FactoryBuilder
		wantErr   bool
		errMsg    string
	}{
		{
			name: "test environment - should succeed",
			setupFunc: func() *FactoryBuilder {
				return NewFactoryBuilder().WithTestMode(true)
			},
			wantErr: false,
		},
		{
			name: "production without database URL - should fail",
			setupFunc: func() *FactoryBuilder {
				return NewFactoryBuilder().
					WithEnvironment("production").
					WithRedis("localhost:6379", "", 0)
			},
			wantErr: true,
			errMsg:  "database URL is required",
		},
		{
			name: "development without database URL - should fail",
			setupFunc: func() *FactoryBuilder {
				// Create a builder and explicitly set development environment
				// with some configuration to prevent auto-detection
				return NewFactoryBuilder().
					WithEnvironment("development").
					WithDatabaseURL("") // Explicitly set empty database URL
			},
			wantErr: true,
			errMsg:  "database URL is required",
		},
		{
			name: "unknown environment - should fail",
			setupFunc: func() *FactoryBuilder {
				return NewFactoryBuilder().WithEnvironment("unknown")
			},
			wantErr: true,
			errMsg:  "unknown environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setupFunc()
			factory, err := builder.Build(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
				if factory != nil {
					t.Error("Expected factory to be nil when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if factory == nil {
					t.Error("Expected factory to be non-nil when no error")
				}
			}
		})
	}
}

func TestFactoryBuilderEnvironmentSelection(t *testing.T) {
	ctx := context.Background()

	// Test that test environment is selected correctly
	builder := NewFactoryBuilder().WithTestMode(true)
	factory, err := builder.Build(ctx)
	if err != nil {
		t.Fatalf("Unexpected error building test factory: %v", err)
	}
	if factory == nil {
		t.Fatal("Expected non-nil factory")
	}

	// Verify it's a test factory by checking if it implements the interface
	if _, ok := factory.(*TestRepositoryFactory); !ok {
		t.Error("Expected TestRepositoryFactory for test environment")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())))
}

func TestBuildWithAutoDetection(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{"PS_PG_DSN", "PS_REDIS_ADDR", "PS_ENVIRONMENT"}
	
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	
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
		name            string
		envVars         map[string]string
		expectError     bool
		expectedMsg     string
		expectedFactory string
	}{
		{
			name: "auto-detect test environment",
			envVars: map[string]string{
				// No database or Redis configured
			},
			expectError:     false,
			expectedFactory: "test",
		},
		{
			name: "auto-detect with explicit test mode",
			envVars: map[string]string{
				"PS_PG_DSN":    "postgres://localhost:5432/test", // Database configured
				"PS_TEST_MODE": "true",                           // But test mode is explicit
			},
			expectError:     false,
			expectedFactory: "test", // Should use test factory due to explicit test mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			factory, err := BuildWithAutoDetection(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.expectedMsg != "" && !contains(err.Error(), tt.expectedMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.expectedMsg, err.Error())
				}
				if factory != nil {
					factory.Close() // Clean up if created
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if factory == nil {
				t.Error("Expected factory to be non-nil")
				return
			}

			// Verify factory type if specified
			if tt.expectedFactory != "" {
				// We can't directly check the factory type, but we can verify it works
				if err := factory.HealthCheck(context.Background()); err != nil {
					t.Errorf("Factory health check failed: %v", err)
				}
			}

			// Clean up
			factory.Close()

			// Clean up environment variables
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestBuildWithFallback(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{"PS_PG_DSN", "PS_REDIS_ADDR", "PS_ENVIRONMENT"}
	
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	
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
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "fallback to test environment",
			envVars: map[string]string{
				// No dependencies configured - should use test factory
			},
			expectError: false,
		},
		{
			name: "fallback from invalid database",
			envVars: map[string]string{
				"PS_PG_DSN": "postgres://invalid:5432/test", // Invalid database
			},
			expectError: false, // Should fallback to test factory
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			factory, err := BuildWithFallback(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if factory != nil {
					factory.Close()
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if factory == nil {
				t.Error("Expected factory to be non-nil")
				return
			}

			// Verify factory works
			if err := factory.HealthCheck(context.Background()); err != nil {
				t.Errorf("Factory health check failed: %v", err)
			}

			// Clean up
			factory.Close()

			// Clean up environment variables
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestFactoryBuilderEnhancedEnvironmentDetection(t *testing.T) {
	tests := []struct {
		name        string
		config      *RepositoryConfig
		expectError bool
		expectedEnv string
	}{
		{
			name: "explicit environment overrides detection",
			config: &RepositoryConfig{
				Environment: "production",
				DatabaseURL: "postgres://localhost:5432/test",
				RedisAddr:   "localhost:6379",
			},
			expectError: true, // Will fail due to invalid connections
			expectedEnv: "production",
		},
		{
			name: "auto-detection with test config",
			config: &RepositoryConfig{
				Environment: "", // Will be auto-detected
			},
			expectError: false,
			expectedEnv: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewFactoryBuilder().WithConfig(tt.config)
			factory, err := builder.Build(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if factory != nil {
					factory.Close()
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if factory == nil {
				t.Error("Expected factory to be non-nil")
				return
			}

			// Verify the environment was set correctly
			if tt.config.Environment != tt.expectedEnv {
				t.Errorf("Expected environment %s, got %s", tt.expectedEnv, tt.config.Environment)
			}

			factory.Close()
		})
	}
}