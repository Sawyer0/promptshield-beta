package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// TenantDB wraps a database connection with automatic tenant context setting
type TenantDB struct {
	db *sql.DB
}

// NewTenantDB creates a new tenant-aware database wrapper
func NewTenantDB(db *sql.DB) *TenantDB {
	return &TenantDB{db: db}
}

// QueryContext executes a query with tenant context automatically set
func (tdb *TenantDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if err := tdb.setTenantContextFromCtx(ctx); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	return tdb.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns at most one row with tenant context
func (tdb *TenantDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if err := tdb.setTenantContextFromCtx(ctx); err != nil {
		slog.Error("Failed to set tenant context for query", "error", err)
		// Return a row that will produce an error when scanned
		return tdb.db.QueryRow("SELECT NULL WHERE FALSE")
	}
	return tdb.db.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query without returning any rows with tenant context
func (tdb *TenantDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if err := tdb.setTenantContextFromCtx(ctx); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	return tdb.db.ExecContext(ctx, query, args...)
}

// BeginTx starts a transaction with tenant context
func (tdb *TenantDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TenantTx, error) {
	if err := tdb.setTenantContextFromCtx(ctx); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	
	tx, err := tdb.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	
	return &TenantTx{tx: tx, ctx: ctx}, nil
}

// GetDB returns the underlying database connection (use with caution)
func (tdb *TenantDB) GetDB() *sql.DB {
	return tdb.db
}

// setTenantContextFromCtx extracts tenant ID from context and sets it in the database
func (tdb *TenantDB) setTenantContextFromCtx(ctx context.Context) error {
	// Extract tenant ID from context
	tenantIDVal := ctx.Value("db.tenant_id")
	if tenantIDVal == nil {
		// No tenant context - this might be a system/admin operation
		slog.Debug("No tenant context in request - system operation")
		return nil
	}
	
	tenantIDStr, ok := tenantIDVal.(string)
	if !ok {
		return fmt.Errorf("invalid tenant ID type in context")
	}
	
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}
	
	// Set tenant context in database for RLS
	_, err = tdb.db.ExecContext(ctx, "SELECT set_tenant_context($1::uuid)", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set tenant context in database: %w", err)
	}
	
	slog.Debug("Set tenant context for database operation", "tenant_id", tenantID)
	return nil
}

// TenantTx wraps a database transaction with tenant context
type TenantTx struct {
	tx  *sql.Tx
	ctx context.Context
}

// QueryContext executes a query within the transaction
func (ttx *TenantTx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return ttx.tx.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns at most one row within the transaction
func (ttx *TenantTx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return ttx.tx.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query without returning any rows within the transaction
func (ttx *TenantTx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return ttx.tx.ExecContext(ctx, query, args...)
}

// Commit commits the transaction
func (ttx *TenantTx) Commit() error {
	return ttx.tx.Commit()
}

// Rollback rolls back the transaction
func (ttx *TenantTx) Rollback() error {
	return ttx.tx.Rollback()
}

// GetTx returns the underlying transaction (use with caution)
func (ttx *TenantTx) GetTx() *sql.Tx {
	return ttx.tx
}