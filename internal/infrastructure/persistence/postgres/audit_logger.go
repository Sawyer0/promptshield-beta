package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// pgHashChainAuditLogger implements contracts.AuditLogger with mandatory hash chaining
type pgHashChainAuditLogger struct {
	hashChain contracts.AuditHashChain
	logger    *slog.Logger
}

// NewAuditLogger creates a new audit logger with mandatory hash chaining
func NewAuditLogger(hashChain contracts.AuditHashChain) contracts.AuditLogger {
	return &pgHashChainAuditLogger{
		hashChain: hashChain,
		logger:    slog.With("component", "audit-logger"),
	}
}

// LogEvent logs an audit event with mandatory hash chaining
func (l *pgHashChainAuditLogger) LogEvent(ctx context.Context, event *types.AuditEvent) error {
	if event == nil {
		l.logger.Error("Audit event cannot be nil - hash chaining is mandatory")
		return errors.New("audit event cannot be nil - hash chaining is mandatory")
	}

	// Ensure timestamp is set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Generate ID if not set
	if event.ObjectID == uuid.Nil {
		event.ObjectID = uuid.New()
	}

	// MANDATORY: All events MUST go through hash chain
	hash, err := l.hashChain.AppendEvent(ctx, event)
	if err != nil {
		l.logger.Error("Failed to append event to hash chain", "error", err, "action", event.Action)
		return err
	}

	l.logger.Debug("Audit event logged with hash chain",
		"action", event.Action,
		"object_type", event.ObjectType,
		"object_id", event.ObjectID,
		"hash", hash,
		"prev_hash", event.PrevHash,
		"tenant_id", event.TenantID,
		"actor_id", event.ActorID,
	)

	return nil
}

// LogEventSync logs an audit event synchronously (same as LogEvent since hash chaining is already sync)
func (l *pgHashChainAuditLogger) LogEventSync(ctx context.Context, event *types.AuditEvent) error {
	return l.LogEvent(ctx, event)
}

// LogEventBatch logs multiple audit events with hash chaining
func (l *pgHashChainAuditLogger) LogEventBatch(ctx context.Context, events []*types.AuditEvent) error {
	if len(events) == 0 {
		l.logger.Error("Audit event batch cannot be empty - hash chaining is mandatory")
		return errors.New("audit event batch cannot be empty - hash chaining is mandatory")
	}

	// Log each event through the hash chain to maintain integrity
	for _, event := range events {
		if err := l.LogEvent(ctx, event); err != nil {
			l.logger.Error("Failed to log event in batch", "error", err, "action", event.Action)
			return err
		}
	}

	l.logger.Info("Batch audit events logged", "count", len(events))
	return nil
}

// LogWithContext logs an audit event with context (required by AuditLogger interface)
func (l *pgHashChainAuditLogger) LogWithContext(ctx context.Context, event types.AuditEvent) error {
	return l.LogEvent(ctx, &event)
}

// Flush flushes any pending audit logs (required by AuditLogger interface)
func (l *pgHashChainAuditLogger) Flush() error {
	l.logger.Debug("Audit logger flush requested - hash chain events are immediately persisted")
	return nil
}

// Close gracefully closes the audit logger
func (l *pgHashChainAuditLogger) Close() error {
	l.logger.Info("Audit logger closed")
	return nil
}

// GetStats returns audit logger statistics
func (l *pgHashChainAuditLogger) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"type":        "hash_chain_audit_logger",
		"hash_chain":  true,
		"mandatory":   true,
		"initialized": true,
	}
}

// Health returns the health status of the audit logger
func (l *pgHashChainAuditLogger) Health() error {
	// Check if hash chain is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := l.hashChain.GetChainInfo(ctx)
	if err != nil {
		l.logger.Error("Audit logger health check failed", "error", err)
		return err
	}

	return nil
}

// Ensure the implementation satisfies the interface
var _ contracts.AuditLogger = (*pgHashChainAuditLogger)(nil)
