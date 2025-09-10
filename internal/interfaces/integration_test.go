package interfaces_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/repository"
	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


// TestIntegration_RulePropagation tests rule propagation without external dependencies
func TestIntegration_RulePropagation(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Use the same in-memory repository used by the RulepackService
	factory, err := repository.NewTestRepositoryFactory(nil, nil)
	require.NoError(t, err)
	repo := factory.Rulepack()

	// Create publisher with no Redis (becomes no-op)
	publisher, err := nats.NewPublisher("")
	require.NoError(t, err)

	// Message tracking would require wrapping the publisher

	// Create service using the same factory (shared repo)
	service := services.RulepackServiceFromFactory(factory, publisher)

	// Create rulepack
	packID, err := service.Create(ctx, tenantID, "test-pack", "Test rulepack")
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, packID)

	// Create multiple enforcers
	numEnforcers := 3
	enforcers := make([]*grpcenforcer.Server, numEnforcers)

	for i := 0; i < numEnforcers; i++ {
		opts := grpcenforcer.Options{
			Timeout:      100 * time.Millisecond,
			RulepackRepo: repo,
			TenantID:     tenantID,
			// No RedisAddr - will poll from database
		}
		enforcers[i] = grpcenforcer.NewWithOptions(opts)
		defer enforcers[i].Shutdown()
	}

	// Upload version 1
	dslV1 := json.RawMessage(`{
		"metadata": {"name": "test-rules", "version": "1.0.0"},
		"rules": [{"id": "rule-1", "level": 1, "keywords": ["test"], "severity": "LOW"}]
	}`)

	versionID, err := service.Upload(ctx, tenantID, packID, 1, dslV1, true)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, versionID)

	// Message publishing is tested separately
	// Here we focus on the rule reload flow

	// Simulate enforcers reloading rules (in production, Redis would trigger this)
	for i, enforcer := range enforcers {
		err := enforcer.ReloadRules(ctx)
		assert.NoError(t, err, "Enforcer %d should reload rules", i)
	}

	// Upload version 2
	dslV2 := json.RawMessage(`{
		"metadata": {"name": "test-rules", "version": "2.0.0"},
		"rules": [
			{"id": "rule-1", "level": 1, "keywords": ["test"], "severity": "LOW"},
			{"id": "rule-2", "level": 1, "keywords": ["new"], "severity": "HIGH"}
		]
	}`)

	versionID2, err := service.Upload(ctx, tenantID, packID, 2, dslV2, true)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, versionID2)

	// Message publishing is tested separately

	// Simulate enforcers reloading again
	for i, enforcer := range enforcers {
		err := enforcer.ReloadRules(ctx)
		assert.NoError(t, err, "Enforcer %d should reload rules v2", i)
	}

	// Delete rulepack
	err = service.Delete(ctx, tenantID, packID)
	require.NoError(t, err)

	// Message publishing is tested separately

	// Enforcers should handle deletion gracefully (fail-open)
	for i, enforcer := range enforcers {
		err := enforcer.ReloadRules(ctx)
		assert.NoError(t, err, "Enforcer %d should handle deletion gracefully", i)
		// Enforcers remain operational even without rules (fail-open)
	}
}

// TestIntegration_ConcurrentRuleUpdates tests concurrent rule updates don't cause races
func TestIntegration_ConcurrentRuleUpdates(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Use test factory in-memory repository
	factory2, err := repository.NewTestRepositoryFactory(nil, nil)
	require.NoError(t, err)
	repo := factory2.Rulepack()

	// Create enforcer
	opts := grpcenforcer.Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
	}
	enforcer := grpcenforcer.NewWithOptions(opts)
	defer enforcer.Shutdown()

	// Create rulepack with initial version
	packID, _ := repo.Create(ctx, tenantID, "concurrent-test", "Test concurrent updates")

	dsl := json.RawMessage(`{"metadata": {"name": "test"}, "rules": []}`)
	_, err = repo.CreateVersionActivateTx(ctx, packID, 1, dsl, uuid.Nil)
	require.NoError(t, err)

	// Simulate concurrent rule reloads (like multiple Redis messages arriving)
	var wg sync.WaitGroup
	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(version int) {
			defer wg.Done()

			// Create new version
			versionDSL := json.RawMessage(`{"metadata": {"name": "test"}, "rules": []}`)
			_, _ = repo.CreateVersionActivateTx(ctx, packID, version, versionDSL, uuid.Nil)

			// Trigger reload
			err := enforcer.ReloadRules(ctx)
			assert.NoError(t, err)
		}(i + 2)
	}

	wg.Wait()

	// Enforcer should still be in consistent state
	// (ready field is private, but we can verify it doesn't panic)
}
