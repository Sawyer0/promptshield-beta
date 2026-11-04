package audit

import (
	"context"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/requestctx"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// enrichEventFromContext adds context information to the audit event
func enrichEventFromContext(ctx context.Context, event *types.AuditEvent) {
	if requestID, ok := requestctx.GetRequestID(ctx); ok {
		event.RequestID = requestID
	}
	if tenantID, ok := requestctx.GetTenantID(ctx); ok {
		if parsed, err := uuid.Parse(tenantID); err == nil {
			event.TenantID = &parsed
		}
	}
	if userID, ok := requestctx.GetUserID(ctx); ok {
		if parsed, err := uuid.Parse(userID); err == nil {
			event.ActorID = &parsed
		}
	}
	if correlationID, ok := requestctx.GetCorrelationID(ctx); ok {
		if event.Metadata == nil {
			event.Metadata = make(map[string]interface{})
		}
		event.Metadata["correlation_id"] = correlationID
	}
}

// EnrichWithSecurityContext adds security-specific context to audit events
func EnrichWithSecurityContext(ctx context.Context, event *types.AuditEvent, ipAddress, userAgent string) {
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}

	if ipAddress != "" {
		event.Metadata["ip_address"] = ipAddress
	}
	if userAgent != "" {
		event.Metadata["user_agent"] = userAgent
	}

	// Add any additional security context from the request context
	enrichEventFromContext(ctx, event)
}

// EnrichWithPolicyContext adds policy enforcement context to audit events
func EnrichWithPolicyContext(event *types.AuditEvent, policyID, ruleID string, severity types.ViolationSeverity) {
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}

	if policyID != "" {
		event.Metadata["policy_id"] = policyID
	}
	if ruleID != "" {
		event.Metadata["rule_id"] = ruleID
	}
	if severity != "" {
		event.Metadata["severity"] = string(severity)
	}
}
