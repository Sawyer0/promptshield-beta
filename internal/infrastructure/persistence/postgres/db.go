package postgres

import (
	"context"
	"fmt"

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
