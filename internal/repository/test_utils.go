package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// TestRepositoryFactoryBuilder provides a fluent interface for creating test factories
type TestRepositoryFactoryBuilder struct {
	config      *RepositoryConfig
	customRepos map[string]interface{}
	cleanups    []func() error
}

// NewTestRepositoryFactoryBuilder creates a new test factory builder
func NewTestRepositoryFactoryBuilder() *TestRepositoryFactoryBuilder {
	return &TestRepositoryFactoryBuilder{
		config:      DefaultConfig(),
		customRepos: make(map[string]interface{}),
		cleanups:    make([]func() error, 0),
	}
}

// WithConfig sets a custom configuration
func (b *TestRepositoryFactoryBuilder) WithConfig(config *RepositoryConfig) *TestRepositoryFactoryBuilder {
	b.config = config
	return b
}

// WithTestMode enables test mode
func (b *TestRepositoryFactoryBuilder) WithTestMode() *TestRepositoryFactoryBuilder {
	b.config.TestMode = true
	b.config.Environment = "test"
	return b
}

// WithCustomTenant sets a custom tenant repository
func (b *TestRepositoryFactoryBuilder) WithCustomTenant(repo domain.TenantRepository) *TestRepositoryFactoryBuilder {
	b.customRepos["tenant"] = repo
	return b
}

// WithCustomAudit sets a custom audit repository
func (b *TestRepositoryFactoryBuilder) WithCustomAudit(repo domain.AuditRepository) *TestRepositoryFactoryBuilder {
	b.customRepos["audit"] = repo
	return b
}

// WithCustomAssignment sets a custom assignment repository
func (b *TestRepositoryFactoryBuilder) WithCustomAssignment(repo domain.RulepackAssignmentRepository) *TestRepositoryFactoryBuilder {
	b.customRepos["assignment"] = repo
	return b
}

// WithCustomAPIToken sets a custom API token repository
func (b *TestRepositoryFactoryBuilder) WithCustomAPIToken(repo domain.APITokenRepository) *TestRepositoryFactoryBuilder {
	b.customRepos["apitoken"] = repo
	return b
}

// WithCustomSettings sets a custom settings repository
func (b *TestRepositoryFactoryBuilder) WithCustomSettings(repo domain.SettingsRepository) *TestRepositoryFactoryBuilder {
	b.customRepos["settings"] = repo
	return b
}

// WithCleanup adds a cleanup function to be called when the factory is closed
func (b *TestRepositoryFactoryBuilder) WithCleanup(cleanup func() error) *TestRepositoryFactoryBuilder {
	b.cleanups = append(b.cleanups, cleanup)
	return b
}

// Build creates the test repository factory
func (b *TestRepositoryFactoryBuilder) Build(t *testing.T) *TestRepositoryFactory {
	factory, err := NewTestRepositoryFactory(b.config, b.customRepos)
	if err != nil {
		t.Fatalf("Failed to create test repository factory: %v", err)
	}

	// Add cleanup functions
	for _, cleanup := range b.cleanups {
		factory.AddCleanupFunc(cleanup)
	}

	// Add automatic cleanup on test completion
	t.Cleanup(func() {
		if err := factory.Close(); err != nil {
			t.Errorf("Failed to close test factory: %v", err)
		}
	})

	return factory
}

// CreateTestTenant creates a test tenant with default values
func CreateTestTenant() *domain.Tenant {
	return &domain.Tenant{
		ID:   uuid.New(),
		Name: "Test Tenant " + uuid.New().String()[:8],
	}
}

// CreateTestAPIToken creates a test API token with default values
func CreateTestAPIToken(tenantID uuid.UUID) *domain.APIToken {
	return &domain.APIToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		TokenHash: "test-hash-" + uuid.New().String()[:8],
		Name:      "Test Token",
		Scopes:    []string{"read", "write"},
		CreatedAt: time.Now(),
	}
}

// CreateTestRulepackAssignment creates a test rulepack assignment with default values
func CreateTestRulepackAssignment(tenantID uuid.UUID) *domain.RulepackAssignment {
	return &domain.RulepackAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RulepackID:  uuid.New(),
		TargetScope: "test-scope",
		Priority:    1,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestAuditEntry creates a test audit entry with default values
func CreateTestAuditEntry(tenantID *uuid.UUID) *domain.AuditEntry {
	actorID := uuid.New()
	return &domain.AuditEntry{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ActorID:    &actorID,
		ActorType:  domain.ActorTypeUser,
		ActorEmail: "test@example.com",
		Action:     "test-action",
		ObjectType: "test-object",
		ObjectID:   uuid.New(),
		CreatedAt:  time.Now(),
		Hash:       "test-hash",
		PrevHash:   "prev-hash",
	}
}

// CreateTestPlatformSettings creates test platform settings with default values
func CreateTestPlatformSettings() *domain.PlatformSettings {
	return &domain.PlatformSettings{
		ID:       uuid.New(),
		Settings: []byte(`{"test": "settings"}`),
	}
}

// AssertRepositoryCount asserts that the factory has the expected number of repositories
func AssertRepositoryCount(t *testing.T, factory *TestRepositoryFactory, expected int) {
	t.Helper()
	if count := factory.GetRepositoryCount(); count != expected {
		t.Errorf("Expected repository count %d, got %d", expected, count)
	}
}

// AssertRepositoryNotNil asserts that all repositories in the factory are not nil
func AssertRepositoryNotNil(t *testing.T, factory *TestRepositoryFactory) {
	t.Helper()
	if factory.Tenant() == nil {
		t.Error("Tenant repository is nil")
	}
	if factory.Audit() == nil {
		t.Error("Audit repository is nil")
	}
	if factory.RulepackAssignment() == nil {
		t.Error("Assignment repository is nil")
	}
	if factory.APIToken() == nil {
		t.Error("API token repository is nil")
	}
	if factory.Settings() == nil {
		t.Error("Settings repository is nil")
	}
	if factory.Rulepack() == nil {
		t.Error("Rulepack repository is nil")
	}
}

// AssertHealthCheckPasses asserts that the factory health check passes
func AssertHealthCheckPasses(t *testing.T, factory *TestRepositoryFactory) {
	t.Helper()
	if err := factory.HealthCheck(context.Background()); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

// ResetAndAssertEmpty resets all repositories and asserts they are empty
func ResetAndAssertEmpty(t *testing.T, factory *TestRepositoryFactory) {
	t.Helper()
	ctx := context.Background()

	// Reset all repositories
	if err := factory.ResetAllRepositories(); err != nil {
		t.Fatalf("Failed to reset repositories: %v", err)
	}

	// Create test data to verify reset worked
	tenant := CreateTestTenant()
	
	// Try to get the tenant (should fail since repositories are reset)
	_, err := factory.Tenant().Get(ctx, tenant.ID)
	if err == nil {
		t.Error("Expected error when getting tenant from reset repository")
	}
}

// WithTestData populates the factory with test data for integration testing
func WithTestData(t *testing.T, factory *TestRepositoryFactory) (*domain.Tenant, *domain.APIToken, *domain.RulepackAssignment) {
	t.Helper()
	ctx := context.Background()

	// Create test tenant
	tenant := CreateTestTenant()
	if err := factory.Tenant().Create(ctx, tenant); err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	// Create test API token
	token := CreateTestAPIToken(tenant.ID)
	if err := factory.APIToken().Create(ctx, token); err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}

	// Create test assignment
	assignment := CreateTestRulepackAssignment(tenant.ID)
	if err := factory.RulepackAssignment().Create(ctx, assignment); err != nil {
		t.Fatalf("Failed to create test assignment: %v", err)
	}

	return tenant, token, assignment
}