package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestDB wraps a database connection for testing
type TestDB struct {
	pool   *pgxpool.Pool
	tx     pgx.Tx
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTestDB creates a new test database connection
func NewTestDB(t *testing.T) *TestDB {
	dsn := os.Getenv("PS_TEST_PG_DSN")
	if dsn == "" {
		// Fallback to PS_PG_DSN for developer convenience
		dsn = os.Getenv("PS_PG_DSN")
	}
	if dsn == "" {
		t.Skip("PS_TEST_PG_DSN/PS_PG_DSN not set, skipping database tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	// Test the connection
	err = pool.Ping(ctx)
	require.NoError(t, err)

	return &TestDB{
		pool:   pool,
		ctx:    ctx,
		cancel: cancel,
	}
}

// helper to check if a table exists
func (tdb *TestDB) tableExists(t *testing.T, table string) bool {
	var exists bool
	err := tdb.pool.QueryRow(tdb.ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`,
		table,
	).Scan(&exists)
	require.NoError(t, err)
	return exists
}

// EnsureTestSchema updates tables/columns needed by tests idempotently
func (tdb *TestDB) EnsureTestSchema(t *testing.T) {
	// Ensure audits has expected columns used by audit services
	tdb.Exec(t, `ALTER TABLE audits ADD COLUMN IF NOT EXISTS actor_type TEXT`)
	tdb.Exec(t, `ALTER TABLE audits ADD COLUMN IF NOT EXISTS before_data JSONB`)
	// Allow update-by-ID semantics in tests without tenant RLS interference
	tdb.Exec(t, `ALTER TABLE rulepack_assignments DISABLE ROW LEVEL SECURITY`)
	tdb.Exec(t, `ALTER TABLE audits ADD COLUMN IF NOT EXISTS after_data JSONB`)
	tdb.Exec(t, `ALTER TABLE audits ADD COLUMN IF NOT EXISTS hash TEXT`)
	tdb.Exec(t, `ALTER TABLE audits ADD COLUMN IF NOT EXISTS prev_hash TEXT`)
}

// BeginTx starts a transaction for the test
func (tdb *TestDB) BeginTx(t *testing.T) {
	tx, err := tdb.pool.Begin(tdb.ctx)
	require.NoError(t, err)
	tdb.tx = tx
}

// RollbackTx rolls back the transaction
func (tdb *TestDB) RollbackTx() {
	if tdb.tx != nil {
		tdb.tx.Rollback(tdb.ctx)
	}
}

// Close closes the database connection
func (tdb *TestDB) Close() {
	tdb.RollbackTx()
	if tdb.pool != nil {
		tdb.pool.Close()
	}
	if tdb.cancel != nil {
		tdb.cancel()
	}
}

// Exec executes a query on the test database
func (tdb *TestDB) Exec(t *testing.T, query string, args ...interface{}) {
	var err error
	if tdb.tx != nil {
		_, err = tdb.tx.Exec(tdb.ctx, query, args...)
	} else {
		_, err = tdb.pool.Exec(tdb.ctx, query, args...)
	}
	require.NoError(t, err)
}

// Query executes a query and returns rows
func (tdb *TestDB) Query(t *testing.T, query string, args ...interface{}) pgx.Rows {
	var rows pgx.Rows
	var err error
	if tdb.tx != nil {
		rows, err = tdb.tx.Query(tdb.ctx, query, args...)
	} else {
		rows, err = tdb.pool.Query(tdb.ctx, query, args...)
	}
	require.NoError(t, err)
	return rows
}

// QueryRow executes a query and returns a single row
func (tdb *TestDB) QueryRow(t *testing.T, query string, args ...interface{}) pgx.Row {
	if tdb.tx != nil {
		return tdb.tx.QueryRow(tdb.ctx, query, args...)
	}
	return tdb.pool.QueryRow(tdb.ctx, query, args...)
}

// RunWithTx runs a test function within a transaction that gets rolled back
func (tdb *TestDB) RunWithTx(t *testing.T, fn func(*testing.T)) {
	tdb.BeginTx(t)
	defer tdb.RollbackTx()
	fn(t)
}

// SeedTestData creates basic test data
func (tdb *TestDB) SeedTestData(t *testing.T) (tenantID uuid.UUID, userID uuid.UUID, rulepackID uuid.UUID, versionID uuid.UUID) {
	// Create test tenant
	tenantID = uuid.New()
	tenantName := fmt.Sprintf("Test Tenant %s", tenantID.String())
	tdb.Exec(t, `
		INSERT INTO tenants (id, name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (name) DO NOTHING
	`,
		tenantID,
		tenantName,
		"active",
		time.Now(),
		time.Now(),
	)

	// Create test user (consolidated schema)
	userID = uuid.New()
	userEmail := fmt.Sprintf("test+%s@example.com", tenantID.String())
	tdb.Exec(t, `
		INSERT INTO users (id, tenant_id, email, full_name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`,
		userID,
		tenantID,
		userEmail,
		"Test User",
		"user",
		time.Now(),
		time.Now(),
	)

	// Create tenant membership if table exists
	if tdb.tableExists(t, "tenant_memberships") {
		tdb.Exec(t, `
			INSERT INTO tenant_memberships (tenant_id, user_id, role, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tenant_id, user_id) DO NOTHING
		`,
			tenantID,
			userID,
			"owner",
			time.Now(),
			time.Now(),
		)
	}

	// Create test rulepack (include tenant_id)
	rulepackID = uuid.New()
	tdb.Exec(t, `
		INSERT INTO rulepacks (id, tenant_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`,
		rulepackID,
		tenantID,
		"Test RulePack",
		"Test description",
		time.Now(),
		time.Now(),
	)

	// Create test rulepack version (dsl JSONB + created_by)
	versionID = uuid.New()
	dsl := `{"rules":[]}`
	tdb.Exec(t, `
		INSERT INTO rulepack_versions (id, rulepack_id, version, dsl, status, created_by, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`,
		versionID,
		rulepackID,
		1,
		dsl,
		"approved",
		userID,
		time.Now(),
	)

	return tenantID, userID, rulepackID, versionID
}

// CleanupTestData removes test data
func (tdb *TestDB) CleanupTestData(t *testing.T) {
	// Best-effort cleanup of recent test data using time window
	tdb.Exec(t, "DELETE FROM rulepack_assignments WHERE created_at > NOW() - INTERVAL '1 hour'")
	tdb.Exec(t, "DELETE FROM rulepack_versions WHERE created_at > NOW() - INTERVAL '1 hour'")
	tdb.Exec(t, "DELETE FROM rulepacks WHERE created_at > NOW() - INTERVAL '1 hour'")
	tdb.Exec(t, "DELETE FROM tenant_memberships WHERE created_at > NOW() - INTERVAL '1 hour'")
	tdb.Exec(t, "DELETE FROM users WHERE created_at > NOW() - INTERVAL '1 hour'")
	tdb.Exec(t, "DELETE FROM tenants WHERE created_at > NOW() - INTERVAL '1 hour'")
}

// AssertTableEmpty asserts that a table is empty
func (tdb *TestDB) AssertTableEmpty(t *testing.T, tableName string) {
	var count int
	err := tdb.QueryRow(t, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

// AssertTableCount asserts that a table has a specific number of rows
func (tdb *TestDB) AssertTableCount(t *testing.T, tableName string, expectedCount int) {
	var count int
	err := tdb.QueryRow(t, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, expectedCount, count)
}
