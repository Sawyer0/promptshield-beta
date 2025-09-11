//go:build integration
// +build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// TestHashChain_InitialEvent tests hash chain initialization with first event
func TestHashChain_InitialEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hash chain initialization test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// First event in chain should have empty previous hash
	firstEvent := &types.AuditEvent{
		ObjectID:   uuid.New(),
		TenantID:   &tenantID,
		Action:     "chain.init",
		ObjectType: "system",
		Timestamp:  time.Now(),
	}

	hash, err := hashChain.AppendEvent(ctx, firstEvent)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the event has empty previous hash
	stored, err := eventStore.GetByID(ctx, firstEvent.ObjectID.String())
	assert.NoError(t, err)
	assert.Empty(t, stored.PrevHash)
	assert.Equal(t, hash, stored.Hash)

	// Test chain info
	chainInfo, err := hashChain.GetChainInfo(ctx)
	assert.NoError(t, err)
	assert.Equal(t, hash, chainInfo.CurrentHash)
	assert.Equal(t, int64(1), chainInfo.TotalEvents)
	assert.NotNil(t, chainInfo.LastEvent)
}

// TestHashChain_SequentialEvents tests proper hash chaining across multiple events
func TestHashChain_SequentialEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sequential hash chain test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// Create a sequence of 5 events
	var events []*types.AuditEvent
	var hashes []string

	for i := 0; i < 5; i++ {
		event := &types.AuditEvent{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "sequence.test",
			ObjectType: "test",
			Metadata:   map[string]interface{}{"sequence": i},
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
		}

		hash, err := hashChain.AppendEvent(ctx, event)
		require.NoError(t, err)
		
		events = append(events, event)
		hashes = append(hashes, hash)
	}

	// Verify hash chain continuity
	for i, event := range events {
		stored, err := eventStore.GetByID(ctx, event.ObjectID.String())
		require.NoError(t, err)

		// Check hash matches
		assert.Equal(t, hashes[i], stored.Hash)

		// Check previous hash linkage
		if i == 0 {
			assert.Empty(t, stored.PrevHash, "First event should have empty previous hash")
		} else {
			assert.Equal(t, hashes[i-1], stored.PrevHash, "Event %d should reference previous hash", i)
		}
	}

	// Verify full chain integrity
	verification, err := hashChain.VerifyChain(ctx, hashes[0], hashes[4])
	assert.NoError(t, err)
	assert.True(t, verification.IsValid)
	assert.Equal(t, int64(5), verification.TotalEvents)
	assert.Equal(t, int64(5), verification.VerifiedEvents)
	assert.Empty(t, verification.BrokenLinks)
}

// TestHashChain_TamperDetection tests detection of various tampering scenarios
func TestHashChain_TamperDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping tamper detection test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// Create initial chain
	events := []*types.AuditEvent{
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "tamper.test1",
			ObjectType: "test",
			Metadata:   map[string]interface{}{"value": "original1"},
			Timestamp:  time.Now(),
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "tamper.test2",
			ObjectType: "test",
			Metadata:   map[string]interface{}{"value": "original2"},
			Timestamp:  time.Now().Add(1 * time.Second),
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "tamper.test3",
			ObjectType: "test",
			Metadata:   map[string]interface{}{"value": "original3"},
			Timestamp:  time.Now().Add(2 * time.Second),
		},
	}

	// Build legitimate chain
	var hashes []string
	for _, event := range events {
		hash, err := hashChain.AppendEvent(ctx, event)
		require.NoError(t, err)
		hashes = append(hashes, hash)
	}

	// Verify chain is initially valid
	verification, err := hashChain.VerifyChain(ctx, hashes[0], hashes[2])
	require.NoError(t, err)
	require.True(t, verification.IsValid)

	// Test Case 1: Modify event data (without updating hash)
	t.Run("DetectDataModification", func(t *testing.T) {
		// Simulate tampering by directly updating database
		_, err := db.Raw().Exec(ctx,
			`UPDATE audits SET metadata = $1 WHERE object_id = $2`,
			`{"value": "tampered1"}`, events[0].ObjectID)
		require.NoError(t, err)

		// Hash validation should fail
		validation, err := hashChain.ValidateEvent(ctx, events[0].ObjectID.String())
		assert.NoError(t, err)
		assert.False(t, validation.IsValid)
		assert.Contains(t, validation.Errors[0], "hash_mismatch")
	})

	// Test Case 2: Break hash chain linkage
	t.Run("DetectChainBreak", func(t *testing.T) {
		// Modify previous hash to break chain
		_, err := db.Raw().Exec(ctx,
			`UPDATE audits SET prev_hash = $1 WHERE object_id = $2`,
			"invalid_hash", events[1].ObjectID)
		require.NoError(t, err)

		// Chain verification should detect break
		verification, err := hashChain.VerifyChain(ctx, hashes[0], hashes[2])
		assert.NoError(t, err)
		assert.False(t, verification.IsValid)
		assert.NotEmpty(t, verification.BrokenLinks)
		
		// Find the broken link
		var foundBreak bool
		for _, link := range verification.BrokenLinks {
			if link.EventID == events[1].ObjectID.String() {
				foundBreak = true
				assert.Equal(t, hashes[0], link.ExpectedHash)
				assert.Equal(t, "invalid_hash", link.ActualHash)
			}
		}
		assert.True(t, foundBreak, "Should detect broken link at second event")
	})

	// Test Case 3: Hash substitution attack
	t.Run("DetectHashSubstitution", func(t *testing.T) {
		// Generate a different hash for same event
		fakeEvent := *events[2]
		fakeEvent.Metadata = map[string]interface{}{"value": "fake"}
		
		// Calculate what hash would be for modified data
		fakeHashChain := NewAuditHashChain(eventStore)
		fakeHash := fakeHashChain.(*pgAuditHashChain).calculateEventHash(&fakeEvent)

		// Replace hash in database
		_, err := db.Raw().Exec(ctx,
			`UPDATE audits SET hash = $1 WHERE object_id = $2`,
			fakeHash, events[2].ObjectID)
		require.NoError(t, err)

		// Validation should detect hash doesn't match actual data
		validation, err := hashChain.ValidateEvent(ctx, events[2].ObjectID.String())
		assert.NoError(t, err)
		assert.False(t, validation.IsValid)
		assert.Contains(t, validation.Errors[0], "hash_mismatch")
	})
}

// TestHashChain_RepairFunctionality tests hash chain repair capabilities
func TestHashChain_RepairFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hash chain repair test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// Create events with intentional corruption
	events := []*types.AuditEvent{
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "repair.test1",
			ObjectType: "test",
			Timestamp:  time.Now(),
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "repair.test2",
			ObjectType: "test",
			Timestamp:  time.Now().Add(1 * time.Second),
		},
	}

	// Add first event
	_, err := hashChain.AppendEvent(ctx, events[0])
	require.NoError(t, err)

	// Add second event
	_, err = hashChain.AppendEvent(ctx, events[1])
	require.NoError(t, err)

	// Corrupt the second event's hash
	_, err = db.Raw().Exec(ctx,
		`UPDATE audits SET hash = $1 WHERE object_id = $2`,
		"corrupted_hash", events[1].ObjectID)
	require.NoError(t, err)

	// Verify corruption detected
	validation, err := hashChain.ValidateEvent(ctx, events[1].ObjectID.String())
	require.NoError(t, err)
	require.False(t, validation.IsValid)

	// Test repair functionality
	// Note: The current implementation returns an error indicating repair methods are not implemented
	err = hashChain.RepairChain(ctx, events[1].ObjectID.String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repair chain functionality requires database update methods")
}

// TestHashChain_ConcurrentOperations tests hash chain under concurrent access
func TestHashChain_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent hash chain test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// Test concurrent event additions
	numGoroutines := 5
	eventsPerGoroutine := 3

	// Channel to collect all hashes
	hashChan := make(chan string, numGoroutines*eventsPerGoroutine)
	errChan := make(chan error, numGoroutines*eventsPerGoroutine)

	// Start concurrent event additions
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			for j := 0; j < eventsPerGoroutine; j++ {
				event := &types.AuditEvent{
					ObjectID:   uuid.New(),
					TenantID:   &tenantID,
					Action:     "concurrent.test",
					ObjectType: "test",
					Metadata:   map[string]interface{}{"worker": workerID, "event": j},
					Timestamp:  time.Now(),
				}

				hash, err := hashChain.AppendEvent(ctx, event)
				if err != nil {
					errChan <- err
				} else {
					hashChan <- hash
				}
			}
		}(i)
	}

	// Collect results
	var hashes []string
	var errors []error

	expectedCount := numGoroutines * eventsPerGoroutine
	for i := 0; i < expectedCount; i++ {
		select {
		case hash := <-hashChan:
			hashes = append(hashes, hash)
		case err := <-errChan:
			errors = append(errors, err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Verify results
	assert.Empty(t, errors, "No errors should occur during concurrent operations")
	assert.Len(t, hashes, expectedCount, "All operations should complete successfully")

	// Verify all hashes are unique
	hashSet := make(map[string]bool)
	for _, hash := range hashes {
		assert.False(t, hashSet[hash], "Hash should be unique: %s", hash)
		hashSet[hash] = true
	}

	// Verify final chain integrity
	count, err := eventStore.Count(ctx, &types.AuditFilter{TenantID: tenantID.String()})
	assert.NoError(t, err)
	assert.Equal(t, int64(expectedCount), count)

	// Verify chain info is consistent
	chainInfo, err := hashChain.GetChainInfo(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(expectedCount), chainInfo.TotalEvents)
	assert.NotEmpty(t, chainInfo.CurrentHash)
}

// TestHashChain_ExportValidation tests exported chain validation
func TestHashChain_ExportValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping export validation test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// Create events over time range
	baseTime := time.Now()
	events := []*types.AuditEvent{
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "export.event1",
			ObjectType: "test",
			Timestamp:  baseTime,
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "export.event2",
			ObjectType: "test",
			Timestamp:  baseTime.Add(30 * time.Minute),
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "export.event3",
			ObjectType: "test",
			Timestamp:  baseTime.Add(60 * time.Minute),
		},
	}

	// Build chain
	for _, event := range events {
		_, err := hashChain.AppendEvent(ctx, event)
		require.NoError(t, err)
	}

	// Export partial time range (middle event only)
	timeRange := types.TimeRange{
		Start: baseTime.Add(15 * time.Minute),
		End:   baseTime.Add(45 * time.Minute),
	}

	exportData, err := hashChain.ExportChain(ctx, timeRange)
	assert.NoError(t, err)
	assert.NotEmpty(t, exportData)

	// Verify export contains only events in range
	// Note: Implementation detail - the export function filters by time range
	// We would need to parse the JSON to verify the actual filtering
	assert.Contains(t, string(exportData), "export.event2")
	
	// Export full range
	fullRange := types.TimeRange{
		Start: baseTime.Add(-1 * time.Hour),
		End:   baseTime.Add(2 * time.Hour),
	}

	fullExport, err := hashChain.ExportChain(ctx, fullRange)
	assert.NoError(t, err)
	assert.NotEmpty(t, fullExport)
	
	// Full export should be larger than partial
	assert.Greater(t, len(fullExport), len(exportData))
}

// TestHashChain_EdgeCases tests various edge cases and error conditions
func TestHashChain_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hash chain edge cases test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)

	// Test nil event
	t.Run("NilEvent", func(t *testing.T) {
		_, err := hashChain.AppendEvent(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event cannot be nil")
	})

	// Test empty chain verification
	t.Run("EmptyChainVerification", func(t *testing.T) {
		verification, err := hashChain.VerifyChain(ctx, "start", "end")
		assert.NoError(t, err)
		assert.True(t, verification.IsValid)
		assert.Equal(t, int64(0), verification.TotalEvents)
		assert.Empty(t, verification.BrokenLinks)
	})

	// Test validation of non-existent event
	t.Run("NonExistentEventValidation", func(t *testing.T) {
		_, err := hashChain.ValidateEvent(ctx, uuid.New().String())
		assert.Error(t, err)
	})

	// Test chain info for empty chain
	t.Run("EmptyChainInfo", func(t *testing.T) {
		chainInfo, err := hashChain.GetChainInfo(ctx)
		assert.NoError(t, err)
		assert.Empty(t, chainInfo.CurrentHash)
		assert.Equal(t, int64(0), chainInfo.TotalEvents)
		assert.Nil(t, chainInfo.FirstEvent)
		assert.Nil(t, chainInfo.LastEvent)
	})

	// Test repair with invalid event ID
	t.Run("RepairInvalidEventID", func(t *testing.T) {
		err := hashChain.RepairChain(ctx, "invalid-uuid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid event ID")
	})
}