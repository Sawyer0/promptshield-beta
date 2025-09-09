package repository

import (
	"context"
	"testing"
	"time"
)

func TestNewDevelopmentRepositoryFactory(t *testing.T) {
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
			errMsg:  "development factory requires PostgreSQL connection",
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
		{
			name: "config with Redis address (should be ignored)",
			config: &RepositoryConfig{
				DatabaseURL: "",
				RedisAddr:   "localhost:6379",
			},
			wantErr: true,
			errMsg:  "development factory requires PostgreSQL connection",
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

			factory, err := NewDevelopmentRepositoryFactory(tt.config, cm)

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

func TestDevelopmentRepositoryFactoryMethods(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}

	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	// This should fail because we don't have a database connection
	_, err = NewDevelopmentRepositoryFactory(config, cm)
	if err == nil {
		t.Fatal("Expected error creating development factory without database")
	}

	// Test with a mock connection manager that has postgres
	mockConfig := &RepositoryConfig{
		DatabaseURL:        "mock-db",
		RedisAddr:          "localhost:6379", // Should be ignored in development
		TenantCacheTTL:     15 * time.Minute,
		AssignmentCacheTTL: 10 * time.Minute,
		TokenCacheTTL:      30 * time.Minute,
	}

	// We can't actually test with real connections in unit tests,
	// but we can test the structure and error handling
	t.Logf("Mock config would have database URL: %s", mockConfig.DatabaseURL)
}

func TestDevelopmentFactoryStats(t *testing.T) {
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
	factory := &DevelopmentRepositoryFactory{
		config:            config,
		connectionManager: cm,
	}

	// Test stats collection
	stats := factory.GetStats(context.Background())
	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	if stats.FactoryType != "development" {
		t.Errorf("Expected factory type 'development', got %s", stats.FactoryType)
	}

	if stats.HasRedisCache != false {
		t.Error("Expected HasRedisCache to be false for development factory")
	}

	if stats.DebugMode != true {
		t.Error("Expected DebugMode to be true for development factory")
	}

	if stats.RepositoryCount != 0 {
		t.Errorf("Expected repository count 0, got %d", stats.RepositoryCount)
	}

	// With empty database URL, the masked URL should be empty too
	if config.DatabaseURL == "" && stats.DatabaseURL != "" {
		t.Error("Expected empty database URL in stats when config has empty URL")
	}
}

func TestDevelopmentFactoryValidation(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}

	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	factory := &DevelopmentRepositoryFactory{
		config:            config,
		connectionManager: cm,
	}

	// Test validation with no repositories initialized
	err = factory.ValidateRepositories(context.Background())
	if err == nil {
		t.Error("Expected validation error with no repositories initialized")
	}

	// Test reconnect (should not panic)
	err = factory.Reconnect(context.Background())
	if err == nil {
		t.Error("Expected reconnect error without database connection")
	}
}

func TestDevelopmentFactoryRepositoryAccess(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}

	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer cm.Close()

	// Since development factory requires PostgreSQL, we can't create it without a database
	// This test verifies that the factory creation fails appropriately
	_, err = NewDevelopmentRepositoryFactory(config, cm)
	if err == nil {
		t.Error("Expected error when creating development factory without database")
	} else {
		t.Logf("Factory creation failed as expected: %v", err)
	}
}

func TestDevelopmentFactoryDebugging(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL:        "",
		RedisAddr:          "",
		MaxConnections:     10,
		MaxIdleConnections: 2,
		ConnectionTimeout:  5 * time.Second,
	}

	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	factory := &DevelopmentRepositoryFactory{
		config:            config,
		connectionManager: cm,
	}

	// Test debugging methods (should not panic)
	factory.LogRepositoryUsage(context.Background())
	factory.EnableVerboseLogging()

	// Test URL masking
	masked := factory.maskDatabaseURL("postgres://user:pass@localhost:5432/db")
	if masked == "postgres://user:pass@localhost:5432/db" {
		t.Error("Expected URL to be masked")
	}

	// Test short URL masking
	shortMasked := factory.maskDatabaseURL("short")
	if shortMasked != "***" {
		t.Errorf("Expected short URL to be masked as '***', got '%s'", shortMasked)
	}
}

// Use the containsString function from production_factory_test.go