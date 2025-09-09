package audit

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// NewLoggerFromEnv constructs a durable audit logger based on environment variables.
//
// Configuration (env):
//
//	PS_AUDIT_SINK:   kafka | postgres | file | none (default: none)
//	File:
//	  PS_AUDIT_FILE: path to audit log file (rotating via lumberjack)
//	Kafka:
//	  PS_AUDIT_KAFKA_BROKERS: comma-separated broker list (host:port,...)
//	  PS_AUDIT_KAFKA_TOPIC:   topic name
//	Postgres:
//	  PS_AUDIT_PG_DSN:   connection string
//	  PS_AUDIT_PG_TABLE: table name (default: audit_events)
//	  PS_AUDIT_PG_AUTO_MIGRATE: when "1"/"true" (default), ensure table exists
//
// Returns the logger, a close function (no-op when not needed), and an error.
func NewLoggerFromEnv() (Logger, func() error, error) {
	sink := strings.ToLower(strings.TrimSpace(os.Getenv("PS_AUDIT_SINK")))
	switch sink {
	case "", "none":
		// Back-compat: allow file sink via PS_AUDIT_FILE even when sink not set
		if path := strings.TrimSpace(os.Getenv("PS_AUDIT_FILE")); path != "" {
			rl, err := NewDailyRotatingLogger(path)
			if err != nil {
				return nil, nil, err
			}
			return rl, rl.Close, nil
		}
		return nil, func() error { return nil }, nil
	case "file":
		path := strings.TrimSpace(os.Getenv("PS_AUDIT_FILE"))
		if path == "" {
			return nil, nil, errors.New("PS_AUDIT_FILE must be set for file sink")
		}
		rl, err := NewDailyRotatingLogger(path)
		if err != nil {
			return nil, nil, err
		}
		return rl, rl.Close, nil
	case "kafka":
		brokers := strings.TrimSpace(os.Getenv("PS_AUDIT_KAFKA_BROKERS"))
		topic := strings.TrimSpace(os.Getenv("PS_AUDIT_KAFKA_TOPIC"))
		if brokers == "" || topic == "" {
			return nil, nil, fmt.Errorf("PS_AUDIT_KAFKA_BROKERS and PS_AUDIT_KAFKA_TOPIC are required for kafka sink")
		}
		kl, err := NewKafkaLogger(strings.Split(brokers, ","), topic)
		if err != nil {
			return nil, nil, err
		}
		return kl, kl.Close, nil
	case "postgres":
		dsn := strings.TrimSpace(os.Getenv("PS_AUDIT_PG_DSN"))
		if dsn == "" {
			return nil, nil, errors.New("PS_AUDIT_PG_DSN must be set for postgres sink")
		}
		table := strings.TrimSpace(os.Getenv("PS_AUDIT_PG_TABLE"))
		if table == "" {
			table = "audit_events"
		}
		auto := strings.ToLower(strings.TrimSpace(os.Getenv("PS_AUDIT_PG_AUTO_MIGRATE")))
		autoMigrate := (auto == "" || auto == "1" || auto == "true" || auto == "yes")
		pl, err := NewPostgresLogger(dsn, table, autoMigrate)
		if err != nil {
			return nil, nil, err
		}
		return pl, pl.Close, nil
	default:
		return nil, nil, fmt.Errorf("unknown PS_AUDIT_SINK: %s", sink)
	}
}
