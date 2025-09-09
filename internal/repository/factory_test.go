package repository

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test default values
	if config.Environment != "development" {
		t.Errorf("Expected default environment to be 'development', got %s", config.Environment)
	}

	if config.TestMode != false {
		t.Errorf("Expected default test mode to be false, got %v", config.TestMode)
	}

	if config.MaxConnections != 25 {
		t.Errorf("Expected default max connections to be 25, got %d", config.MaxConnections)
	}

	if config.MaxIdleConnections != 5 {
		t.Errorf("Expected default max idle connections to be 5, got %d", config.MaxIdleConnections)
	}

	if config.ConnectionTimeout != 30*time.Second {
		t.Errorf("Expected default connection timeout to be 30s, got %v", config.ConnectionTimeout)
	}

	if config.TenantCacheTTL != 15*time.Minute {
		t.Errorf("Expected default tenant cache TTL to be 15m, got %v", config.TenantCacheTTL)
	}

	if config.AssignmentCacheTTL != 10*time.Minute {
		t.Errorf("Expected default assignment cache TTL to be 10m, got %v", config.AssignmentCacheTTL)
	}

	if config.TokenCacheTTL != 30*time.Minute {
		t.Errorf("Expected default token cache TTL to be 30m, got %v", config.TokenCacheTTL)
	}

	if config.RedisDB != 0 {
		t.Errorf("Expected default Redis DB to be 0, got %d", config.RedisDB)
	}
}

func TestRepositoryConfigValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   *RepositoryConfig
		wantErr  bool
		expected *RepositoryConfig
	}{
		{
			name: "valid config",
			config: &RepositoryConfig{
				Environment:        "production",
				MaxConnections:     10,
				MaxIdleConnections: 2,
				ConnectionTimeout:  10 * time.Second,
				TenantCacheTTL:     5 * time.Minute,
			},
			wantErr: false,
			expected: &RepositoryConfig{
				Environment:        "production",
				MaxConnections:     10,
				MaxIdleConnections: 2,
				ConnectionTimeout:  10 * time.Second,
				TenantCacheTTL:     5 * time.Minute,
				AssignmentCacheTTL: 10 * time.Minute, // default
				TokenCacheTTL:      30 * time.Minute, // default
			},
		},
		{
			name: "empty environment gets default",
			config: &RepositoryConfig{
				Environment: "",
			},
			wantErr: false,
			expected: &RepositoryConfig{
				Environment:        "development", // default
				MaxConnections:     25,            // default
				MaxIdleConnections: 5,             // default
				ConnectionTimeout:  30 * time.Second,
				TenantCacheTTL:     15 * time.Minute,
				AssignmentCacheTTL: 10 * time.Minute,
				TokenCacheTTL:      30 * time.Minute,
			},
		},
		{
			name: "zero values get defaults",
			config: &RepositoryConfig{
				MaxConnections:     0,
				MaxIdleConnections: 0,
				ConnectionTimeout:  0,
				TenantCacheTTL:     0,
			},
			wantErr: false,
			expected: &RepositoryConfig{
				Environment:        "development", // default
				MaxConnections:     25,            // default
				MaxIdleConnections: 5,             // default
				ConnectionTimeout:  30 * time.Second,
				TenantCacheTTL:     15 * time.Minute,
				AssignmentCacheTTL: 10 * time.Minute,
				TokenCacheTTL:      30 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RepositoryConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check that defaults were applied correctly
			if tt.config.Environment != tt.expected.Environment {
				t.Errorf("Expected environment %s, got %s", tt.expected.Environment, tt.config.Environment)
			}
			if tt.config.MaxConnections != tt.expected.MaxConnections {
				t.Errorf("Expected max connections %d, got %d", tt.expected.MaxConnections, tt.config.MaxConnections)
			}
			if tt.config.MaxIdleConnections != tt.expected.MaxIdleConnections {
				t.Errorf("Expected max idle connections %d, got %d", tt.expected.MaxIdleConnections, tt.config.MaxIdleConnections)
			}
			if tt.config.ConnectionTimeout != tt.expected.ConnectionTimeout {
				t.Errorf("Expected connection timeout %v, got %v", tt.expected.ConnectionTimeout, tt.config.ConnectionTimeout)
			}
			if tt.config.TenantCacheTTL != tt.expected.TenantCacheTTL {
				t.Errorf("Expected tenant cache TTL %v, got %v", tt.expected.TenantCacheTTL, tt.config.TenantCacheTTL)
			}
		})
	}
}

func TestRepositoryConfigEnvironmentDetection(t *testing.T) {
	tests := []struct {
		name        string
		config      *RepositoryConfig
		isProduction bool
		isDevelopment bool
		isTest      bool
	}{
		{
			name: "production with redis",
			config: &RepositoryConfig{
				Environment: "production",
				RedisAddr:   "localhost:6379",
			},
			isProduction:  true,
			isDevelopment: false,
			isTest:        false,
		},
		{
			name: "production without redis",
			config: &RepositoryConfig{
				Environment: "production",
				RedisAddr:   "",
			},
			isProduction:  false,
			isDevelopment: true,
			isTest:        false,
		},
		{
			name: "development",
			config: &RepositoryConfig{
				Environment: "development",
			},
			isProduction:  false,
			isDevelopment: true,
			isTest:        false,
		},
		{
			name: "test environment",
			config: &RepositoryConfig{
				Environment: "test",
			},
			isProduction:  false,
			isDevelopment: false,
			isTest:        true,
		},
		{
			name: "test mode enabled",
			config: &RepositoryConfig{
				Environment: "development",
				TestMode:    true,
			},
			isProduction:  false,
			isDevelopment: false,
			isTest:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsProduction(); got != tt.isProduction {
				t.Errorf("IsProduction() = %v, want %v", got, tt.isProduction)
			}
			if got := tt.config.IsDevelopment(); got != tt.isDevelopment {
				t.Errorf("IsDevelopment() = %v, want %v", got, tt.isDevelopment)
			}
			if got := tt.config.IsTest(); got != tt.isTest {
				t.Errorf("IsTest() = %v, want %v", got, tt.isTest)
			}
		})
	}
}

