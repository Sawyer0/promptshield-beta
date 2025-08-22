package eventstore

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/audit"
	"github.com/promptshield/promptshield/internal/testutil/fixtures"
)

// Helper function to create a test event that matches the current audit.Event
// schema (Type + Data). Extra fields are embedded in the Data map so they are
// still available for assertions.
func createTestEvent(tenantID, eventType string) audit.Event {
	return audit.Event{
		Type: eventType,
		Data: map[string]interface{}{
			"tenant_id": tenantID,
			"actor":     "test-actor",
			"resource":  "test-resource",
			"action":    "test-action",
			"test_key":  "test_value",
		},
	}
}

// Helper to open a test store
func openTestStore(t *testing.T) (*Store, func()) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test-events.log")

	store, err := Open(storePath)
	require.NoError(t, err)
	require.NotNil(t, store)

	cleanup := func() {
		err := store.Close()
		assert.NoError(t, err)
	}

	return store, cleanup
}

func TestEventStore_AppendAndGet(t *testing.T) {
	// TP-5.1 Append then Get sequence-1 returns identical event
	store, cleanup := openTestStore(t)
	defer cleanup()

	event := createTestEvent(fixtures.TenantID1.String(), "SCAN_START")

	// Append event
	seq, err := store.Append(context.Background(), event)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), seq)

	// Get event back
	rec, err := store.Get(context.Background(), seq)
	require.NoError(t, err)

	// Verify fields match
	assert.Equal(t, event.Type, rec.Event.Type)
	assert.Equal(t, event.Data, rec.Event.Data)
}

func TestEventStore_ConcurrentAppend(t *testing.T) {
	// TP-5.2 Append N events in goroutines → seq numbers strictly increasing
	store, cleanup := openTestStore(t)
	defer cleanup()

	numEvents := 100
	var wg sync.WaitGroup
	sequences := make([]uint64, numEvents)
	errors := make([]error, numEvents)

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := createTestEvent(
				fixtures.TenantID1.String(),
				fmt.Sprintf("CONCURRENT_EVENT_%d", idx),
			)
			seq, err := store.Append(context.Background(), event)
			sequences[idx] = seq
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Check no errors
	for i, err := range errors {
		assert.NoError(t, err, "Event %d should append without error", i)
	}

	// Verify all sequences are unique and valid
	seqMap := make(map[uint64]bool)
	for _, seq := range sequences {
		assert.Greater(t, seq, uint64(0), "Sequence should be positive")
		assert.False(t, seqMap[seq], "Sequence %d should be unique", seq)
		seqMap[seq] = true
	}

	// Verify we can retrieve random samples
	sampleIndices := []int{0, numEvents / 2, numEvents - 1}
	for _, idx := range sampleIndices {
		if sequences[idx] > 0 {
			rec, err := store.Get(context.Background(), sequences[idx])
			assert.NoError(t, err)
			assert.NotNil(t, rec)
			assert.Contains(t, rec.Event.Type, "CONCURRENT_EVENT")
		}
	}
}

func TestEventStore_GetOutOfRange(t *testing.T) {
	// TP-5.3 Get out-of-range returns error
	store, cleanup := openTestStore(t)
	defer cleanup()

	// Try to get sequence 0 (invalid)
	_, err := store.Get(context.Background(), 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sequence number")

	// Add one event
	event := createTestEvent(fixtures.TenantID1.String(), "TEST")
	seq, err := store.Append(context.Background(), event)
	require.NoError(t, err)

	// Try to get sequence beyond what exists
	_, err = store.Get(context.Background(), seq+100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEventStore_AppendWithCanceledContext(t *testing.T) {
	// TP-5.4 Append with ctx canceled returns ctx.Err
	store, cleanup := openTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	event := createTestEvent(fixtures.TenantID1.String(), "CANCELED")
	_, err := store.Append(ctx, event)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestEventStore_RaceCondition(t *testing.T) {
	// TP-5.5 Race test - run with: go test -race
	store, cleanup := openTestStore(t)
	defer cleanup()

	var readWg, writeWg sync.WaitGroup
	stopCh := make(chan struct{})

	// Counter for sequences
	var lastSeq atomic.Uint64

	// Writer goroutines
	for i := 0; i < 5; i++ {
		writeWg.Add(1)
		go func(id int) {
			defer writeWg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					event := createTestEvent(
						fixtures.TenantID1.String(),
						fmt.Sprintf("RACE_WRITE_%d", id),
					)
					seq, err := store.Append(context.Background(), event)
					if err == nil {
						lastSeq.Store(seq)
					}
				}
			}
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 5; i++ {
		readWg.Add(1)
		go func(id int) {
			defer readWg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					seq := lastSeq.Load()
					if seq > 0 {
						_, _ = store.Get(context.Background(), seq)
					}
				}
			}
		}(i)
	}

	// Let them race for a bit
	time.Sleep(100 * time.Millisecond)
	close(stopCh)

	writeWg.Wait()
	readWg.Wait()
}

func TestEventStore_LargeEvents(t *testing.T) {
	store, cleanup := openTestStore(t)
	defer cleanup()

	// Create event with large metadata
	largeMetadata := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeMetadata[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	event := audit.Event{
		Type: "LARGE_EVENT",
		Data: largeMetadata,
	}

	// Append large event
	seq, err := store.Append(context.Background(), event)
	require.NoError(t, err)

	// Retrieve and verify
	rec, err := store.Get(context.Background(), seq)
	require.NoError(t, err)
	assert.Equal(t, len(largeMetadata), len(rec.Event.Data))
}

func TestEventStore_SequentialAppends(t *testing.T) {
	store, cleanup := openTestStore(t)
	defer cleanup()

	numEvents := 50
	events := make([]audit.Event, numEvents)
	sequences := make([]uint64, numEvents)

	// Append events sequentially
	for i := 0; i < numEvents; i++ {
		events[i] = createTestEvent(
			fixtures.TenantID1.String(),
			fmt.Sprintf("SEQUENTIAL_%d", i),
		)
		seq, err := store.Append(context.Background(), events[i])
		require.NoError(t, err)
		sequences[i] = seq
	}

	// Verify sequences are strictly increasing
	for i := 1; i < numEvents; i++ {
		assert.Greater(t, sequences[i], sequences[i-1],
			"Sequence %d should be greater than sequence %d", i, i-1)
	}

	// Verify all events can be retrieved
	for i, seq := range sequences {
		rec, err := store.Get(context.Background(), seq)
		require.NoError(t, err)
		assert.Equal(t, events[i].Type, rec.Event.Type)
	}
}

func TestEventStore_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "persist-test.log")

	// Create store and add events
	store1, err := Open(storePath)
	require.NoError(t, err)

	event1 := createTestEvent(fixtures.TenantID1.String(), "PERSIST_1")
	seq1, err := store1.Append(context.Background(), event1)
	require.NoError(t, err)

	event2 := createTestEvent(fixtures.TenantID2.String(), "PERSIST_2")
	seq2, err := store1.Append(context.Background(), event2)
	require.NoError(t, err)

	err = store1.Close()
	require.NoError(t, err)

	// Reopen store
	store2, err := Open(storePath)
	require.NoError(t, err)
	defer store2.Close()

	// Verify events are still there
	r1, err := store2.Get(context.Background(), seq1)
	require.NoError(t, err)
	assert.Equal(t, "PERSIST_1", r1.Event.Type)

	r2, err := store2.Get(context.Background(), seq2)
	require.NoError(t, err)
	assert.Equal(t, "PERSIST_2", r2.Event.Type)

	// Add new event after reopening
	event3 := createTestEvent(fixtures.TenantID1.String(), "PERSIST_3")
	seq3, err := store2.Append(context.Background(), event3)
	require.NoError(t, err)
	assert.Greater(t, seq3, seq2, "New sequence should be greater than old")
}
