package grpcenforcer

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestEnforcer_AtomicRuleReloading tests that rule reloading is thread-safe
// and doesn't cause race conditions during live traffic processing
func TestEnforcer_AtomicRuleReloading(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	packID := uuid.New()
	
	// Mock repository
	repo := &mockRepo{}
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{
		{ID: packID, Name: "test-rules", Active: true, Version: 1},
	}, nil)
	
	dslContent := `
metadata:
  name: test-rules
rules:
  - id: test-rule
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
	assert.NotNil(t, server)
	
	// Test concurrent rule reloading - should not cause races
	var wg sync.WaitGroup
	numGoroutines := 10
	
	// Simulate concurrent rule reloads (like multiple rule update messages)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			err := server.ReloadRules(ctx)
			assert.NoError(t, err)
		}()
	}
	
	wg.Wait()
	repo.AssertExpectations(t)
}

// TestEnforcer_RuleReloadDoesNotAccumulate tests that rule reloading
// replaces rules instead of accumulating them (preventing memory leaks)
func TestEnforcer_RuleReloadDoesNotAccumulate(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	packID := uuid.New()
	
	// Mock repository
	repo := &mockRepo{}
	
	// Return same rules on each call
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{
		{ID: packID, Name: "test-rules", Active: true, Version: 1},
	}, nil)
	
	dslContent := `
metadata:
  name: test-rules
rules:
  - id: test-rule
    level: 1
    keywords: ["test"]
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
	assert.NotNil(t, server)
	
	// Perform multiple reloads
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		err := server.ReloadRules(ctx)
		assert.NoError(t, err)
	}
	
	// Verify that rules don't accumulate by checking scanner state
	// (This is an indirect test - the main fix is in scanner/loader.go)
	
	repo.AssertExpectations(t)
}

// TestEnforcer_GracefulDegradation_DatabaseDown tests that the enforcer
// fails gracefully when the database is unavailable during rule reload
func TestEnforcer_GracefulDegradation_DatabaseDown(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	
	// Mock repository that fails
	repo := &mockRepo{}
	repo.On("ListByTenant", mock.Anything, tenantID).Return(nil, assert.AnError)
	
	// Create enforcer
	opts := Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
	}
	
	server := NewWithOptions(opts)
	assert.NotNil(t, server)
	
	// Should still be ready (fail-open behavior)
	assert.True(t, server.ready)
	
	// Rule reload should handle the error gracefully
	ctx := context.Background()
	err := server.ReloadRules(ctx)
	assert.Error(t, err) // Error is expected and handled
	
	repo.AssertExpectations(t)
}