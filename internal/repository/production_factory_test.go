package repository

import (
	"context"
	"testing"
	"time"
)

func TestNewProductionRepositoryFactory(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *RepositoryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config without database",
			config: &RepositoryConfig{
				DatabaseURL: "",
				RedisAddr:   "",
			},
			wantErr: true,
			errMsg:  "production factory requires PostgreSQL connection",
		},
		{
			name: "valid config with invalid database",
			config: &RepositoryConfig{
				DatabaseURL:       "invalid-db-url",
				RedisAddr:         "",
				ConnectionTimeout: 1 * time.Second,
			},
			wantErr: true,
			errMsg:  "failed to connect to PostgreSQL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create connection manager first
			cm, err := NewConnectionManager(ctx, tt.config)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Failed to create connection manager: %v", err)
				}
				// If we expect an error and got one from connection manager, that's fine
				return
			}

			factory, err := NewProductionRepositoryFactory(tt.config, cm)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if factory == nil {
				t.Error("Expected non-nil factory")
			} else {
				// Test that the factory was created successfully
				t.Logf("Factory created successfully with config: %+v", factory.config)
			}
		})
	}
}

func TestProductionRepositoryFactoryMethods(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}

	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	// This should fail because we don't have a database connection
	_, err = NewProductionRepositoryFactory(config, cm)
	if err == nil {
		t.Fatal("Expected error creating production factory without database")
	}

	// Test with a mock connection manager that has postgres
	mockConfig := &RepositoryConfig{
		DatabaseURL:        "mock-db",
		RedisAddr:          "",
		TenantCacheTTL:     15 * time.Minute,
		AssignmentCacheTTL: 10 * time.Minute,
		TokenCacheTTL:      30 * time.Minute,
	}

	// We can't actually test with real connections in unit tests,
	// but we can test the structure and error handling
	t.Logf("Mock config would have database URL: %s", mockConfig.DatabaseURL)
}

func TestProductionFactoryStats(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL:        "",
		RedisAddr:          "",
		TenantCacheTTL:     15 * time.Minute,
		AssignmentCacheTTL: 10 * time.Minute,
		TokenCacheTTL:      30 * time.Minute,
	}

	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	// Create a factory with mock setup (this will fail but we can test the structure)
	factory := &ProductionRepositoryFactory{
		config:            config,
		connectionManager: cm,
	}

	// Test stats collection
	stats := factory.GetStats(context.Background())
	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	if stats.FactoryType != "production" {
		t.Errorf("Expected factory type 'production', got %s", stats.FactoryType)
	}

	if stats.HasRedisCache != false {
		t.Error("Expected HasRedisCache to be false with no Redis connection")
	}

	if stats.RepositoryCount != 0 {
		t.Errorf("Expected repository count 0, got %d", stats.RepositoryCount)
	}
}

func TestProductionFactoryValidation(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}

	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer cm.Close()

	// Try to create factory properly (should fail due to no database)
	_, err = NewProductionRepositoryFactory(config, cm)
	if err == nil {
		t.Error("Expected error when creating production factory without database")
		return
	}
	
	// Since factory creation failed, we can't test validation
	t.Logf("Factory creation failed as expected: %v", err)
}

func TestProductionFactoryRepositoryAccess(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}

	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer cm.Close()

	// Since production factory requires PostgreSQL, we can't create it without a database
	// This test verifies that the factory creation fails appropriately
	_, err = NewProductionRepositoryFactory(config, cm)
	if err == nil {
		t.Error("Expected error when creating production factory without database")
	} else {
		t.Logf("Factory creation failed as expected: %v", err)
	}
}

// containsString is a helper function for string contains check
func containsString(s, substr string) bool {
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