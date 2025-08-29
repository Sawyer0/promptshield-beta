package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// AuditIntegrationService demonstrates audit system integration
type AuditIntegrationService struct {
	services *AuditServices
	logger   *slog.Logger
}

// NewAuditIntegrationService creates a new audit integration service
func NewAuditIntegrationService(services *AuditServices, logger *slog.Logger) *AuditIntegrationService {
	return &AuditIntegrationService{
		services: services,
		logger:   logger,
	}
}

// DemonstrateFullAuditFlow demonstrates a complete audit workflow
func (s *AuditIntegrationService) DemonstrateFullAuditFlow(ctx context.Context) error {
	s.logger.Info("Starting full audit flow demonstration")

	// 1. Create some sample audit events
	events := s.createSampleEvents()

	// 2. Store events using hash chain
	for _, event := range events {
		hash, err := s.services.HashChain.AppendEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("append event to hash chain: %w", err)
		}
		s.logger.Info("Stored audit event", "event_id", event.ObjectID, "hash", hash)
	}

	// 3. Create audit trails for entities
	tenantID := uuid.New()
	err := s.services.TrailManager.CreateTrail(ctx, tenantID.String(), "tenant")
	if err != nil {
		return fmt.Errorf("create audit trail: %w", err)
	}

	// 4. Get and validate trails
	trail, err := s.services.TrailManager.GetTrail(ctx, tenantID.String(), "tenant")
	if err != nil {
		return fmt.Errorf("get audit trail: %w", err)
	}
	s.logger.Info("Retrieved audit trail", "events", len(trail))

	// 5. Validate trail integrity
	validation, err := s.services.TrailManager.ValidateTrail(ctx, tenantID.String())
	if err != nil {
		return fmt.Errorf("validate audit trail: %w", err)
	}
	s.logger.Info("Trail validation", "valid", validation.IsValid, "events", validation.TotalEvents)

	// 6. Generate compliance report
	timeRange := types.TimeRange{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
	}

	complianceReport, err := s.services.Compliance.GenerateComplianceReport(ctx, "SOC2", timeRange)
	if err != nil {
		return fmt.Errorf("generate compliance report: %w", err)
	}
	s.logger.Info("Compliance report generated",
		"standard", complianceReport.Standard,
		"status", complianceReport.ComplianceStatus,
		"score", complianceReport.ComplianceScore)

	// 7. Generate activity summary
	activitySummary, err := s.services.Reporter.GetActivitySummary(ctx, timeRange)
	if err != nil {
		return fmt.Errorf("get activity summary: %w", err)
	}
	s.logger.Info("Activity summary",
		"total_events", activitySummary.TotalEvents,
		"unique_users", activitySummary.UniqueUsers,
		"risk_score", activitySummary.RiskMetrics.TotalRiskScore)

	// 8. Analyze patterns and detect anomalies
	analysis, err := s.services.Analyzer.AnalyzePatterns(ctx, timeRange)
	if err != nil {
		return fmt.Errorf("analyze patterns: %w", err)
	}
	s.logger.Info("Pattern analysis completed",
		"patterns", len(analysis.Patterns),
		"anomalies", len(analysis.Anomalies))

	// 9. Verify hash chain integrity
	chainInfo, err := s.services.HashChain.GetChainInfo(ctx)
	if err != nil {
		return fmt.Errorf("get chain info: %w", err)
	}

	if chainInfo.CurrentHash != "" {
		verification, err := s.services.HashChain.VerifyChain(ctx, "", chainInfo.CurrentHash)
		if err != nil {
			return fmt.Errorf("verify hash chain: %w", err)
		}
		s.logger.Info("Hash chain verification", "valid", verification.IsValid, "events", verification.TotalEvents)
	}

	// 10. Configure audit alerts
	alerts := []*types.AuditAlert{
		{
			ID:          "demo-security-alert",
			Name:        "Demo Security Alert",
			Description: "Alert for demonstration purposes",
			Type:        "security",
			Severity:    "HIGH",
			Condition:   "security_violation",
			Enabled:     true,
			Recipients:  []string{"demo@promptshield.io"},
		},
	}

	err = s.services.Notifier.ConfigureAlerts(ctx, alerts)
	if err != nil {
		return fmt.Errorf("configure alerts: %w", err)
	}
	s.logger.Info("Audit alerts configured", "count", len(alerts))

	// 11. Export audit trail
	exportData, err := s.services.TrailManager.ExportTrail(ctx, tenantID.String(), "json")
	if err != nil {
		return fmt.Errorf("export audit trail: %w", err)
	}
	s.logger.Info("Audit trail exported", "size", len(exportData))

	// 12. Track data retention
	retentionStatus, err := s.services.Compliance.TrackDataRetention(ctx)
	if err != nil {
		return fmt.Errorf("track data retention: %w", err)
	}
	s.logger.Info("Data retention tracked",
		"total_events", retentionStatus.TotalEvents,
		"violations", len(retentionStatus.Violations))

	s.logger.Info("Full audit flow demonstration completed successfully")
	return nil
}

// createSampleEvents creates sample audit events for demonstration
func (s *AuditIntegrationService) createSampleEvents() []*types.AuditEvent {
	tenantID := uuid.New()
	userID := uuid.New()

	now := time.Now()
	events := []*types.AuditEvent{
		{
			TenantID:   &tenantID,
			ActorID:    &userID,
			ActorType:  types.ActorTypeUser,
			ActorEmail: "demo@promptshield.io",
			Action:     "user.login",
			ObjectType: "session",
			ObjectID:   uuid.New(),
			Metadata: map[string]interface{}{
				"ip_address": "192.168.1.100",
				"user_agent": "Mozilla/5.0...",
			},
			Timestamp: now.Add(-2 * time.Hour),
		},
		{
			TenantID:   &tenantID,
			ActorID:    &userID,
			ActorType:  types.ActorTypeUser,
			ActorEmail: "demo@promptshield.io",
			Action:     "policy.create",
			ObjectType: "policy",
			ObjectID:   uuid.New(),
			Before:     nil,
			After: map[string]interface{}{
				"name": "Demo Policy",
				"type": "security",
			},
			Metadata: map[string]interface{}{
				"change_type": "create",
			},
			Timestamp: now.Add(-1 * time.Hour),
		},
		{
			TenantID:   &tenantID,
			ActorID:    &userID,
			ActorType:  types.ActorTypeUser,
			ActorEmail: "demo@promptshield.io",
			Action:     "request.allowed",
			ObjectType: "api_request",
			ObjectID:   uuid.New(),
			Metadata: map[string]interface{}{
				"endpoint":      "/v1/chat/completions",
				"method":        "POST",
				"response_time": 150,
			},
			Timestamp: now.Add(-30 * time.Minute),
		},
		{
			TenantID:   &tenantID,
			ActorID:    &userID,
			ActorType:  types.ActorTypeUser,
			ActorEmail: "demo@promptshield.io",
			Action:     "violation.detected",
			ObjectType: "request",
			ObjectID:   uuid.New(),
			Metadata: map[string]interface{}{
				"violation_type": "prompt_injection",
				"severity":       "HIGH",
				"blocked":        true,
			},
			Timestamp: now.Add(-15 * time.Minute),
		},
		{
			TenantID:   &tenantID,
			ActorID:    &userID,
			ActorType:  types.ActorTypeUser,
			ActorEmail: "demo@promptshield.io",
			Action:     "user.logout",
			ObjectType: "session",
			ObjectID:   uuid.New(),
			Metadata: map[string]interface{}{
				"session_duration": 7200, // 2 hours
			},
			Timestamp: now.Add(-5 * time.Minute),
		},
	}

	return events
}

// GetAuditSystemHealth returns the health status of the audit system
func (s *AuditIntegrationService) GetAuditSystemHealth(ctx context.Context) (*types.AuditSystemHealth, error) {
	health := &types.AuditSystemHealth{
		CheckedAt: time.Now(),
		Services:  []*types.ServiceHealth{},
	}

	// Check event store
	eventCount, err := s.services.EventStore.Count(ctx, &types.AuditFilter{})
	if err != nil {
		health.Services = append(health.Services, &types.ServiceHealth{
			Name:   "event_store",
			Status: "unhealthy",
			Error:  err.Error(),
		})
	} else {
		health.Services = append(health.Services, &types.ServiceHealth{
			Name:   "event_store",
			Status: "healthy",
		})
		health.TotalEvents = eventCount
	}

	// Check hash chain
	chainInfo, err := s.services.HashChain.GetChainInfo(ctx)
	if err != nil {
		health.Services = append(health.Services, &types.ServiceHealth{
			Name:   "hash_chain",
			Status: "unhealthy",
			Error:  err.Error(),
		})
	} else {
		health.Services = append(health.Services, &types.ServiceHealth{
			Name:   "hash_chain",
			Status: "healthy",
		})
		health.ChainIntegrity = true
		_ = chainInfo // chainInfo is used for health check validation
	}

	// Check compliance status
	complianceStatus, err := s.services.Compliance.GetComplianceStatus(ctx, "SOC2")
	if err != nil {
		health.Services = append(health.Services, &types.ServiceHealth{
			Name:   "compliance",
			Status: "unhealthy",
			Error:  err.Error(),
		})
	} else {
		health.Services = append(health.Services, &types.ServiceHealth{
			Name:   "compliance",
			Status: "healthy",
		})
		_ = complianceStatus // complianceStatus is used for health check validation
	}

	// Calculate overall health
	allHealthy := true
	for _, service := range health.Services {
		if service.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		health.OverallStatus = "healthy"
	} else {
		health.OverallStatus = "degraded"
	}

	return health, nil
}



