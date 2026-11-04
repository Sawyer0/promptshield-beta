package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresLogger persists audit events into a SQL table compatible with Postgres.
// We use the database/sql API to keep the dependency small and portable. The DSN
// should point to a Postgres-compatible driver. For local/dev, SQLite may also be
// used when DSN references a file and a sqlite driver is selected.

type PostgresLogger struct {
	mu    sync.Mutex
	db    *sql.DB
	table string
}

// NewPostgresLogger opens the database and optionally creates the audit table.
func NewPostgresLogger(dsn, table string, autoMigrate bool) (*PostgresLogger, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("empty DSN")
	}
	if strings.TrimSpace(table) == "" {
		table = "audit_events"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	l := &PostgresLogger{db: db, table: table}
	if autoMigrate {
		if err := l.ensureTable(); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return l, nil
}

func (l *PostgresLogger) ensureTable() error {
	ddl := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  id BIGSERIAL PRIMARY KEY,
  ts TIMESTAMPTZ NOT NULL,
  type TEXT NOT NULL,
  hash TEXT NOT NULL,
  prev_hash TEXT,
  data JSONB NOT NULL
)`, l.table)
	_, err := l.db.Exec(ddl)
	return err
}

func (l *PostgresLogger) Log(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.Data != nil {
		e.Data = SanitizeMap(e.Data)
	}
	e.Timestamp = time.Now().UTC()
	e.Hash = hashEvent(e)
	b, err := json.Marshal(e.Data)
	if err != nil {
		return err
	}
	_, err = l.db.ExecContext(context.Background(),
		fmt.Sprintf("INSERT INTO %s (ts,type,hash,prev_hash,data) VALUES ($1,$2,$3,$4,$5)", l.table),
		e.Timestamp, e.Type, e.Hash, e.PrevHash, string(b))
	return err
}

func (l *PostgresLogger) Close() error { return l.db.Close() }
