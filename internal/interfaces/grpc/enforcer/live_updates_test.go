package grpcenforcer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/contracts"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEnforcer_ReloadRules_Unit(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	packID := uuid.New()

	// Mock repository with initial active rulepack
	repo := &mockRepo{}
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{
		{
			ID:          packID,
			Name:        "test-rules",
			Description: "Test rules",
			Version:     1,
			Active:      true,
		},
	}, nil)

	// Mock DSL content
	dslContent := `
metadata:
  name: test-rules
  description: Test rules
rules:
  - id: test-rule-v1
    level: 1
    keywords: ["malicious"]
    severity: HIGH
    category: security
`

	repo.On("GetActive", mock.Anything, packID).Return(json.RawMessage(dslContent), 1, nil)

	// Create enforcer
	opts := Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
	}

	server := NewWithOptions(opts)

	// Test the ReloadRules method directly
	ctx := context.Background()
	err := server.ReloadRules(ctx)
	assert.NoError(t, err)

	// Verify repo was called correctly - ListByTenant should be called twice (once in constructor, once in ReloadRules)
	repo.AssertExpectations(t)
}

func TestEnforcer_RuleUpdateHandler_Unit(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	packID := uuid.New()

	// Mock repository - simplified expectations
	repo := &mockRepo{}

	// Initial load during constructor
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{}, nil).Once()

	// Handler calls ReloadRules which calls ListByTenant again
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{
		{
			ID:          packID,
			Name:        "updated-rules",
			Description: "Updated rules",
			Version:     2,
			Active:      true,
		},
	}, nil).Once()

	// Updated DSL content for the second call
	updatedDSL := `
metadata:
  name: updated-rules
  description: Updated rules
rules:
  - id: test-rule-v2
    level: 1
    keywords: ["harmful"]
    severity: MEDIUM
    category: security
`

	repo.On("GetActive", mock.Anything, packID).Return(json.RawMessage(updatedDSL), 2, nil).Once()

	// Create enforcer
	opts := Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
	}

	server := NewWithOptions(opts)

	// Test the ReloadRules method directly (which is what the handler would call)
	ctx := context.Background()
	err := server.ReloadRules(ctx)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestEnforcer_RuleUpdateSubscriber_Creation(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Create a simple handler
	handler := func(ctx context.Context, update nats.RuleUpdate) error {
		return nil
	}

	// Create subscriber for specific tenant (without Redis connection)
	sub, err := nats.NewSubscriber("", "test-group", "test-consumer", tenantID.String(), handler)
	assert.NoError(t, err)
	assert.NotNil(t, sub)

	// Test that subscriber can be created and closed without errors
	sub.Close()
}

func TestEnforcer_LiveUpdates_NoRedis(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Mock repository
	repo := &mockRepo{}
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{}, nil)

	// Create enforcer without Redis
	opts := Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
		// No RedisAddr - live updates disabled
	}

	server := NewWithOptions(opts)

	// Should still work, just without live updates
	assert.NotNil(t, server)
	assert.Nil(t, server.subscriber) // No subscriber when Redis not configured

	// Shutdown should not panic
	server.Shutdown()

	repo.AssertExpectations(t)
}

func TestEnforcer_GracefulShutdown(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Mock repository
	repo := &mockRepo{}
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{}, nil)

	// Create enforcer (without actual Redis for testing)
	opts := Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
		// RedisAddr: "", // No Redis for test
	}

	server := NewWithOptions(opts)

	// Test graceful shutdown
	server.Shutdown() // Should not panic

	repo.AssertExpectations(t)
}
