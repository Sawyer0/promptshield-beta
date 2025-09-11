package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

func TestNewTestRepositoryFactory(t *testing.T) {
	tests := []struct {
		name        string
		config      *RepositoryConfig
		customRepos map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "valid config with no custom repositories",
			config:      DefaultConfig(),
			customRepos: nil,
			wantErr:     false,
		},
		{
			name:        "valid config with custom repositories",
			config:      DefaultConfig(),
			customRepos: map[string]interface{}{"tenant": NewMockTenantRepository()},
			wantErr:     false,
		},
		{
			name:        "test mode enabled",
			config:      &RepositoryConfig{TestMode: true, Environment: "test"},
			customRepos: nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewTestRepositoryFactory(tt.config, tt.customRepos)

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

			if factory == nil {
				t.Error("Expected non-nil factory")
				return
			}

			// Verify all repositories are initialized
			if factory.Tenant() == nil {
				t.Error("Tenant repository not initialized")
			}
			if factory.Audit() == nil {
				t.Error("Audit repository not initialized")
			}
			if factory.RulepackAssignment() == nil {
				t.Error("Assignment repository not initialized")
			}
			if factory.APIToken() == nil {
				t.Error("API token repository not initialized")
			}
			if factory.Settings() == nil {
				t.Error("Settings repository not initialized")
			}
			if factory.Rulepack() == nil {
				t.Error("Rulepack repository not initialized")
			}

			// Test repository count
			expectedCount := 6 // All repositories should be initialized
			if count := factory.GetRepositoryCount(); count != expectedCount {
				t.Errorf("Expected repository count %d, got %d", expectedCount, count)
			}

			// Test health check
			if err := factory.HealthCheck(context.Background()); err != nil {
				t.Errorf("Health check failed: %v", err)
			}

			// Test close
			if err := factory.Close(); err != nil {
				t.Errorf("Close failed: %v", err)
			}
		})
	}
}

func TestTestRepositoryFactoryCustomRepositories(t *testing.T) {
	customTenant := NewMockTenantRepository()
	customAudit := NewMockAuditRepository()

	customRepos := map[string]interface{}{
		"tenant": customTenant,
		"audit":  customAudit,
	}

	factory, err := NewTestRepositoryFactory(DefaultConfig(), customRepos)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Close()

	// Test that custom repositories are used
	if factory.Tenant() != customTenant {
		t.Error("Custom tenant repository not used")
	}
	if factory.Audit() != customAudit {
		t.Error("Custom audit repository not used")
	}

	// Test that non-custom repositories use defaults
	if factory.RulepackAssignment() == nil {
		t.Error("Default assignment repository not initialized")
	}
}

func TestTestRepositoryFactoryRepositoryOperations(t *testing.T) {
	factory, err := NewTestRepositoryFactory(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Close()

	ctx := context.Background()

	// Test tenant repository operations
	t.Run("tenant operations", func(t *testing.T) {
		tenantRepo := factory.Tenant()
		
		// Create a test tenant
		tenant := &domain.Tenant{
			ID:   uuid.New(),
			Name: "Test Tenant",
		}

		// Test create
		if err := tenantRepo.Create(ctx, tenant); err != nil {
			t.Errorf("Failed to create tenant: %v", err)
		}

		// Test get
		retrieved, err := tenantRepo.Get(ctx, tenant.ID)
		if err != nil {
			t.Errorf("Failed to get tenant: %v", err)
		}
		if retrieved.ID != tenant.ID || retrieved.Name != tenant.Name {
			t.Errorf("Retrieved tenant doesn't match: got %+v, want %+v", retrieved, tenant)
		}

		// Test list
		tenants, total, err := tenantRepo.List(ctx, 0, 10)
		if err != nil {
			t.Errorf("Failed to list tenants: %v", err)
		}
		if total != 1 {
			t.Errorf("Expected 1 tenant, got %d", total)
		}
		if len(tenants) != 1 {
			t.Errorf("Expected 1 tenant in list, got %d", len(tenants))
		}

		// Test delete
		if err := tenantRepo.Delete(ctx, tenant.ID); err != nil {
			t.Errorf("Failed to delete tenant: %v", err)
		}

		// Verify deletion
		_, err = tenantRepo.Get(ctx, tenant.ID)
		if err == nil {
			t.Error("Expected error when getting deleted tenant")
		}
	})

	// Test API token repository operations
	t.Run("api token operations", func(t *testing.T) {
		tokenRepo := factory.APIToken()
		
		// Create a test token
		token := &domain.APIToken{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			TokenHash: "test-hash",
			Name:      "Test Token",
			Scopes:    []string{"read", "write"},
			CreatedAt: time.Now(),
		}

		// Test create
		if err := tokenRepo.Create(ctx, token); err != nil {
			t.Errorf("Failed to create token: %v", err)
		}

		// Test get
		retrieved, err := tokenRepo.Get(ctx, token.ID)
		if err != nil {
			t.Errorf("Failed to get token: %v", err)
		}
		if retrieved.ID != token.ID {
			t.Errorf("Retrieved token ID doesn't match: got %v, want %v", retrieved.ID, token.ID)
		}

		// Test get by hash
		retrieved, err = tokenRepo.GetByHash(ctx, token.TokenHash)
		if err != nil {
			t.Errorf("Failed to get token by hash: %v", err)
		}
		if retrieved.ID != token.ID {
			t.Errorf("Retrieved token by hash doesn't match: got %v, want %v", retrieved.ID, token.ID)
		}

		// Test list by tenant
		tokens, err := tokenRepo.ListByTenant(ctx, token.TenantID)
		if err != nil {
			t.Errorf("Failed to list tokens by tenant: %v", err)
		}
		if len(tokens) != 1 {
			t.Errorf("Expected 1 token, got %d", len(tokens))
		}
	})
}

func TestTestRepositoryFactoryResetFunctionality(t *testing.T) {
	factory, err := NewTestRepositoryFactory(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Close()

	ctx := context.Background()

	// Add some test data
	tenant := &domain.Tenant{
		ID:   uuid.New(),
		Name: "Test Tenant",
	}
	if err := factory.Tenant().Create(ctx, tenant); err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	// Verify data exists
	_, err = factory.Tenant().Get(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Failed to get tenant before reset: %v", err)
	}

	// Reset all repositories
	if err := factory.ResetAllRepositories(); err != nil {
		t.Errorf("Failed to reset repositories: %v", err)
	}

	// Verify data is cleared
	_, err = factory.Tenant().Get(ctx, tenant.ID)
	if err == nil {
		t.Error("Expected error when getting tenant after reset")
	}
}

func TestTestRepositoryFactoryCleanupFunctions(t *testing.T) {
	factory, err := NewTestRepositoryFactory(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	cleanupCalled := false
	factory.AddCleanupFunc(func() error {
		cleanupCalled = true
		return nil
	})

	// Test that cleanup function is called on close
	if err := factory.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if !cleanupCalled {
		t.Error("Cleanup function was not called")
	}
}

func TestTestRepositoryFactoryCleanupErrors(t *testing.T) {
	factory, err := NewTestRepositoryFactory(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Add a cleanup function that returns an error
	factory.AddCleanupFunc(func() error {
		return fmt.Errorf("cleanup error")
	})

	// Test that close returns the cleanup error
	err = factory.Close()
	if err == nil {
		t.Error("Expected error from cleanup function")
	}
	if !containsString(err.Error(), "cleanup error") {
		t.Errorf("Expected cleanup error in message, got: %v", err)
	}
}

// Use the containsString function from production_factory_test.go