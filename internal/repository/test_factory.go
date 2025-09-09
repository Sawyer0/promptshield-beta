package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/promptshield/promptshield/internal/contracts"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
	sharedcontracts "github.com/promptshield/promptshield/internal/shared/contracts"
)

// TestRepositoryFactory provides in-memory and mock repositories for testing
type TestRepositoryFactory struct {
	mu                 sync.Mutex
	config             *RepositoryConfig
	customRepositories map[string]interface{}

	// Repository instances
	tenantRepo     domain.TenantRepository
	auditRepo      domain.AuditRepository
	assignmentRepo domain.RulepackAssignmentRepository
	apiTokenRepo   domain.APITokenRepository
	settingsRepo   domain.SettingsRepository
	rulepackRepo   contracts.RulepackRepository
	policyRepo     sharedcontracts.PolicyRepository

	// Test utilities
	cleanupFuncs []func() error
}

// NewTestRepositoryFactory creates a new test repository factory
func NewTestRepositoryFactory(config *RepositoryConfig, customRepos map[string]interface{}) (*TestRepositoryFactory, error) {
	factory := &TestRepositoryFactory{
		config:             config,
		customRepositories: customRepos,
		cleanupFuncs:       make([]func() error, 0),
	}

	// Initialize in-memory repositories
	if err := factory.initializeRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize test repositories: %w", err)
	}

	return factory, nil
}

// initializeRepositories sets up all repository instances with in-memory implementations
func (f *TestRepositoryFactory) initializeRepositories() error {
	// Initialize rulepack repository (already has in-memory implementation)
	f.rulepackRepo = memory.NewRulepackRepository()

	// Initialize policy repository (in-memory implementation)
	f.policyRepo = memory.NewPolicyRepository()

	// Initialize other repositories with mock implementations
	f.tenantRepo = NewMockTenantRepository()
	f.auditRepo = NewMockAuditRepository()
	f.assignmentRepo = NewMockRulepackAssignmentRepository()
	f.apiTokenRepo = NewMockAPITokenRepository()
	f.settingsRepo = NewMockSettingsRepository()

	return nil
}

// Tenant returns the tenant repository
func (f *TestRepositoryFactory) Tenant() domain.TenantRepository {
	if custom, exists := f.customRepositories["tenant"]; exists {
		if repo, ok := custom.(domain.TenantRepository); ok {
			return repo
		}
	}
	return f.tenantRepo
}

// Audit returns the audit repository
func (f *TestRepositoryFactory) Audit() domain.AuditRepository {
	if custom, exists := f.customRepositories["audit"]; exists {
		if repo, ok := custom.(domain.AuditRepository); ok {
			return repo
		}
	}
	return f.auditRepo
}

// RulepackAssignment returns the rulepack assignment repository
func (f *TestRepositoryFactory) RulepackAssignment() domain.RulepackAssignmentRepository {
	if custom, exists := f.customRepositories["assignment"]; exists {
		if repo, ok := custom.(domain.RulepackAssignmentRepository); ok {
			return repo
		}
	}
	return f.assignmentRepo
}

// APIToken returns the API token repository
func (f *TestRepositoryFactory) APIToken() domain.APITokenRepository {
	if custom, exists := f.customRepositories["apitoken"]; exists {
		if repo, ok := custom.(domain.APITokenRepository); ok {
			return repo
		}
	}
	return f.apiTokenRepo
}

// Settings returns the settings repository
func (f *TestRepositoryFactory) Settings() domain.SettingsRepository {
	if custom, exists := f.customRepositories["settings"]; exists {
		if repo, ok := custom.(domain.SettingsRepository); ok {
			return repo
		}
	}
	return f.settingsRepo
}

// Rulepack returns the rulepack repository
func (f *TestRepositoryFactory) Rulepack() contracts.RulepackRepository {
	if custom, exists := f.customRepositories["rulepack"]; exists {
		if repo, ok := custom.(contracts.RulepackRepository); ok {
			return repo
		}
	}
	return f.rulepackRepo
}

// Policy returns the policy repository
func (f *TestRepositoryFactory) Policy() sharedcontracts.PolicyRepository {
	if custom, exists := f.customRepositories["policy"]; exists {
		if repo, ok := custom.(sharedcontracts.PolicyRepository); ok {
			return repo
		}
	}
	return f.policyRepo
}

// Close closes all repository connections and runs cleanup functions
func (f *TestRepositoryFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	var errors []error

	// Run all cleanup functions in reverse order (LIFO)
	for i := len(f.cleanupFuncs) - 1; i >= 0; i-- {
		if err := f.cleanupFuncs[i](); err != nil {
			errors = append(errors, fmt.Errorf("cleanup function %d: %w", i, err))
		}
	}
	
	// Clear repository references to help with garbage collection
	f.tenantRepo = nil
	f.auditRepo = nil
	f.assignmentRepo = nil
	f.apiTokenRepo = nil
	f.settingsRepo = nil
	f.rulepackRepo = nil
	
	// Clear cleanup functions
	f.cleanupFuncs = nil

	if len(errors) > 0 {
		return fmt.Errorf("test factory cleanup errors: %v", errors)
	}

	return nil
}

// HealthCheck verifies repository health (always returns nil for test factory)
func (f *TestRepositoryFactory) HealthCheck(ctx context.Context) error {
	return nil
}

// AddCleanupFunc adds a cleanup function to be called when the factory is closed
func (f *TestRepositoryFactory) AddCleanupFunc(cleanup func() error) {
	f.cleanupFuncs = append(f.cleanupFuncs, cleanup)
}

// ResetAllRepositories clears all data from in-memory repositories
func (f *TestRepositoryFactory) ResetAllRepositories() error {
	// Reset each repository if it supports reset
	if resettable, ok := f.tenantRepo.(interface{ Reset() error }); ok {
		if err := resettable.Reset(); err != nil {
			return fmt.Errorf("failed to reset tenant repository: %w", err)
		}
	}

	if resettable, ok := f.auditRepo.(interface{ Reset() error }); ok {
		if err := resettable.Reset(); err != nil {
			return fmt.Errorf("failed to reset audit repository: %w", err)
		}
	}

	if resettable, ok := f.assignmentRepo.(interface{ Reset() error }); ok {
		if err := resettable.Reset(); err != nil {
			return fmt.Errorf("failed to reset assignment repository: %w", err)
		}
	}

	if resettable, ok := f.apiTokenRepo.(interface{ Reset() error }); ok {
		if err := resettable.Reset(); err != nil {
			return fmt.Errorf("failed to reset API token repository: %w", err)
		}
	}

	if resettable, ok := f.settingsRepo.(interface{ Reset() error }); ok {
		if err := resettable.Reset(); err != nil {
			return fmt.Errorf("failed to reset settings repository: %w", err)
		}
	}

	return nil
}

// GetRepositoryCount returns the number of initialized repositories
func (f *TestRepositoryFactory) GetRepositoryCount() int {
	count := 0
	if f.tenantRepo != nil {
		count++
	}
	if f.auditRepo != nil {
		count++
	}
	if f.assignmentRepo != nil {
		count++
	}
	if f.apiTokenRepo != nil {
		count++
	}
	if f.settingsRepo != nil {
		count++
	}
	if f.rulepackRepo != nil {
		count++
	}
	return count
}

// AddCleanup adds a cleanup function to be called when the factory is closed
func (f *TestRepositoryFactory) AddCleanup(cleanup func() error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleanupFuncs = append(f.cleanupFuncs, cleanup)
}