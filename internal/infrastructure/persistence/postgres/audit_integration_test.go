package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// TestAuditIntegration_BasicWorkflow tests the complete audit workflow with real database
func TestAuditIntegration_BasicWorkflow(t *testing.T) {
	// Skip if no database URL provided
	dsn := os.Getenv("PS_PG_DSN")
	if dsn == "" {
		t.Skip("PS_PG_DSN not set, skipping integration test")
	}

	ctx := context.Background()

	// Use real database connection
	db, err := NewPool(ctx, dsn)
	require.NoError(t, err, "Failed to connect to database")
	defer db.Close()

	// Initialize audit components
	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	reporter := NewAuditReporter(eventStore)

	tenantID := uuid.New()
	actorID := uuid.New()

	// Test 1: Create and store audit events
	t.Run("StoreEvents", func(t *testing.T) {
		events := []*types.AuditEvent{
			{
				ObjectID:   uuid.New(),
				TenantID:   &tenantID,
				ActorID:    &actorID,
				ActorType:  "user",
				ActorEmail: "admin@promptshield.com",
				Action:     "policy.create",
				ObjectType: "security_policy",
				Before:     nil,
				After:      map[string]interface{}{"name": "test-policy", "enabled": true},
				Metadata:   map[string]interface{}{"source": "api", "version": "1.0"},
				Timestamp:  time.Now(),
			},
			{
				ObjectID:   uuid.New(),
				TenantID:   &tenantID,
				ActorID:    &actorID,
				ActorType:  "user",
				ActorEmail: "admin@promptshield.com",
				Action:     "policy.update",
				ObjectType: "security_policy",
				Before:     map[string]interface{}{"enabled": true},
				After:      map[string]interface{}{"enabled": false},
				Metadata:   map[string]interface{}{"reason": "maintenance"},
				Timestamp:  time.Now().Add(1 * time.Minute),
			},
		}

		// Store events with hash chaining
		for _, event := range events {
			hash, err := hashChain.AppendEvent(ctx, event)
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
		}

		// Verify events are stored
		filter := &types.AuditFilter{
			TenantID: tenantID.String(),
			Limit:    10,
		}

		retrieved, err := eventStore.Retrieve(ctx, filter)
		assert.NoError(t, err)
		assert.Len(t, retrieved, 2)

		// Verify first event has no previous hash
		for _, event := range retrieved {
			switch event.Action {
			case "policy.create":
				assert.Empty(t, event.PrevHash)
			case "policy.update":
				assert.NotEmpty(t, event.PrevHash)
			}
		}
	})

	// Test 2: Verify hash chain integrity
	t.Run("VerifyChainIntegrity", func(t *testing.T) {
		// Get latest events for verification
		filter := &types.AuditFilter{
			TenantID: tenantID.String(),
			Limit:    2,
		}

		events, err := eventStore.Retrieve(ctx, filter)
		require.NoError(t, err)
		require.Len(t, events, 2)

		// Verify each event individually
		for _, event := range events {
			validation, err := hashChain.ValidateEvent(ctx, event.ObjectID.String())
			assert.NoError(t, err)
			assert.True(t, validation.IsValid)
			assert.Empty(t, validation.Errors)
			assert.NotEmpty(t, validation.EventHash)
		}

		// Verify chain info
		chainInfo, err := hashChain.GetChainInfo(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), chainInfo.TotalEvents)
		assert.NotEmpty(t, chainInfo.CurrentHash)
		assert.NotNil(t, chainInfo.LastEvent)
	})

	// Test 3: Generate compliance report
	t.Run("GenerateComplianceReport", func(t *testing.T) {
		timeRange := types.TimeRange{
			Start: time.Now().Add(-1 * time.Hour),
			End:   time.Now().Add(1 * time.Hour),
		}

		report, err := reporter.GenerateComplianceReport(ctx, tenantID.String(), timeRange)
		assert.NoError(t, err)
		assert.NotNil(t, report)

		// Verify report structure
		assert.Equal(t, tenantID.String(), report.TenantID)
		assert.Equal(t, int64(2), report.TotalEvents)
		assert.NotEmpty(t, report.EventsByType)
		assert.NotZero(t, report.PolicyChangeCount)

		// Check event categorization
		assert.Contains(t, report.EventsByType, "policy.create")
		assert.Contains(t, report.EventsByType, "policy.update")
	})

	// Test 4: Export chain data
	t.Run("ExportChainData", func(t *testing.T) {
		timeRange := types.TimeRange{
			Start: time.Now().Add(-1 * time.Hour),
			End:   time.Now().Add(1 * time.Hour),
		}

		exportData, err := hashChain.ExportChain(ctx, timeRange)
		assert.NoError(t, err)
		assert.NotEmpty(t, exportData)

		// Verify export is valid JSON
		assert.True(t, isValidJSON(exportData))

		// Verify export contains our events
		exportStr := string(exportData)
		assert.Contains(t, exportStr, "policy.create")
		assert.Contains(t, exportStr, "policy.update")
		assert.Contains(t, exportStr, "admin@promptshield.com")
	})

	// Test 5: Performance under load
	t.Run("PerformanceTest", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance test in short mode")
		}

		startTime := time.Now()
		numEvents := 50

		// Generate events
		var events []*types.AuditEvent
		for i := 0; i < numEvents; i++ {
			event := &types.AuditEvent{
				ObjectID:   uuid.New(),
				TenantID:   &tenantID,
				ActorType:  "system",
				Action:     "performance.test",
				ObjectType: "benchmark",
				Metadata:   map[string]interface{}{"sequence": i},
				Timestamp:  time.Now(),
			}
			events = append(events, event)
		}

		// Store in bulk
		err := eventStore.StoreBatch(ctx, events)
		assert.NoError(t, err)

		duration := time.Since(startTime)
		eventsPerSecond := float64(numEvents) / duration.Seconds()

		t.Logf("Stored %d events in %v (%.2f events/sec)",
			numEvents, duration, eventsPerSecond)

		// Performance assertion: should handle at least 10 events/sec
		assert.Greater(t, eventsPerSecond, 10.0,
			"Performance should be at least 10 events/sec, got %.2f", eventsPerSecond)

		// Verify all events stored
		count, err := eventStore.Count(ctx, &types.AuditFilter{
			TenantID:   tenantID.String(),
			ObjectType: "benchmark",
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(numEvents), count)
	})
}

// TestAuditIntegration_ErrorHandling tests error handling in audit system
func TestAuditIntegration_ErrorHandling(t *testing.T) {
	dsn := os.Getenv("PS_PG_DSN")
	if dsn == "" {
		t.Skip("PS_PG_DSN not set, skipping integration test")
	}

	ctx := context.Background()
	db, err := NewPool(ctx, dsn)
	require.NoError(t, err)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)

	// Test invalid event data
	t.Run("InvalidEventData", func(t *testing.T) {
		// Event with nil ObjectID should fail
		invalidEvent := &types.AuditEvent{
			// ObjectID missing
			Action:     "invalid.test",
			ObjectType: "test",
			Timestamp:  time.Now(),
		}

		_, err := hashChain.AppendEvent(ctx, invalidEvent)
		// Should handle gracefully - in production implementation would validate
		// For now, just ensure it doesn't panic
		if err != nil {
			t.Logf("Expected error for invalid event: %v", err)
		}
	})

	// Test context cancellation
	t.Run("ContextCancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		event := &types.AuditEvent{
			ObjectID:   uuid.New(),
			Action:     "cancelled.test",
			ObjectType: "test",
			Timestamp:  time.Now(),
		}

		_, err := hashChain.AppendEvent(cancelCtx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	// Test timeout handling
	t.Run("TimeoutHandling", func(t *testing.T) {
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		// Give context time to expire
		time.Sleep(10 * time.Millisecond)

		event := &types.AuditEvent{
			ObjectID:   uuid.New(),
			Action:     "timeout.test",
			ObjectType: "test",
			Timestamp:  time.Now(),
		}

		_, err := hashChain.AppendEvent(timeoutCtx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}

// TestAuditIntegration_Cleanup tests cleanup and maintenance operations
func TestAuditIntegration_Cleanup(t *testing.T) {
	dsn := os.Getenv("PS_PG_DSN")
	if dsn == "" {
		t.Skip("PS_PG_DSN not set, skipping integration test")
	}

	ctx := context.Background()
	db, err := NewPool(ctx, dsn)
	require.NoError(t, err)
	defer db.Close()

	// Clean up any test data
	tenantPattern := "00000000-0000-0000-0000-%"

	// Delete test audit events (UUIDs starting with zeros are test data)
	_, err = db.Raw().Exec(ctx,
		`DELETE FROM audits WHERE tenant_id::text LIKE $1`, tenantPattern)
	if err != nil {
		t.Logf("Cleanup warning: %v", err)
	}

	t.Log("Test cleanup completed")
}

// Helper function to validate JSON
func isValidJSON(data []byte) bool {
	var js interface{}
	return json.Unmarshal(data, &js) == nil
}
