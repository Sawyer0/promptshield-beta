package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"
)

// DecisionRow is a minimal schema for analytics
type DecisionRow struct {
	OccurredAt    time.Time
	TenantID      string
	Endpoint      string
	Method        string
	Decision      string
	Reason        string
	LatencyMs     int64
	BytesRequest  int64
	BytesResponse int64
}

type ClickHouseWriter struct {
	db    *sql.DB
	table string
}

func NewClickHouseWriterFromEnv() (*ClickHouseWriter, error) {
	dsn := os.Getenv("PS_CLICKHOUSE_DSN")
	if dsn == "" {
		return nil, nil
	}
	table := os.Getenv("PS_CLICKHOUSE_DECISIONS_TABLE")
	if table == "" {
		table = "ps_decisions"
	}
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, nil
	}
	if err := db.PingContext(context.Background()); err != nil {
		return nil, nil
	}
	return &ClickHouseWriter{db: db, table: table}, nil
}

func (w *ClickHouseWriter) InsertDecision(ctx context.Context, r DecisionRow) error {
	if w == nil || w.db == nil {
		return nil
	}
	q := fmt.Sprintf("INSERT INTO %s (occurred_at, tenant_id, endpoint, method, decision, reason, latency_ms, bytes_request, bytes_response) VALUES (?,?,?,?,?,?,?,?,?)", w.table)
	_, err := w.db.ExecContext(ctx, q, r.OccurredAt, r.TenantID, r.Endpoint, r.Method, r.Decision, r.Reason, r.LatencyMs, r.BytesRequest, r.BytesResponse)
	return err
}

func (w *ClickHouseWriter) Close() error {
	if w == nil || w.db == nil {
		return nil
	}
	return w.db.Close()
}
