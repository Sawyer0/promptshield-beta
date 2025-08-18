package audit

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Service provides centralized audit logging functionality (instance-based, no global state)
type Service struct {
	logger contracts.AuditLogger
	mu     sync.RWMutex
}

// NewService creates a new audit service
func NewService(logger contracts.AuditLogger) *Service {
	return &Service{
		logger: logger,
	}
}

// LogEvent logs an audit event with context enrichment (legacy method)
func (s *Service) LogEvent(ctx context.Context, action string, objectType string, objectID string, metadata map[string]interface{}) error {
	// This method is kept for compatibility but will be deprecated
	// Try to parse objectID as UUID, use zero UUID if invalid
	var objUUID uuid.UUID
	if parsed, err := uuid.Parse(objectID); err == nil {
		objUUID = parsed
	}
	
	event := &types.AuditEvent{
		Action:     action,
		ObjectType: objectType,
		ObjectID:   objUUID,
		Metadata:   metadata,
		Timestamp:  time.Now().UTC(),
	}

	// Enrich with context information
	enrichEventFromContext(ctx, event)

	s.mu.RLock()
	logger := s.logger
	s.mu.RUnlock()

	if logger != nil {
		return logger.LogWithContext(ctx, *event)
	}
	return nil
}

// LogAuditEvent logs a pre-constructed audit event with context enrichment
func (s *Service) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
	// Enrich with context information
	enrichEventFromContext(ctx, event)

	s.mu.RLock()
	logger := s.logger
	s.mu.RUnlock()

	if logger != nil {
		return logger.LogWithContext(ctx, *event)
	}
	return nil
}

// SetLogger updates the audit logger
func (s *Service) SetLogger(logger contracts.AuditLogger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = logger
}

// GetLogger returns the current audit logger
func (s *Service) GetLogger() contracts.AuditLogger {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logger
}

// Flush flushes any pending audit logs
func (s *Service) Flush() error {
	s.mu.RLock()
	logger := s.logger
	s.mu.RUnlock()

	if logger != nil {
		return logger.Flush()
	}
	return nil
}

// Close closes the audit service
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.logger != nil {
		return s.logger.Close()
	}
	return nil
}