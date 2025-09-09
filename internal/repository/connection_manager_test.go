package repository

import (
	"context"
	"testing"
	"time"
)

func TestNewConnectionManager(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *RepositoryConfig
		wantErr bool
	}{
		{
			name: "empty config",
			config: &RepositoryConfig{
				DatabaseURL: "",
				RedisAddr:   "",
			},
			wantErr: false, // Should succeed with no connections
		},
		{
			name: "invalid database URL",
			config: &RepositoryConfig{
				DatabaseURL: "invalid-url",
				RedisAddr:   "",
				ConnectionTimeout: 1 * time.Second,
			},
			wantErr: true, // Should fail with invalid database URL
		},
		{
			name: "invalid redis address",
			config: &RepositoryConfig{
				DatabaseURL: "",
				RedisAddr:   "invalid-redis-addr",
				ConnectionTimeout: 1 * time.Second,
			},
			wantErr: false, // Should succeed but without Redis (fallback mode)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := NewConnectionManager(ctx, tt.config)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if cm == nil {
				t.Error("Expected non-nil connection manager")
				return
			}

			// Test connection status methods
			if tt.config.DatabaseURL != "" {
				hasPostgres := cm.HasPostgres()
				_ = hasPostgres // Just verify the method exists
			}

			if tt.config.RedisAddr != "" {
				hasRedis := cm.HasRedis()
				_ = hasRedis // Just verify the method exists
			}

			// Test health check (should not panic)
			err = cm.HealthCheck(ctx)
			_ = err // We don't assert on the result since we don't have real connections in tests

			// Test close (should not panic)
			err = cm.Close()
			if err != nil {
				t.Logf("Close returned error (expected in test): %v", err)
			}
		})
	}
}

func TestConnectionManagerMethods(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}
	
	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	// Test that methods don't panic with nil connections
	if cm.HasPostgres() {
		t.Error("Expected HasPostgres to return false with empty database URL")
	}

	if cm.HasRedis() {
		t.Error("Expected HasRedis to return false with empty Redis address")
	}

	if pool := cm.PostgresPool(); pool != nil {
		t.Error("Expected PostgresPool to return nil with empty database URL")
	}

	if client := cm.RedisClient(); client != nil {
		t.Error("Expected RedisClient to return nil with empty Redis address")
	}
}

func TestConnectionManagerStats(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}
	
	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	// Test getting stats with no connections
	stats := cm.GetConnectionStats(context.Background())
	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	if stats.PostgreSQL != nil {
		t.Error("Expected nil PostgreSQL stats with no database connection")
	}

	if stats.Redis != nil {
		t.Error("Expected nil Redis stats with no Redis connection")
	}
}

func TestConnectionManagerReconnect(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}
	
	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	// Test reconnect with no connections configured (should succeed)
	err = cm.Reconnect(context.Background())
	if err != nil {
		t.Errorf("Unexpected error on reconnect with no connections: %v", err)
	}
}

func TestConnectionManagerWaitForConnections(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
	}
	
	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}

	// Test waiting for connections with no connections configured (should succeed immediately)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	err = cm.WaitForConnections(ctx)
	if err != nil {
		t.Errorf("Unexpected error waiting for connections: %v", err)
	}
}

func TestConnectionManagerConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config *RepositoryConfig
	}{
		{
			name: "with connection limits",
			config: &RepositoryConfig{
				DatabaseURL:        "",
				RedisAddr:          "",
				MaxConnections:     10,
				MaxIdleConnections: 2,
				ConnectionTimeout:  5 * time.Second,
			},
		},
		{
			name: "with cache TTL settings",
			config: &RepositoryConfig{
				DatabaseURL:        "",
				RedisAddr:          "",
				TenantCacheTTL:     10 * time.Minute,
				AssignmentCacheTTL: 5 * time.Minute,
				TokenCacheTTL:      20 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := NewConnectionManager(context.Background(), tt.config)
			if err != nil {
				t.Errorf("Failed to create connection manager: %v", err)
				return
			}

			if cm.config != tt.config {
				t.Error("Connection manager should store the provided config")
			}

			// Test that close doesn't panic
			_ = cm.Close()
		})
	}
}