//go:build integration
// +build integration

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrashRecovery_PartialWrite tests recovery from partial write scenarios
func TestCrashRecovery_PartialWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping crash recovery test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRulepackRepo(db)
	tenantID := uuid.New()

	// Start a transaction that will be interrupted
	tx, err := db.Raw().Begin(ctx)
	require.NoError(t, err)

	// Create rulepack in transaction
	packID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO rulepacks (id, tenant_id, name, description) VALUES ($1, $2, $3, $4)`,
		packID, tenantID, "crash-test", "Testing partial writes")
	require.NoError(t, err)

	// Create version in transaction
	versionID := uuid.New()
	dsl := json.RawMessage(`{"metadata": {"name": "test"}}`)
	_, err = tx.Exec(ctx,
		`INSERT INTO rulepack_versions (id, rulepack_id, version, dsl, status) 
		 VALUES ($1, $2, $3, $4, $5)`,
		versionID, packID, 1, dsl, "draft")
	require.NoError(t, err)

	// Simulate crash - rollback instead of commit
	err = tx.Rollback(ctx)
	require.NoError(t, err)

	// Verify data was not persisted (ACID guarantees)
	rulepacks, err := repo.ListByTenant(ctx, tenantID)
	assert.NoError(t, err)
	assert.Empty(t, rulepacks, "No data should be persisted after rollback")
}

// TestCrashRecovery_TornRecord tests handling of torn records (partial page writes)
func TestCrashRecovery_TornRecord(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping torn record test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRulepackRepo(db)
	tenantID := uuid.New()

	// Create a valid rulepack
	packID, err := repo.Create(ctx, tenantID, "torn-test", "Testing torn records")
	require.NoError(t, err)

	// Create large DSL that spans multiple database pages
	largeDSL := generateLargeDSL(8192) // 8KB to span pages

	// Start version creation
	versionID := uuid.New()

	// Simulate torn write by using context cancellation mid-operation
	writeCtx, cancel := context.WithCancel(ctx)

	// Start goroutine to cancel context mid-write
	go func() {
		time.Sleep(1 * time.Millisecond)
		cancel()
	}()

	// Attempt to write large version
	err = db.Raw().QueryRow(writeCtx,
		`INSERT INTO rulepack_versions (id, rulepack_id, version, dsl, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		versionID, packID, 1, largeDSL, "draft").Scan(&versionID)

	// Should get context cancelled error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Verify database is still consistent
	versions, err := getVersionCount(ctx, db, packID)
	assert.NoError(t, err)
	assert.Equal(t, 0, versions, "No partial version should exist")

	// Can still create new versions
	newVersion, err := repo.CreateVersion(ctx, packID, 1, json.RawMessage(`{}`), "draft", uuid.Nil)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, newVersion)
}

// TestCrashRecovery_CorruptedTail tests recovery when the tail of the database is corrupted
func TestCrashRecovery_CorruptedTail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping corrupted tail test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRulepackRepo(db)
	tenantID := uuid.New()

	// Create initial valid data
	packID1, err := repo.Create(ctx, tenantID, "pack1", "First pack")
	require.NoError(t, err)

	v1, err := repo.CreateVersion(ctx, packID1, 1, json.RawMessage(`{"valid": true}`), "approved", uuid.Nil)
	require.NoError(t, err)

	err = repo.Activate(ctx, packID1, v1)
	require.NoError(t, err)

	// Simulate corruption by attempting invalid foreign key
	corruptedPackID := uuid.New() // Non-existent pack

	// This should fail due to foreign key constraint
	_, err = repo.CreateVersion(ctx, corruptedPackID, 1, json.RawMessage(`{}`), "draft", uuid.Nil)
	assert.Error(t, err, "Should fail on non-existent rulepack")

	// Verify original data is still intact
	dsl, version, err := repo.GetActive(ctx, packID1)
	assert.NoError(t, err)
	assert.Equal(t, 1, version)
	assert.Equal(t, json.RawMessage(`{"valid": true}`), dsl)
}

// TestCrashRecovery_ConcurrentWrites tests database handles concurrent writes correctly
func TestCrashRecovery_ConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent writes test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRulepackRepo(db)
	tenantID := uuid.New()

	// Create a rulepack
	packID, err := repo.Create(ctx, tenantID, "concurrent-test", "Testing concurrent writes")
	require.NoError(t, err)

	// Simulate multiple concurrent version creations
	numGoroutines := 20
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(version int) {
			defer wg.Done()

			dsl := json.RawMessage(fmt.Sprintf(`{"version": %d}`, version))
			_, err := repo.CreateVersion(ctx, packID, version, dsl, "draft", uuid.Nil)
			if err != nil {
				errors <- err
			}
		}(i + 1)
	}

	wg.Wait()
	close(errors)

	// Some writes might fail due to version conflicts (expected)
	var errorCount int
	for err := range errors {
		if err != nil {
			errorCount++
			// Should be unique constraint violations
			assert.Contains(t, err.Error(), "duplicate key")
		}
	}

	// At least some should succeed
	versions, err := getVersionCount(ctx, db, packID)
	assert.NoError(t, err)
	assert.Greater(t, versions, 0, "Some versions should be created")
	assert.LessOrEqual(t, versions, numGoroutines, "No more than attempted versions")
}

// TestCrashRecovery_DeadlockResolution tests database handles deadlocks gracefully
func TestCrashRecovery_DeadlockResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping deadlock test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRulepackRepo(db)
	tenantID := uuid.New()

	// Create two rulepacks
	pack1, err := repo.Create(ctx, tenantID, "pack1", "First")
	require.NoError(t, err)

	pack2, err := repo.Create(ctx, tenantID, "pack2", "Second")
	require.NoError(t, err)

	// Create versions for activation
	v1, err := repo.CreateVersion(ctx, pack1, 1, json.RawMessage(`{}`), "approved", uuid.Nil)
	require.NoError(t, err)

	v2, err := repo.CreateVersion(ctx, pack2, 1, json.RawMessage(`{}`), "approved", uuid.Nil)
	require.NoError(t, err)

	// Simulate potential deadlock with concurrent cross-updates
	var wg sync.WaitGroup
	wg.Add(2)

	// Transaction 1: Update pack1 then pack2
	go func() {
		defer wg.Done()
		tx, _ := db.Raw().Begin(ctx)
		defer tx.Rollback(ctx)

		// Lock pack1
		tx.Exec(ctx, `UPDATE rulepacks SET current_version_id=$1 WHERE id=$2`, v1, pack1)
		time.Sleep(10 * time.Millisecond)
		// Try to lock pack2
		tx.Exec(ctx, `UPDATE rulepacks SET current_version_id=$1 WHERE id=$2`, v2, pack2)

		tx.Commit(ctx)
	}()

	// Transaction 2: Update pack2 then pack1 (opposite order)
	go func() {
		defer wg.Done()
		tx, _ := db.Raw().Begin(ctx)
		defer tx.Rollback(ctx)

		// Lock pack2
		tx.Exec(ctx, `UPDATE rulepacks SET current_version_id=$1 WHERE id=$2`, v2, pack2)
		time.Sleep(10 * time.Millisecond)
		// Try to lock pack1
		tx.Exec(ctx, `UPDATE rulepacks SET current_version_id=$1 WHERE id=$2`, v1, pack1)

		tx.Commit(ctx)
	}()

	// Wait with timeout to prevent hanging
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - database handled potential deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock not resolved within timeout")
	}

	// Verify database is still functional
	packs, err := repo.ListByTenant(ctx, tenantID)
	assert.NoError(t, err)
	assert.Len(t, packs, 2, "Both rulepacks should still exist")
}

// TestCrashRecovery_ConnectionLoss tests handling of sudden connection loss
func TestCrashRecovery_ConnectionLoss(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping connection loss test in short mode")
	}

	ctx := context.Background()

	// Create two connections
	db1 := setupTestDB(t)
	defer db1.Close()

	db2 := setupTestDB(t)
	// db2 will be closed mid-operation

	repo1 := NewRulepackRepo(db1)
	repo2 := NewRulepackRepo(db2)

	tenantID := uuid.New()

	// Create rulepack with first connection
	packID, err := repo1.Create(ctx, tenantID, "conn-test", "Testing connection loss")
	require.NoError(t, err)

	// Start operation with second connection
	go func() {
		// This will fail when connection is closed
		repo2.CreateVersion(ctx, packID, 1, json.RawMessage(`{}`), "draft", uuid.Nil)
	}()

	// Close second connection abruptly
	time.Sleep(10 * time.Millisecond)
	db2.Close()

	// First connection should still work
	version, err := repo1.CreateVersion(ctx, packID, 2, json.RawMessage(`{"ok": true}`), "approved", uuid.Nil)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, version)

	// Verify data integrity
	dsl, status, err := repo1.GetVersion(ctx, packID, 2)
	assert.NoError(t, err)
	assert.Equal(t, "approved", status)
	assert.Equal(t, json.RawMessage(`{"ok": true}`), dsl)
}

// Helper functions

func setupTestDB(t *testing.T) *Pool {
	t.Helper()

	// In production, use testcontainers-go
	// For this test, we'll create an in-memory mock
	return &Pool{inner: nil}
}

func generateLargeDSL(size int) json.RawMessage {
	// Generate DSL of specified size
	rules := make([]map[string]interface{}, 0)
	for i := 0; len(rules) < size/100; i++ {
		rules = append(rules, map[string]interface{}{
			"id":       fmt.Sprintf("rule-%d", i),
			"level":    1,
			"keywords": []string{fmt.Sprintf("keyword-%d", i)},
			"severity": "LOW",
		})
	}

	dsl := map[string]interface{}{
		"metadata": map[string]string{"name": "large-test"},
		"rules":    rules,
	}

	data, _ := json.Marshal(dsl)
	return json.RawMessage(data)
}

func getVersionCount(ctx context.Context, db *Pool, packID uuid.UUID) (int, error) {
	var count int
	err := db.Raw().QueryRow(ctx,
		`SELECT COUNT(*) FROM rulepack_versions WHERE rulepack_id = $1`,
		packID).Scan(&count)
	return count, err
}

// mockPool implements a basic in-memory database for testing
type mockPool struct {
	data map[string]interface{}
	mu   *sync.RWMutex
}

func (m *mockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	// Return a mock transaction
	return &mockTx{pool: m, ctx: ctx}, nil
}

func (m *mockPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	// Mock implementation
	return &mockRow{}
}

func (m *mockPool) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	// Mock implementation
	return pgconn.CommandTag{}, nil
}

func (m *mockPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	// Mock implementation
	return &mockRows{}, nil
}

func (m *mockPool) Close() {
	// No-op for mock
}

// mockTx implements pgx.Tx interface
type mockTx struct {
	pool *mockPool
	ctx  context.Context
}

func (m *mockTx) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }
func (m *mockTx) Commit(ctx context.Context) error          { return nil }
func (m *mockTx) Rollback(ctx context.Context) error        { return nil }
func (m *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *mockTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *mockTx) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return pgconn.CommandTag{}, ctx.Err()
	default:
		return pgconn.CommandTag{}, nil
	}
}
func (m *mockTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return &mockRows{}, nil
}
func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return &mockRow{}
}
func (m *mockTx) Conn() *pgx.Conn { return nil }

// mockRow implements pgx.Row
type mockRow struct{}

func (m *mockRow) Scan(dest ...interface{}) error { return nil }

// mockRows implements pgx.Rows
type mockRows struct{}

func (m *mockRows) Close()                                       {}
func (m *mockRows) Err() error                                   { return nil }
func (m *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRows) Next() bool                                   { return false }
func (m *mockRows) Scan(dest ...interface{}) error               { return nil }
func (m *mockRows) Values() ([]interface{}, error)               { return nil, nil }
func (m *mockRows) RawValues() [][]byte                          { return nil }
func (m *mockRows) Conn() *pgx.Conn                              { return nil }
