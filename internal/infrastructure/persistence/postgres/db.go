package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps pgxpool.Pool to allow future instrumentation.
type Pool struct{ inner *pgxpool.Pool }

func NewPool(ctx context.Context, dsn string) (*Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres dsn required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return &Pool{inner: pool}, nil
}

func (p *Pool) Close() {
	if p != nil && p.inner != nil {
		p.inner.Close()
	}
}

func (p *Pool) Raw() *pgxpool.Pool { return p.inner }

// QueryRowContext implements sql-like interface for middleware compatibility
func (p *Pool) QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner {
	row := p.inner.QueryRow(ctx, query, args...)
	return &pgxRowAdapter{row: row}
}

// ExecContext implements sql-like interface for middleware compatibility  
func (p *Pool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ct, err := p.inner.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResultAdapter{ct: ct}, nil
}

// QueryContext implements sql-like interface for middleware compatibility
func (p *Pool) QueryContext(ctx context.Context, query string, args ...interface{}) (RowsScanner, error) {
	rows, err := p.inner.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRowsAdapter{rows: rows}, nil
}

// RowScanner interface for scanning database rows
type RowScanner interface {
	Scan(dest ...interface{}) error
}

// pgxRowAdapter adapts pgx.Row to RowScanner interface
type pgxRowAdapter struct {
	row pgx.Row
}

func (r *pgxRowAdapter) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

// pgxResultAdapter adapts pgconn.CommandTag to sql.Result interface
type pgxResultAdapter struct {
	ct pgconn.CommandTag
}

func (r *pgxResultAdapter) LastInsertId() (int64, error) {
	return 0, fmt.Errorf("LastInsertId not supported")
}

func (r *pgxResultAdapter) RowsAffected() (int64, error) {
	return r.ct.RowsAffected(), nil
}

// pgxRowsAdapter adapts pgx.Rows to RowsScanner interface
type pgxRowsAdapter struct {
	rows pgx.Rows
}

func (r *pgxRowsAdapter) Next() bool {
	return r.rows.Next()
}

func (r *pgxRowsAdapter) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *pgxRowsAdapter) Close() error {
	r.rows.Close()
	return nil
}

func (r *pgxRowsAdapter) Err() error {
	return r.rows.Err()
}
