// +build integration

package services_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/testutil/fixtures"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
)

// TestIntegration_RulepackCreationToActivation tests the complete rulepack lifecycle
func TestIntegration_RulepackCreationToActivation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	
	// Step 1: Validate a rulepack DSL
	validationSvc := &services.ValidationService{}
	err := validationSvc.ValidateDSL(ctx, []byte(fixtures.ValidRulepackJSON))
	require.NoError(t, err, "Rulepack should be valid")
	
	// Step 2: Normalize the DSL
	normalizedDSL, err := validationSvc.NormalizeDSL(ctx, []byte(fixtures.ValidRulepackJSON))
	require.NoError(t, err)
	assert.Contains(t, string(normalizedDSL), "apiVersion")
	
	// Step 3: Create version and activate with mock repository and publisher
	repo := new(mocks.MockRulepackRepository)
	pub := new(mocks.MockPublisher)
	
	expectedChecksum := fixtures.ComputeChecksum(string(normalizedDSL))
	
	// Setup expectations
	repo.On("CreateVersion", mock.Anything, fixtures.TenantID1, fixtures.RulepackID1, 1, string(normalizedDSL), expectedChecksum).
		Return(fixtures.VersionID1, nil).Once()
	repo.On("Activate", mock.Anything, fixtures.TenantID1, fixtures.RulepackID1, fixtures.VersionID1).
		Return(nil).Once()
	
	// Expect the rule update to be published
	var capturedUpdate interface{}
	pub.On("PublishRuleUpdate", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedUpdate = args.Get(1)
		}).
		Return(nil).Once()
	
	rulepackSvc := &services.RulepackService{
		repo:      repo,
		publisher: pub,
	}
	
	// Execute the full flow
	err = rulepackSvc.CreateVersionActivate(ctx, fixtures.TenantID1, fixtures.RulepackID1, 1, string(normalizedDSL))
	require.NoError(t, err)
	
	// Verify all expectations were met
	repo.AssertExpectations(t)
	pub.AssertExpectations(t)
	
	// Verify the published update
	assert.NotNil(t, capturedUpdate)
}

// TestIntegration_ValidationToRulepackFlow tests validation service integration with rulepack service
func TestIntegration_ValidationToRulepackFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	
	// Test with YAML input that needs normalization
	yamlInput := []byte(fixtures.ValidRulepackYAML)
	
	validationSvc := &services.ValidationService{}
	
	// Validate YAML
	err := validationSvc.ValidateDSL(ctx, yamlInput)
	require.NoError(t, err)
	
	// Normalize YAML to JSON
	jsonOutput, err := validationSvc.NormalizeDSL(ctx, yamlInput)
	require.NoError(t, err)
	
	// Verify normalized output is valid JSON
	var parsed interface{}
	err = json.Unmarshal(jsonOutput, &parsed)
	require.NoError(t, err, "Normalized output should be valid JSON")
	
	// Now use the normalized JSON in rulepack service
	repo := new(mocks.MockRulepackRepository)
	expectedChecksum := fixtures.ComputeChecksum(string(jsonOutput))
	
	repo.On("CreateVersion", mock.Anything, fixtures.TenantID1, fixtures.RulepackID1, 1, string(jsonOutput), expectedChecksum).
		Return(fixtures.VersionID1, nil)
	repo.On("Activate", mock.Anything, fixtures.TenantID1, fixtures.RulepackID1, fixtures.VersionID1).
		Return(nil)
	
	rulepackSvc := &services.RulepackService{
		repo:      repo,
		publisher: nil, // Test without publisher
	}
	
	err = rulepackSvc.CreateVersionActivate(ctx, fixtures.TenantID1, fixtures.RulepackID1, 1, string(jsonOutput))
	require.NoError(t, err)
	
	repo.AssertExpectations(t)
}