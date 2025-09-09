package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestTestRepositoryFactoryBuilder(t *testing.T) {
	t.Run("basic builder usage", func(t *testing.T) {
		factory := NewTestRepositoryFactoryBuilder().
			WithTestMode().
			Build(t)

		AssertRepositoryNotNil(t, factory)
		AssertRepositoryCount(t, factory, 6)
		AssertHealthCheckPasses(t, factory)
	})

	t.Run("builder with custom repositories", func(t *testing.T) {
		customTenant := NewMockTenantRepository()
		
		factory := NewTestRepositoryFactoryBuilder().
			WithCustomTenant(customTenant).
			WithTestMode().
			Build(t)

		if factory.Tenant() != customTenant {
			t.Error("Custom tenant repository not used")
		}
	})

	t.Run("builder with cleanup", func(t *testing.T) {
		cleanupCalled := false
		
		factory := NewTestRepositoryFactoryBuilder().
			WithCleanup(func() error {
				cleanupCalled = true
				return nil
			}).
			Build(t)

		// Close manually to test cleanup
		if err := factory.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}

		if !cleanupCalled {
			t.Error("Cleanup function was not called")
		}
	})
}

func TestTestDataCreators(t *testing.T) {
	t.Run("create test tenant", func(t *testing.T) {
		tenant := CreateTestTenant()
		if tenant == nil {
			t.Error("CreateTestTenant returned nil")
		}
		if tenant.ID == uuid.Nil {
			t.Error("Tenant ID is nil")
		}
		if tenant.Name == "" {
			t.Error("Tenant name is empty")
		}
	})

	t.Run("create test API token", func(t *testing.T) {
		tenantID := uuid.New()
		token := CreateTestAPIToken(tenantID)
		if token == nil {
			t.Error("CreateTestAPIToken returned nil")
		}
		if token.ID == uuid.Nil {
			t.Error("Token ID is nil")
		}
		if token.TenantID != tenantID {
			t.Error("Token tenant ID doesn't match")
		}
		if token.TokenHash == "" {
			t.Error("Token hash is empty")
		}
	})

	t.Run("create test assignment", func(t *testing.T) {
		tenantID := uuid.New()
		assignment := CreateTestRulepackAssignment(tenantID)
		if assignment == nil {
			t.Error("CreateTestRulepackAssignment returned nil")
		}
		if assignment.ID == uuid.Nil {
			t.Error("Assignment ID is nil")
		}
		if assignment.TenantID != tenantID {
			t.Error("Assignment tenant ID doesn't match")
		}
	})

	t.Run("create test audit entry", func(t *testing.T) {
		tenantID := uuid.New()
		entry := CreateTestAuditEntry(&tenantID)
		if entry == nil {
			t.Error("CreateTestAuditEntry returned nil")
		}
		if entry.ID == uuid.Nil {
			t.Error("Entry ID is nil")
		}
		if entry.TenantID == nil || *entry.TenantID != tenantID {
			t.Error("Entry tenant ID doesn't match")
		}
	})

	t.Run("create test platform settings", func(t *testing.T) {
		settings := CreateTestPlatformSettings()
		if settings == nil {
			t.Error("CreateTestPlatformSettings returned nil")
		}
		if settings.ID == uuid.Nil {
			t.Error("Settings ID is nil")
		}
		if len(settings.Settings) == 0 {
			t.Error("Settings data is empty")
		}
	})
}

func TestTestUtilityFunctions(t *testing.T) {
	factory := NewTestRepositoryFactoryBuilder().Build(t)

	t.Run("reset and assert empty", func(t *testing.T) {
		// Add some test data first
		tenant := CreateTestTenant()
		ctx := context.Background()
		if err := factory.Tenant().Create(ctx, tenant); err != nil {
			t.Fatalf("Failed to create test tenant: %v", err)
		}

		// Reset and verify empty
		ResetAndAssertEmpty(t, factory)
	})

	t.Run("with test data", func(t *testing.T) {
		tenant, token, assignment := WithTestData(t, factory)

		if tenant == nil || token == nil || assignment == nil {
			t.Error("WithTestData returned nil values")
		}

		// Verify data was created
		ctx := context.Background()
		
		retrievedTenant, err := factory.Tenant().Get(ctx, tenant.ID)
		if err != nil {
			t.Errorf("Failed to retrieve created tenant: %v", err)
		}
		if retrievedTenant.ID != tenant.ID {
			t.Error("Retrieved tenant ID doesn't match")
		}

		retrievedToken, err := factory.APIToken().Get(ctx, token.ID)
		if err != nil {
			t.Errorf("Failed to retrieve created token: %v", err)
		}
		if retrievedToken.ID != token.ID {
			t.Error("Retrieved token ID doesn't match")
		}

		retrievedAssignment, err := factory.RulepackAssignment().Get(ctx, assignment.ID)
		if err != nil {
			t.Errorf("Failed to retrieve created assignment: %v", err)
		}
		if retrievedAssignment.ID != assignment.ID {
			t.Error("Retrieved assignment ID doesn't match")
		}
	})
}