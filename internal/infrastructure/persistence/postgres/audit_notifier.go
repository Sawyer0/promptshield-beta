package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// pgAuditNotifier implements contracts.AuditNotifier with PostgreSQL backend
type pgAuditNotifier struct {
	eventStore contracts.AuditEventStore
	logger     *slog.Logger
	alerts     []*types.AuditAlert
}

// NewAuditNotifier creates a new PostgreSQL audit notifier
func NewAuditNotifier(eventStore contracts.AuditEventStore, logger *slog.Logger) contracts.AuditNotifier {
	return &pgAuditNotifier{
		eventStore: eventStore,
		logger:     logger,
		alerts:     make([]*types.AuditAlert, 0),
	}
}

// NotifySecurityEvent sends notification for security events
func (n *pgAuditNotifier) NotifySecurityEvent(ctx context.Context, event *types.SecurityEvent) error {
	if event == nil {
		return fmt.Errorf("security event cannot be nil")
	}

	// Log the security event
	n.logger.Warn("Security event detected",
		"type", event.Type,
		"severity", event.Severity,
		"user_id", event.UserID,
		"resource", event.Resource,
		"ip_address", event.IPAddress,
		"timestamp", event.Timestamp,
	)

	// Create audit event for the notification
	var tenantUUID, actorUUID, objectUUID *uuid.UUID

	if event.TenantID != "" {
		if parsedUUID, parseErr := uuid.Parse(event.TenantID); parseErr == nil {
			tenantUUID = &parsedUUID
		}
	}
	if event.UserID != "" {
		if parsedUUID, parseErr := uuid.Parse(event.UserID); parseErr == nil {
			actorUUID = &parsedUUID
		}
	}
	if parsedUUID, parseErr := uuid.Parse(event.ID); parseErr == nil {
		objectUUID = &parsedUUID
	} else {
		// If parsing fails, generate a new UUID
		newUUID := uuid.New()
		objectUUID = &newUUID
	}

	auditEvent := &types.AuditEvent{
		TenantID:   tenantUUID,
		ActorID:    actorUUID,
		ActorType:  types.ActorTypeUser,
		Action:     "security.notification.sent",
		ObjectType: "security_event",
		ObjectID:   *objectUUID,
		Metadata: map[string]interface{}{
			"event_type": event.Type,
			"severity":   event.Severity,
			"resource":   event.Resource,
			"ip_address": event.IPAddress,
			"user_agent": event.UserAgent,
		},
		Timestamp: time.Now(),
	}

	return n.eventStore.Store(ctx, auditEvent)
}

// NotifyComplianceViolation sends notification for compliance violations
func (n *pgAuditNotifier) NotifyComplianceViolation(ctx context.Context, violation *types.ComplianceViolation) error {
	if violation == nil {
		return fmt.Errorf("compliance violation cannot be nil")
	}

	// Log the compliance violation
	n.logger.Error("Compliance violation detected",
		"requirement_id", violation.RequirementID,
		"description", violation.Description,
		"severity", violation.Severity,
		"violation_id", violation.ID,
	)

	// Create audit event for the notification
	var objectUUID uuid.UUID
	if parsedUUID, parseErr := uuid.Parse(violation.ID); parseErr == nil {
		objectUUID = parsedUUID
	} else {
		// If parsing fails, generate a new UUID
		objectUUID = uuid.New()
	}

	auditEvent := &types.AuditEvent{
		ActorType:  types.ActorTypeSystem,
		ActorEmail: "system@promptshield.io",
		Action:     "compliance.violation.notification.sent",
		ObjectType: "compliance_violation",
		ObjectID:   objectUUID,
		Metadata: map[string]interface{}{
			"requirement_id": violation.RequirementID,
			"description":    violation.Description,
			"severity":       violation.Severity,
			"standard":       violation.Standard,
			"mitigation":     violation.Mitigation,
		},
		Timestamp: time.Now(),
	}

	return n.eventStore.Store(ctx, auditEvent)
}

// NotifyAnomalousActivity sends notification for anomalous activities
func (n *pgAuditNotifier) NotifyAnomalousActivity(ctx context.Context, anomaly *types.AuditAnomaly) error {
	if anomaly == nil {
		return fmt.Errorf("audit anomaly cannot be nil")
	}

	// Log the anomalous activity
	n.logger.Warn("Anomalous activity detected",
		"type", anomaly.Type,
		"severity", anomaly.Severity,
		"description", anomaly.Description,
		"user_id", anomaly.UserID,
		"resource", anomaly.Resource,
		"confidence", anomaly.Confidence,
	)

	// Create audit event for the notification
	var anomalyActorUUID, anomalyObjectUUID *uuid.UUID
	if anomaly.UserID != "" {
		if parsedUUID, parseErr := uuid.Parse(anomaly.UserID); parseErr == nil {
			anomalyActorUUID = &parsedUUID
		}
	}
	if parsedUUID, parseErr := uuid.Parse(anomaly.ID); parseErr == nil {
		anomalyObjectUUID = &parsedUUID
	} else {
		anomalyUUID := uuid.New()
		anomalyObjectUUID = &anomalyUUID
	}

	auditEvent := &types.AuditEvent{
		ActorID:    anomalyActorUUID,
		ActorType:  types.ActorTypeUser,
		Action:     "anomaly.notification.sent",
		ObjectType: "audit_anomaly",
		ObjectID:   *anomalyObjectUUID,
		Metadata: map[string]interface{}{
			"anomaly_type": anomaly.Type,
			"severity":     anomaly.Severity,
			"description":  anomaly.Description,
			"resource":     anomaly.Resource,
			"confidence":   anomaly.Confidence,
			"baseline":     anomaly.Baseline,
			"deviation":    anomaly.Deviation,
		},
		Timestamp: time.Now(),
	}

	return n.eventStore.Store(ctx, auditEvent)
}

// ConfigureAlerts configures audit-based alerts
func (n *pgAuditNotifier) ConfigureAlerts(ctx context.Context, alerts []*types.AuditAlert) error {
	if len(alerts) == 0 {
		return fmt.Errorf("alerts cannot be empty")
	}

	// Validate alerts
	for _, alert := range alerts {
		if err := n.validateAlert(alert); err != nil {
			return fmt.Errorf("invalid alert %s: %w", alert.ID, err)
		}
	}

	n.alerts = alerts

	// Log alert configuration
	n.logger.Info("Audit alerts configured", "count", len(alerts))

	// Create audit event for alert configuration
	alertsData, _ := json.Marshal(alerts)
	auditEvent := &types.AuditEvent{
		ActorType:  types.ActorTypeSystem,
		ActorEmail: "system@promptshield.io",
		Action:     "audit.alerts.configured",
		ObjectType: "audit_configuration",
		ObjectID:   uuid.New(),
		Metadata: map[string]interface{}{
			"alert_count": len(alerts),
			"alerts_data": string(alertsData),
		},
		Timestamp: time.Now(),
	}

	return n.eventStore.Store(ctx, auditEvent)
}

// TestAlert tests alert configuration
func (n *pgAuditNotifier) TestAlert(ctx context.Context, alertID string) error {
	alert := n.findAlert(alertID)
	if alert == nil {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	// Create a test notification
	testEvent := &types.AuditEvent{
		ActorType:  types.ActorTypeSystem,
		ActorEmail: "system@promptshield.io",
		Action:     "audit.alert.test",
		ObjectType: "alert_test",
		ObjectID:   uuid.New(),
		Metadata: map[string]interface{}{
			"alert_id":   alertID,
			"alert_name": alert.Name,
			"test_time":  time.Now(),
		},
		Timestamp: time.Now(),
	}

	n.logger.Info("Testing audit alert", "alert_id", alertID, "alert_name", alert.Name)

	return n.eventStore.Store(ctx, testEvent)
}

// GetAlertHistory returns alert notification history
func (n *pgAuditNotifier) GetAlertHistory(ctx context.Context, filter *types.AlertFilter) ([]*types.AlertNotification, error) {
	if filter == nil {
		filter = &types.AlertFilter{}
	}

	// Build query for alert-related events
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Filter by alert-related actions
	conditions = append(conditions, "action IN ('security.notification.sent', 'compliance.violation.notification.sent', 'anomaly.notification.sent', 'audit.alert.test')")

	if !filter.TimeRange.Start.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, filter.TimeRange.Start)
		argIdx++
	}

	if !filter.TimeRange.End.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, filter.TimeRange.End)
		argIdx++
	}

	if filter.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("metadata->>'severity' = $%d", argIdx))
		args = append(args, filter.Severity)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")
	query := fmt.Sprintf("SELECT * FROM audits WHERE %s ORDER BY created_at DESC", whereClause)

	// Execute the query with proper arguments
	rows, err := n.eventStore.(*pgAuditEventStore).db.Raw().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying alert history: %w", err)
	}
	defer rows.Close()

	var notifications []*types.AlertNotification
	for rows.Next() {
		var auditEvent types.AuditEvent
		if err := rows.Scan(&auditEvent); err != nil {
			n.logger.Warn("Failed to scan audit event", "error", err)
			continue
		}

		// Convert audit event to notification
		notification := &types.AlertNotification{
			ID:       auditEvent.ObjectID.String(),
			AlertID:  auditEvent.Action, // Use action as alert ID
			Type:     auditEvent.ObjectType,
			Severity: auditEvent.Metadata["severity"].(string),
			Message:  auditEvent.Action,
			SentAt:   auditEvent.Timestamp,
			Status:   "sent",
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating alert history: %w", err)
	}

	return notifications, nil
}

// SetNotificationThresholds sets thresholds for audit notifications
func (n *pgAuditNotifier) SetNotificationThresholds(ctx context.Context, thresholds map[string]interface{}) error {
	if len(thresholds) == 0 {
		return fmt.Errorf("thresholds cannot be empty")
	}

	// Validate thresholds
	for key, value := range thresholds {
		switch key {
		case "max_notifications_per_hour", "max_notifications_per_day":
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("threshold %s must be a number", key)
			}
		case "min_severity":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("threshold %s must be a string", key)
			}
		default:
			n.logger.Warn("Unknown threshold", "key", key)
		}
	}

	// Store threshold configuration
	thresholdsData, _ := json.Marshal(thresholds)
	auditEvent := &types.AuditEvent{
		ActorType:  types.ActorTypeSystem,
		ActorEmail: "system@promptshield.io",
		Action:     "audit.thresholds.configured",
		ObjectType: "audit_configuration",
		ObjectID:   uuid.New(),
		Metadata: map[string]interface{}{
			"thresholds": string(thresholdsData),
		},
		Timestamp: time.Now(),
	}

	n.logger.Info("Notification thresholds configured", "thresholds", thresholds)

	return n.eventStore.Store(ctx, auditEvent)
}

// Helper functions

func (n *pgAuditNotifier) validateAlert(alert *types.AuditAlert) error {
	if alert.ID == "" {
		return fmt.Errorf("alert ID is required")
	}
	if alert.Name == "" {
		return fmt.Errorf("alert name is required")
	}
	if alert.Condition == "" {
		return fmt.Errorf("alert condition is required")
	}
	if alert.Severity == "" {
		return fmt.Errorf("alert severity is required")
	}

	// Validate severity levels
	validSeverities := []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}
	severityValid := false
	for _, s := range validSeverities {
		if alert.Severity == s {
			severityValid = true
			break
		}
	}
	if !severityValid {
		return fmt.Errorf("invalid severity level: %s", alert.Severity)
	}

	return nil
}

func (n *pgAuditNotifier) findAlert(alertID string) *types.AuditAlert {
	for _, alert := range n.alerts {
		if alert.ID == alertID {
			return alert
		}
	}
	return nil
}
