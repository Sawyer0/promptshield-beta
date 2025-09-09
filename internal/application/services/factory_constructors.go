package services

import (
	"context"
	"log/slog"

	"github.com/promptshield/promptshield/internal/audit"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/repository"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// RulepackServiceFromFactory creates a RulepackService using repositories from a factory
func RulepackServiceFromFactory(factory repository.RepositoryFactory, pub *nats.Publisher) *RulepackService {
	auditLogger, closeFunc, err := audit.NewLoggerFromEnv()
	if err != nil || auditLogger == nil {
		if err != nil {
			slog.Error("Failed to initialize audit logger", "error", err)
		} else {
			slog.Debug("No audit logger configured, using no-op logger")
		}
		// Create a no-op audit logger for degraded functionality
		auditLogger = &noOpAuditLogger{}
	}
	
	// Store close function for proper cleanup (in production, this would be managed by DI container)
	if closeFunc != nil {
		// In a production system, you would register this with a cleanup handler
		// For now, we'll just log that it exists
		slog.Debug("Audit logger initialized with cleanup function")
	}
	
	return &RulepackService{
		repo:  factory.Rulepack(),
		pub:   pub,
		audit: auditLogger,
	}
}

// PolicyServiceFromFactory creates a PolicyService using repositories from a factory
func PolicyServiceFromFactory(factory repository.RepositoryFactory) contracts.PolicyService {
	auditLogger, closeFunc, err := audit.NewLoggerFromEnv()
	var contractsAuditLogger contracts.AuditLogger
	
	if err != nil {
		slog.Error("Failed to initialize audit logger", "error", err)
		// Create a no-op audit logger for degraded functionality
		contractsAuditLogger = &noOpAuditLogger{}
	} else {
		// Create an adapter to convert audit.Logger to contracts.AuditLogger
		contractsAuditLogger = &auditLoggerAdapter{logger: auditLogger, closeFunc: closeFunc}
	}

	// Create policy repository (this would need to be added to the factory interface)
	// For now, we'll use a placeholder - this shows the pattern for when PolicyRepository is added
	var policyRepo contracts.PolicyRepository
	// policyRepo = factory.Policy() // This would be the factory method when implemented
	
	// Create other dependencies
	var validator contracts.RuleCompiler
	var scanEngine contracts.ScanEngine
	
	return NewPolicyService(policyRepo, validator, scanEngine, contractsAuditLogger)
}

// PolicyScannerServiceFromFactory creates a PolicyScannerService using repositories from a factory
func PolicyScannerServiceFromFactory(factory repository.RepositoryFactory) *PolicyScannerService {
	// Create policy repository (this would need to be added to the factory interface)
	// For now, we'll use a placeholder - this shows the pattern for when PolicyRepository is added
	var policyRepo contracts.PolicyRepository
	// policyRepo = factory.Policy() // This would be the factory method when implemented
	
	return NewPolicyScannerService(policyRepo)
}

// NewServicesFromFactory creates all application services using repositories from a factory
func NewServicesFromFactory(factory repository.RepositoryFactory, pub *nats.Publisher) *Services {
	return &Services{
		Rulepack:       RulepackServiceFromFactory(factory, pub),
		Policy:         PolicyServiceFromFactory(factory),
		PolicyScanner:  PolicyScannerServiceFromFactory(factory),
		// Add other services here as they are created
	}
}

// Services holds all application services
type Services struct {
	Rulepack       *RulepackService
	Policy         contracts.PolicyService
	PolicyScanner  *PolicyScannerService
	// Add other services here as they are created
}

// noOpAuditLogger provides a no-op implementation for graceful degradation
// It implements both audit.Logger and contracts.AuditLogger interfaces
type noOpAuditLogger struct{}

// Log implements audit.Logger interface
func (n *noOpAuditLogger) Log(event audit.Event) error {
	// No-op implementation - just log that we would have audited this
	slog.Debug("Audit event not logged due to initialization failure", 
		"event_type", event.Type, 
		"timestamp", event.Timestamp)
	return nil
}

// LogWithContext implements contracts.AuditLogger interface
func (n *noOpAuditLogger) LogWithContext(ctx context.Context, event types.AuditEvent) error {
	// No-op implementation - just log that we would have audited this
	slog.Debug("Audit event not logged due to initialization failure", 
		"action", event.Action, 
		"object_id", event.ObjectID,
		"timestamp", event.Timestamp)
	return nil
}

func (n *noOpAuditLogger) Flush() error {
	return nil
}

func (n *noOpAuditLogger) Close() error {
	return nil
}

// auditLoggerAdapter adapts audit.Logger to contracts.AuditLogger
type auditLoggerAdapter struct {
	logger    audit.Logger
	closeFunc func() error
}

func (a *auditLoggerAdapter) LogWithContext(ctx context.Context, event types.AuditEvent) error {
	// Convert types.AuditEvent to audit.Event
	auditEvent := audit.Event{
		Type: event.Action,
		Data: map[string]interface{}{
			"tenant_id":    event.TenantID,
			"actor_id":     event.ActorID,
			"actor_type":   event.ActorType,
			"actor_email":  event.ActorEmail,
			"object_type":  event.ObjectType,
			"object_id":    event.ObjectID,
			"request_id":   event.RequestID,
			"metadata":     event.Metadata,
			"before":       event.Before,
			"after":        event.After,
		},
		Timestamp: event.Timestamp,
	}
	
	return a.logger.Log(auditEvent)
}

func (a *auditLoggerAdapter) Flush() error {
	// audit.Logger doesn't have a Flush method, so this is a no-op
	return nil
}

func (a *auditLoggerAdapter) Close() error {
	if a.closeFunc != nil {
		return a.closeFunc()
	}
	return nil
}