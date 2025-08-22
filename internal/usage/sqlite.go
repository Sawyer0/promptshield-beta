package usage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore is a simple WAL-enabled SQLite implementation of UsageStore.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	if dsn == "" {
		dsn = "file:usage.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Create table with token tracking
	schema := `
CREATE TABLE IF NOT EXISTS usage_minute (
    tenant TEXT NOT NULL,
    route TEXT NOT NULL,
    ts_minute INTEGER NOT NULL,
    allow INTEGER NOT NULL DEFAULT 0,
    quarantine INTEGER NOT NULL DEFAULT 0,
    deny INTEGER NOT NULL DEFAULT 0,
    bytes BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant, route, ts_minute)
);

CREATE TABLE IF NOT EXISTS usage_tokens (
    tenant TEXT NOT NULL,
    route TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    ts_minute INTEGER NOT NULL,
    allow INTEGER NOT NULL DEFAULT 0,
    quarantine INTEGER NOT NULL DEFAULT 0,
    deny INTEGER NOT NULL DEFAULT 0,
    bytes BIGINT NOT NULL DEFAULT 0,
    prompt_tokens BIGINT NOT NULL DEFAULT 0,
    completion_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant, route, provider, model, ts_minute)
);`
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func floorToMinute(t time.Time) time.Time {
	return t.UTC().Truncate(time.Minute)
}

func (s *SQLiteStore) Record(ctx context.Context, tenant, route, decision string, bytes int64, t time.Time) error {
	if tenant == "" {
		tenant = "default"
	}
	if route == "" {
		route = "default"
	}
	ts := floorToMinute(t).Unix()
	// Upsert with decision-specific increment
	var col string
	switch decision {
	case DecisionAllow:
		col = "allow"
	case DecisionQuarantine:
		col = "quarantine"
	case DecisionDeny:
		col = "deny"
	default:
		col = "allow"
	}
	q := fmt.Sprintf(` //nolint:gosec // Column names are from closed set (allow/quarantine/deny), not user input
INSERT INTO usage_minute(tenant, route, ts_minute, %s, bytes)
VALUES(?,?,?,?,?)
ON CONFLICT(tenant, route, ts_minute)
DO UPDATE SET %s = %s + excluded.%s, bytes = usage_minute.bytes + excluded.bytes
`, col, col, col, col)
	_, err := s.db.ExecContext(ctx, q, tenant, route, ts, 1, bytes)
	if err != nil {
		return fmt.Errorf("record usage: %w", err)
	}
	return nil
}

// RecordTokens records usage with detailed token tracking for LLM billing/observability
func (s *SQLiteStore) RecordTokens(ctx context.Context, record Record) error {
	tenant := record.Tenant
	if tenant == "" {
		tenant = "default"
	}
	route := record.Route
	if route == "" {
		route = "default"
	}
	provider := record.Provider
	if provider == "" {
		provider = "unknown"
	}
	model := record.Model
	if model == "" {
		model = "unknown"
	}
	
	ts := floorToMinute(record.Timestamp).Unix()
	
	// Determine decision column
	var col string
	switch record.Decision {
	case DecisionAllow:
		col = "allow"
	case DecisionQuarantine:
		col = "quarantine"
	case DecisionDeny:
		col = "deny"
	default:
		col = "allow"
	}
	
	q := fmt.Sprintf(` //nolint:gosec // Column names are from closed set (allow/quarantine/deny), not user input
INSERT INTO usage_tokens(tenant, route, provider, model, ts_minute, %s, bytes, prompt_tokens, completion_tokens, total_tokens)
VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(tenant, route, provider, model, ts_minute)
DO UPDATE SET 
	%s = %s + excluded.%s,
	bytes = usage_tokens.bytes + excluded.bytes,
	prompt_tokens = usage_tokens.prompt_tokens + excluded.prompt_tokens,
	completion_tokens = usage_tokens.completion_tokens + excluded.completion_tokens,
	total_tokens = usage_tokens.total_tokens + excluded.total_tokens
`, col, col, col, col)
	
	_, err := s.db.ExecContext(ctx, q, tenant, route, provider, model, ts, 1, record.Bytes, 
		record.PromptTokens, record.CompletionTokens, record.TotalTokens)
	if err != nil {
		return fmt.Errorf("record token usage: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Close(ctx context.Context) error {
	return s.db.Close()
}

func (s *SQLiteStore) Query(ctx context.Context, q Query) (Result, error) {
	if q.End.Before(q.Start) {
		return Result{}, errors.New("invalid window")
	}
	// Determine time bucketing
	var bucketExpr string
	switch q.Interval {
	case IntervalHour:
		bucketExpr = "(ts_minute/3600)*3600"
	case IntervalDay:
		bucketExpr = "(ts_minute/86400)*86400"
	default:
		bucketExpr = "ts_minute"
	}
	// Build group by
	groupCols := []string{bucketExpr + " AS bucket_ts"}
	scanCols := []string{"SUM(allow) AS allow", "SUM(quarantine) AS quarantine", "SUM(deny) AS deny", "SUM(bytes) AS bytes"}
	includeTenant := false
	includeRoute := false
	for _, g := range q.GroupBy {
		switch g {
		case GroupByTenant:
			groupCols = append(groupCols, "tenant")
			includeTenant = true
		case GroupByRoute:
			groupCols = append(groupCols, "route")
			includeRoute = true
		case GroupByDecision:
			return Result{}, errors.New("group by decision not supported with current schema")
		}
	}
	// Base query without decision expansion
	where := []string{"ts_minute >= ?", "ts_minute < ?"}
	args := []any{floorToMinute(q.Start).Unix(), floorToMinute(q.End).Unix()}
	if q.Tenant != "" {
		where = append(where, "tenant = ?")
		args = append(args, q.Tenant)
	}
	sqlStr := fmt.Sprintf("SELECT %s, %s FROM usage_minute WHERE %s GROUP BY %s ORDER BY bucket_ts ASC", //nolint:gosec // Column names are controlled, query params are bound
		strings.Join(groupCols, ","), strings.Join(scanCols, ","), strings.Join(where, " AND "), strings.Join(groupCols, ","))
	rows, err := s.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return Result{}, fmt.Errorf("query usage: %w", err)
	}
	defer rows.Close()
	var out Result
	out.WindowStart = floorToMinute(q.Start)
	out.WindowEnd = floorToMinute(q.End)
	for rows.Next() {
		var (
			bucketTs                       int64
			tenantVal, routeVal            sqlNullString
			allow, quarantine, deny, bytes int64
		)
		// Build scan destination slice dynamically
		dest := []any{&bucketTs}
		if includeTenant {
			dest = append(dest, &tenantVal)
		}
		if includeRoute {
			dest = append(dest, &routeVal)
		}
		dest = append(dest, &allow, &quarantine, &deny, &bytes)
		if err := rows.Scan(dest...); err != nil {
			return Result{}, fmt.Errorf("scan usage: %w", err)
		}
		r := Row{
			IntervalStart: time.Unix(bucketTs, 0).UTC(),
			Count:         allow + quarantine + deny,
			Bytes:         bytes,
		}
		if includeTenant && tenantVal.Valid {
			r.Tenant = tenantVal.String
		}
		if includeRoute && routeVal.Valid {
			r.Route = routeVal.String
		}
		out.Rows = append(out.Rows, r)
	}
	return out, nil
}

// sqlNullString is a tiny wrapper to avoid importing database/sql just for NullString in callers.
type sqlNullString struct {
	String string
	Valid  bool
}
