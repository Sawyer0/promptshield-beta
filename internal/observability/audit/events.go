package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/events"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// CreateSecurityEvent creates a security-related audit event
func CreateSecurityEvent(action, objectType string, objectID uuid.UUID, metadata map[string]interface{}) *types.AuditEvent {
	return &types.AuditEvent{
		Action:     action,
		ObjectType: objectType,
		ObjectID:   objectID,
		Metadata:   metadata,
		Timestamp:  time.Now().UTC(),
	}
}

// CreateUserEvent creates a user-related audit event
func CreateUserEvent(action string, userID uuid.UUID, metadata map[string]interface{}) *types.AuditEvent {
	return CreateSecurityEvent(action, events.ObjectTypeUser, userID, metadata)
}

// CreatePolicyEvent creates a policy-related audit event
func CreatePolicyEvent(action string, policyID uuid.UUID, metadata map[string]interface{}) *types.AuditEvent {
	return CreateSecurityEvent(action, events.ObjectTypePolicy, policyID, metadata)
}

// CreateEnforcementEvent creates an enforcement-related audit event
func CreateEnforcementEvent(action string, requestID uuid.UUID, decision types.EnforcementDecision, metadata map[string]interface{}) *types.AuditEvent {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["enforcement_decision"] = map[string]interface{}{
		"allow":        decision.Allow,
		"reason":       decision.Reason,
		"violations":   decision.Violations,
		"processed_at": decision.ProcessedAt,
		"latency_ms":   decision.Latency.Milliseconds(),
	}
	
	return CreateSecurityEvent(action, events.ObjectTypeRequest, requestID, metadata)
}

// LogSecurityEvent logs a security event using the provided audit service
func LogSecurityEvent(ctx context.Context, service *Service, action, objectType string, objectID uuid.UUID, metadata map[string]interface{}) error {
	event := CreateSecurityEvent(action, objectType, objectID, metadata)
	return service.LogAuditEvent(ctx, event)
}

// LogUserEvent logs a user event using the provided audit service
func LogUserEvent(ctx context.Context, service *Service, action string, userID uuid.UUID, metadata map[string]interface{}) error {
	return LogSecurityEvent(ctx, service, action, events.ObjectTypeUser, userID, metadata)
}

// LogPolicyEvent logs a policy event using the provided audit service
func LogPolicyEvent(ctx context.Context, service *Service, action string, policyID uuid.UUID, metadata map[string]interface{}) error {
	return LogSecurityEvent(ctx, service, action, events.ObjectTypePolicy, policyID, metadata)
}

// LogEnforcementEvent logs an enforcement event using the provided audit service
func LogEnforcementEvent(ctx context.Context, service *Service, action string, requestID uuid.UUID, decision types.EnforcementDecision, metadata map[string]interface{}) error {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["enforcement_decision"] = map[string]interface{}{
		"allow":        decision.Allow,
		"reason":       decision.Reason,
		"violations":   decision.Violations,
		"processed_at": decision.ProcessedAt,
		"latency_ms":   decision.Latency.Milliseconds(),
	}
	
	return LogSecurityEvent(ctx, service, action, events.ObjectTypeRequest, requestID, metadata)
}