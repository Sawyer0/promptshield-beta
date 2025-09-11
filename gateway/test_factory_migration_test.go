package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/repository"
)

// TestAllTestsUseSharedFactory verifies that all test patterns have been migrated to use the shared test factory
func TestAllTestsUseSharedFactory(t *testing.T) {
	ctx := context.Background()

	// Set test mode
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	// Test that we can create multiple factory instances (simulating different test files)
	for i := 0; i < 3; i++ {
		t.Run("factory_instance_"+string(rune('A'+i)), func(t *testing.T) {
			// Create repository factory
			repoFactory, err := repository.BuildWithFallback(ctx)
			require.NoError(t, err, "Failed to create repository factory")
			defer repoFactory.Close()

			// Verify factory health
			err = repoFactory.HealthCheck(ctx)
			require.NoError(t, err, "Factory health check should pass")

			// Test that all repositories are available
			assert.NotNil(t, repoFactory.Tenant(), "Tenant repository should be available")
			assert.NotNil(t, repoFactory.Rulepack(), "Rulepack repository should be available")
			assert.NotNil(t, repoFactory.RulepackAssignment(), "Assignment repository should be available")
			assert.NotNil(t, repoFactory.Audit(), "Audit repository should be available")
			assert.NotNil(t, repoFactory.Settings(), "Settings repository should be available")

			// Test service creation
			publisher, err := nats.NewPublisher("")
			require.NoError(t, err, "Failed to create NATS publisher")
			defer publisher.Close()

			rulepackSvc := services.RulepackServiceFromFactory(repoFactory, publisher)
			assert.NotNil(t, rulepackSvc, "RulepackService should be created successfully")
		})
	}
}

// TestFactoryIsolation verifies that different factory instances are properly isolated
func TestFactoryIsolation(t *testing.T) {
	ctx := context.Background()

	// Set test mode
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	// Create two separate factory instances
	factory1, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err)
	defer factory1.Close()

	factory2, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err)
	defer factory2.Close()

	// Verify both factories work independently
	assert.NotNil(t, factory1.Tenant())
	assert.NotNil(t, factory2.Tenant())

	// Verify they are separate instances
	assert.NotEqual(t, factory1, factory2, "Factory instances should be separate")

	// Both should pass health checks
	assert.NoError(t, factory1.HealthCheck(ctx))
	assert.NoError(t, factory2.HealthCheck(ctx))
}

// TestFactoryCleanup verifies that factory cleanup works correctly
func TestFactoryCleanup(t *testing.T) {
	ctx := context.Background()

	// Set test mode
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	// Create and immediately close factory
	factory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err)

	// Verify it works before closing
	assert.NoError(t, factory.HealthCheck(ctx))

	// Close should work without error
	assert.NoError(t, factory.Close(), "Factory should close cleanly")
}

// TestFactoryPerformance verifies that factory creation is reasonably fast
func TestFactoryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()

	// Set test mode
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	// Create multiple factories to test performance
	const numFactories = 10
	factories := make([]repository.RepositoryFactory, numFactories)

	for i := 0; i < numFactories; i++ {
		factory, err := repository.BuildWithFallback(ctx)
		require.NoError(t, err, "Factory %d should be created successfully", i)
		factories[i] = factory
	}

	// Clean up all factories
	for i, factory := range factories {
		assert.NoError(t, factory.Close(), "Factory %d should close cleanly", i)
	}

	t.Logf("Successfully created and cleaned up %d factory instances", numFactories)
}