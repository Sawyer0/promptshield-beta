//go:build integration
// +build integration

package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// TestAuditEventStore_BasicOperations tests basic audit event storage and retrieval
func TestAuditEventStore_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping audit integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	store := NewAuditEventStore(db)
	tenantID := uuid.New()
	actorID := uuid.New()

	// Create test audit event
	event := &types.AuditEvent{
		ObjectID:   uuid.New(),
		TenantID:   &tenantID,
		ActorID:    &actorID,
		ActorType:  "user",
		ActorEmail: "test@example.com",
		Action:     "policy.create",
		ObjectType: "policy",
		Before:     map[string]interface{}{"status": "draft"},
		After:      map[string]interface{}{"status": "active"},
		Metadata:   map[string]interface{}{"source": "api"},
		Timestamp:  time.Now(),
	}

	// Test Store
	err := store.Store(ctx, event)
	assert.NoError(t, err)

	// Test Retrieve
	filter := &types.AuditFilter{
		TenantID:   tenantID.String(),
		ObjectType: "policy",
		Limit:      10,
	}

	events, err := store.Retrieve(ctx, filter)
	assert.NoError(t, err)
	assert.Len(t, events, 1)

	retrieved := events[0]
	assert.Equal(t, event.Action, retrieved.Action)
	assert.Equal(t, event.ObjectType, retrieved.ObjectType)
	assert.Equal(t, event.ActorEmail, retrieved.ActorEmail)
	assert.Equal(t, event.Before, retrieved.Before)
	assert.Equal(t, event.After, retrieved.After)
	assert.Equal(t, event.Metadata, retrieved.Metadata)
}

// TestAuditEventStore_BulkOperations tests batch storage of audit events
func TestAuditEventStore_BulkOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping audit bulk test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	store := NewAuditEventStore(db)
	tenantID := uuid.New()

	// Create multiple events
	events := make([]*types.AuditEvent, 5)
	for i := 0; i < 5; i++ {
		events[i] = &types.AuditEvent{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			ActorID:    &uuid.Nil,
			ActorType:  "system",
			Action:     "policy.scan",
			ObjectType: "policy",
			Metadata:   map[string]interface{}{"sequence": i},
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
		}
	}

	// Test batch store
	err := store.StoreBatch(ctx, events)
	assert.NoError(t, err)

	// Test count
	filter := &types.AuditFilter{TenantID: tenantID.String()}
	count, err := store.Count(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Test retrieval with ordering
	retrieved, err := store.Retrieve(ctx, &types.AuditFilter{
		TenantID: tenantID.String(),
		Limit:    10,
	})
	assert.NoError(t, err)
	assert.Len(t, retrieved, 5)

	// Verify events are ordered by timestamp (latest first)
	for i := 1; i < len(retrieved); i++ {
		assert.True(t, retrieved[i-1].Timestamp.After(retrieved[i].Timestamp) || 
			retrieved[i-1].Timestamp.Equal(retrieved[i].Timestamp))
	}
}

// TestAuditEventStore_Filtering tests various filter combinations
func TestAuditEventStore_Filtering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping audit filtering test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	store := NewAuditEventStore(db)
	tenantID := uuid.New()
	actorID := uuid.New()

	// Create events with different properties
	baseTime := time.Now()
	events := []*types.AuditEvent{
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			ActorID:    &actorID,
			Action:     "policy.create",
			ObjectType: "policy",
			Timestamp:  baseTime,
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			ActorID:    &actorID,
			Action:     "policy.update",
			ObjectType: "policy",
			Timestamp:  baseTime.Add(1 * time.Minute),
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			ActorID:    &actorID,
			Action:     "rule.create",
			ObjectType: "rule",
			Timestamp:  baseTime.Add(2 * time.Minute),
		},
	}

	// Store events
	err := store.StoreBatch(ctx, events)
	require.NoError(t, err)

	// Test action filtering
	actionFilter := &types.AuditFilter{
		TenantID: tenantID.String(),
		Action:   "policy.create",
	}
	filtered, err := store.Retrieve(ctx, actionFilter)
	assert.NoError(t, err)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "policy.create", filtered[0].Action)

	// Test object type filtering
	objectFilter := &types.AuditFilter{
		TenantID:   tenantID.String(),
		ObjectType: "policy",
	}
	filtered, err = store.Retrieve(ctx, objectFilter)
	assert.NoError(t, err)
	assert.Len(t, filtered, 2)

	// Test time range filtering
	startTime := baseTime.Add(30 * time.Second)
	endTime := baseTime.Add(90 * time.Second)
	timeFilter := &types.AuditFilter{
		TenantID:  tenantID.String(),
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	filtered, err = store.Retrieve(ctx, timeFilter)
	assert.NoError(t, err)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "policy.update", filtered[0].Action)
}

// TestAuditHashChain_Integrity tests hash chain integrity verification
func TestAuditHashChain_Integrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hash chain test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// Create a sequence of events
	events := make([]*types.AuditEvent, 3)
	for i := 0; i < 3; i++ {
		events[i] = &types.AuditEvent{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			ActorType:  "system",
			Action:     "test.event",
			ObjectType: "test",
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
		}
	}

	// Test AppendEvent with hash chaining
	var hashes []string
	for _, event := range events {
		hash, err := hashChain.AppendEvent(ctx, event)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		hashes = append(hashes, hash)
	}

	// Test chain verification
	verification, err := hashChain.VerifyChain(ctx, hashes[0], hashes[2])
	assert.NoError(t, err)
	assert.True(t, verification.IsValid)
	assert.Equal(t, int64(3), verification.TotalEvents)
	assert.Equal(t, int64(3), verification.VerifiedEvents)
	assert.Empty(t, verification.BrokenLinks)

	// Test individual event validation
	for i, hash := range hashes {
		validation, err := hashChain.ValidateEvent(ctx, events[i].ObjectID.String())
		assert.NoError(t, err)
		assert.True(t, validation.IsValid)
		assert.Empty(t, validation.Errors)
		assert.Equal(t, hash, validation.EventHash)
	}
}

// TestAuditHashChain_CorruptionDetection tests detection of hash chain corruption
func TestAuditHashChain_CorruptionDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping corruption detection test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// Create valid event chain
	events := []*types.AuditEvent{
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "test.create",
			ObjectType: "test",
			Timestamp:  time.Now(),
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "test.update",
			ObjectType: "test",
			Timestamp:  time.Now().Add(1 * time.Second),
		},
	}

	// Add events to chain
	_, err := hashChain.AppendEvent(ctx, events[0])
	require.NoError(t, err)
	_, err = hashChain.AppendEvent(ctx, events[1])
	require.NoError(t, err)

	// Simulate corruption by manually modifying an event's data
	corruptedEvent := *events[1]
	corruptedEvent.Action = "test.corrupted"

	// Store corrupted event (simulating tampering)
	err = eventStore.Store(ctx, &corruptedEvent)
	require.NoError(t, err)

	// Verification should detect corruption
	validation, err := hashChain.ValidateEvent(ctx, events[1].ObjectID.String())
	assert.NoError(t, err)
	assert.False(t, validation.IsValid)
	assert.NotEmpty(t, validation.Errors)
	assert.Contains(t, validation.Errors[0], "hash_mismatch")
}

// TestAuditHashChain_ExportImport tests chain export functionality
func TestAuditHashChain_ExportImport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chain export test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	hashChain := NewAuditHashChain(eventStore)
	tenantID := uuid.New()

	// Create events
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Minute)
	
	events := []*types.AuditEvent{
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "export.test1",
			ObjectType: "test",
			Timestamp:  startTime.Add(1 * time.Minute),
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "export.test2",
			ObjectType: "test",
			Timestamp:  startTime.Add(2 * time.Minute),
		},
	}

	// Add to chain
	for _, event := range events {
		_, err := hashChain.AppendEvent(ctx, event)
		require.NoError(t, err)
	}

	// Test export
	timeRange := types.TimeRange{
		Start: startTime,
		End:   endTime,
	}

	exportData, err := hashChain.ExportChain(ctx, timeRange)
	assert.NoError(t, err)
	assert.NotEmpty(t, exportData)

	// Verify export format
	var chainExport types.ChainExport
	err = json.Unmarshal(exportData, &chainExport)
	assert.NoError(t, err)
	assert.Equal(t, timeRange, chainExport.TimeRange)
	assert.Equal(t, int64(2), chainExport.TotalEvents)
	assert.NotEmpty(t, chainExport.FirstHash)
	assert.NotEmpty(t, chainExport.LastHash)
	assert.Len(t, chainExport.Events, 2)
}

// TestAuditReporter_ComplianceReports tests compliance reporting functionality
func TestAuditReporter_ComplianceReports(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compliance reporting test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	reporter := NewAuditReporter(eventStore)
	tenantID := uuid.New()

	// Create audit events for compliance scenarios
	baseTime := time.Now()
	events := []*types.AuditEvent{
		// Data access events
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "data.access",
			ObjectType: "sensitive_data",
			Metadata:   map[string]interface{}{"data_type": "pii", "classification": "confidential"},
			Timestamp:  baseTime,
		},
		// Policy changes
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "policy.update",
			ObjectType: "security_policy",
			Before:     map[string]interface{}{"level": "medium"},
			After:      map[string]interface{}{"level": "high"},
			Timestamp:  baseTime.Add(1 * time.Hour),
		},
		// User activities
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "user.login",
			ObjectType: "user_session",
			Metadata:   map[string]interface{}{"ip": "192.168.1.100", "user_agent": "Mozilla/5.0"},
			Timestamp:  baseTime.Add(2 * time.Hour),
		},
	}

	// Store events
	err := eventStore.StoreBulk(ctx, events)
	require.NoError(t, err)

	// Test compliance report generation
	timeRange := types.TimeRange{
		Start: baseTime.Add(-1 * time.Hour),
		End:   baseTime.Add(3 * time.Hour),
	}

	report, err := reporter.GenerateComplianceReport(ctx, tenantID.String(), timeRange)
	assert.NoError(t, err)
	assert.NotNil(t, report)

	// Verify report structure
	assert.Equal(t, tenantID.String(), report.TenantID)
	assert.Equal(t, timeRange, report.TimeRange)
	assert.Equal(t, int64(3), report.TotalEvents)
	assert.NotEmpty(t, report.EventsByType)
	assert.NotEmpty(t, report.GeneratedAt)

	// Check event categorization
	assert.Contains(t, report.EventsByType, "data.access")
	assert.Contains(t, report.EventsByType, "policy.update")
	assert.Contains(t, report.EventsByType, "user.login")

	// Verify metrics
	assert.True(t, report.DataAccessCount > 0)
	assert.True(t, report.PolicyChangeCount > 0)
	assert.True(t, report.UserActivityCount > 0)
}

// TestAuditNotifier_AlertTriggering tests audit alert notification system
func TestAuditNotifier_AlertTriggering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping audit notification test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	notifier := NewAuditNotifier(eventStore)
	tenantID := uuid.New()

	// Create high-severity security events
	securityEvents := []*types.AuditEvent{
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "security.violation",
			ObjectType: "policy",
			Metadata:   map[string]interface{}{"severity": "HIGH", "type": "injection_attempt"},
			Timestamp:  time.Now(),
		},
		{
			ObjectID:   uuid.New(),
			TenantID:   &tenantID,
			Action:     "security.breach",
			ObjectType: "system",
			Metadata:   map[string]interface{}{"severity": "CRITICAL", "source": "external"},
			Timestamp:  time.Now().Add(1 * time.Minute),
		},
	}

	// Store security events
	err := eventStore.StoreBatch(ctx, securityEvents)
	require.NoError(t, err)

	// Test notification processing
	notifications, err := notifier.ProcessSecurityAlerts(ctx, tenantID.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, notifications)

	// Verify alert characteristics
	for _, notification := range notifications {
		assert.Equal(t, tenantID.String(), notification.TenantID)
		assert.NotEmpty(t, notification.AlertType)
		assert.NotEmpty(t, notification.Message)
		assert.True(t, notification.Severity == "HIGH" || notification.Severity == "CRITICAL")
		assert.False(t, notification.CreatedAt.IsZero())
	}
}

// TestAuditSystem_PerformanceUnderLoad tests audit system performance under concurrent load
func TestAuditSystem_PerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	eventStore := NewAuditEventStore(db)
	tenantID := uuid.New()

	// Test concurrent writes
	numGoroutines := 10
	eventsPerGoroutine := 20

	startTime := time.Now()

	// Generate events concurrently
	eventChan := make(chan *types.AuditEvent, numGoroutines*eventsPerGoroutine)
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			for j := 0; j < eventsPerGoroutine; j++ {
				event := &types.AuditEvent{
					ObjectID:   uuid.New(),
					TenantID:   &tenantID,
					Action:     "performance.test",
					ObjectType: "test",
					Metadata:   map[string]interface{}{"worker": workerID, "sequence": j},
					Timestamp:  time.Now(),
				}
				eventChan <- event
			}
		}(i)
	}

	// Collect events
	var events []*types.AuditEvent
	for i := 0; i < numGoroutines*eventsPerGoroutine; i++ {
		events = append(events, <-eventChan)
	}
	close(eventChan)

	// Store events in batch
	err := eventStore.StoreBatch(ctx, events)
	assert.NoError(t, err)

	duration := time.Since(startTime)
	t.Logf("Stored %d events in %v (%.2f events/sec)", 
		len(events), duration, float64(len(events))/duration.Seconds())

	// Verify all events were stored
	count, err := eventStore.Count(ctx, &types.AuditFilter{TenantID: tenantID.String()})
	assert.NoError(t, err)
	assert.Equal(t, int64(len(events)), count)

	// Test retrieval performance
	retrieveStart := time.Now()
	retrieved, err := eventStore.Retrieve(ctx, &types.AuditFilter{
		TenantID: tenantID.String(),
		Limit:    1000,
	})
	retrieveDuration := time.Since(retrieveStart)

	assert.NoError(t, err)
	assert.Len(t, retrieved, len(events))
	t.Logf("Retrieved %d events in %v", len(retrieved), retrieveDuration)

	// Performance assertions
	assert.Less(t, duration.Seconds(), 10.0, "Bulk insert should complete within 10 seconds")
	assert.Less(t, retrieveDuration.Seconds(), 2.0, "Retrieval should complete within 2 seconds")
}