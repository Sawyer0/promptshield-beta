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

// TestRepositoryFactoryIntegration tests that the gateway components work with the repository factory
func TestRepositoryFactoryIntegration(t *testing.T) {
	ctx := context.Background()

	// Save original environment
	originalTestMode := os.Getenv("PS_TEST_MODE")
	defer func() {
		if originalTestMode == "" {
			os.Unsetenv("PS_TEST_MODE")
		} else {
			os.Setenv("PS_TEST_MODE", originalTestMode)
		}
	}()

	// Set test mode to ensure we get a test factory
	os.Setenv("PS_TEST_MODE", "true")

	// Test repository factory creation (same as in main.go)
	repoFactory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err, "Failed to create repository factory")
	defer repoFactory.Close()

	// Verify factory health
	err = repoFactory.HealthCheck(ctx)
	require.NoError(t, err, "Factory health check should pass")

	// Test that we can get all required repositories (same as in main.go)
	tenantRepo := repoFactory.Tenant()
	require.NotNil(t, tenantRepo, "Tenant repository should not be nil")

	rulepackRepo := repoFactory.Rulepack()
	require.NotNil(t, rulepackRepo, "Rulepack repository should not be nil")

	assignmentRepo := repoFactory.RulepackAssignment()
	require.NotNil(t, assignmentRepo, "Assignment repository should not be nil")

	auditRepo := repoFactory.Audit()
	require.NotNil(t, auditRepo, "Audit repository should not be nil")

	settingsRepo := repoFactory.Settings()
	require.NotNil(t, settingsRepo, "Settings repository should not be nil")

	// Test service creation using factory (same as in main.go)
	publisher, err := nats.NewPublisher("")
	require.NoError(t, err, "Failed to create NATS publisher")
	defer publisher.Close()

	rulepackSvc := services.RulepackServiceFromFactory(repoFactory, publisher)
	require.NotNil(t, rulepackSvc, "RulepackService should not be nil")

	t.Log("Repository factory integration test completed successfully")
}

// TestFactoryGracefulShutdown tests that the repository factory shuts down gracefully
func TestFactoryGracefulShutdown(t *testing.T) {
	ctx := context.Background()

	// Set test mode
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	// Create factory
	factory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err)

	// Verify it's working
	err = factory.HealthCheck(ctx)
	require.NoError(t, err)

	// Test graceful shutdown
	err = factory.Close()
	assert.NoError(t, err, "Factory should close gracefully")
}

// TestFactoryEnvironmentDetection tests that the factory correctly detects different environments
func TestFactoryEnvironmentDetection(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		envVars map[string]string
		expectSuccess bool
	}{
		{
			name: "explicit test mode",
			envVars: map[string]string{
				"PS_TEST_MODE": "true",
			},
			expectSuccess: true,
		},
		{
			name: "CI environment",
			envVars: map[string]string{
				"CI": "true",
			},
			expectSuccess: true,
		},
		{
			name: "no configuration (should default to test)",
			envVars: map[string]string{},
			expectSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save and restore environment
			originalEnv := make(map[string]string)
			allKeys := []string{"PS_TEST_MODE", "CI", "PS_PG_DSN", "PS_REDIS_ADDR", "PS_ENVIRONMENT"}
			for _, key := range allKeys {
				originalEnv[key] = os.Getenv(key)
				os.Unsetenv(key)
			}
			defer func() {
				for key, val := range originalEnv {
					if val == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, val)
					}
				}
			}()

			// Set test environment
			for key, val := range tc.envVars {
				os.Setenv(key, val)
			}

			// Test factory creation
			factory, err := repository.BuildWithAutoDetection(ctx)
			
			if tc.expectSuccess {
				require.NoError(t, err, "Factory creation should succeed")
				require.NotNil(t, factory, "Factory should not be nil")
				
				// Test that all repositories are available
				assert.NotNil(t, factory.Tenant())
				assert.NotNil(t, factory.Rulepack())
				assert.NotNil(t, factory.RulepackAssignment())
				assert.NotNil(t, factory.Audit())
				assert.NotNil(t, factory.Settings())
				
				factory.Close()
			} else {
				assert.Error(t, err, "Factory creation should fail")
			}
		})
	}
}