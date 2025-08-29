package postgres

import (
	"context"
	"database/sql"
)

// DB defines the database interface used by middleware and repositories
// Both *sql.DB and *Pool implement this interface
type DB interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (RowsScanner, error)
}

// RowsScanner interface for scanning multiple database rows
type RowsScanner interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
	Err() error
}

// Ensure Pool implements DB interface
var _ DB = (*Pool)(nil)

// sqlDBAdapter adapts standard sql.DB to our DB interface
type sqlDBAdapter struct {
	db *sql.DB
}

// NewDBFromSQL creates a DB interface from standard sql.DB
func NewDBFromSQL(db *sql.DB) DB {
	return &sqlDBAdapter{db: db}
}

func (a *sqlDBAdapter) QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner {
	row := a.db.QueryRowContext(ctx, query, args...)
	return &sqlRowAdapter{row: row}
}

func (a *sqlDBAdapter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

func (a *sqlDBAdapter) QueryContext(ctx context.Context, query string, args ...interface{}) (RowsScanner, error) {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsAdapter{rows: rows}, nil
}

// sqlRowAdapter adapts sql.Row to RowScanner interface
type sqlRowAdapter struct {
	row *sql.Row
}

func (r *sqlRowAdapter) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

// sqlRowsAdapter adapts sql.Rows to RowsScanner interface
type sqlRowsAdapter struct {
	rows *sql.Rows
}

func (r *sqlRowsAdapter) Next() bool {
	return r.rows.Next()
}

func (r *sqlRowsAdapter) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRowsAdapter) Close() error {
	return r.rows.Close()
}

func (r *sqlRowsAdapter) Err() error {
	return r.rows.Err()
}

// Ensure sql.DB adapter implements DB interface
var _ DB = (*sqlDBAdapter)(nil)