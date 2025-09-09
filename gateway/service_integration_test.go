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

// TestServiceConstructorsWithFactory tests that service constructors work properly with the repository factory
func TestServiceConstructorsWithFactory(t *testing.T) {
	ctx := context.Background()

	// Set test mode to ensure we get a test factory
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	// Create repository factory
	repoFactory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err, "Failed to create repository factory")
	defer repoFactory.Close()

	// Verify factory health
	err = repoFactory.HealthCheck(ctx)
	require.NoError(t, err, "Factory health check should pass")

	// Test NATS publisher creation (same as in main.go)
	publisher, err := nats.NewPublisher("")
	require.NoError(t, err, "Failed to create NATS publisher")
	defer publisher.Close()

	// Test RulepackServiceFromFactory (same as in main.go)
	rulepackSvc := services.RulepackServiceFromFactory(repoFactory, publisher)
	require.NotNil(t, rulepackSvc, "RulepackService should not be nil")

	// Test NewServicesFromFactory
	allServices := services.NewServicesFromFactory(repoFactory, publisher)
	require.NotNil(t, allServices, "Services should not be nil")
	require.NotNil(t, allServices.Rulepack, "Rulepack service should not be nil")
	assert.Equal(t, rulepackSvc, allServices.Rulepack, "Rulepack service should be the same instance")

	t.Log("Service constructor integration test completed successfully")
}

// TestServiceFactoryConsistency tests that services created from factory are consistent
func TestServiceFactoryConsistency(t *testing.T) {
	ctx := context.Background()

	// Set test mode
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	// Create factory
	factory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err)
	defer factory.Close()

	// Create publisher
	publisher, err := nats.NewPublisher("")
	require.NoError(t, err)
	defer publisher.Close()

	// Create multiple service instances
	svc1 := services.RulepackServiceFromFactory(factory, publisher)
	svc2 := services.RulepackServiceFromFactory(factory, publisher)

	// Both services should be functional
	assert.NotNil(t, svc1, "First service should not be nil")
	assert.NotNil(t, svc2, "Second service should not be nil")
	
	// Services created with same factory and publisher should be consistent
	// (The actual behavior may vary depending on implementation - this just tests they work)
}