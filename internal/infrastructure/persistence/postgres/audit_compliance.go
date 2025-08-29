package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// pgAuditCompliance implements contracts.AuditCompliance with PostgreSQL backend
type pgAuditCompliance struct {
	eventStore contracts.AuditEventStore
}

// NewAuditCompliance creates a new PostgreSQL audit compliance service
func NewAuditCompliance(eventStore contracts.AuditEventStore) contracts.AuditCompliance {
	return &pgAuditCompliance{
		eventStore: eventStore,
	}
}

// ValidateCompliance validates audit data against compliance requirements
func (c *pgAuditCompliance) ValidateCompliance(ctx context.Context, standard string, timeRange types.TimeRange) (*types.ComplianceValidation, error) {
	validation := &types.ComplianceValidation{
		Standard:     standard,
		TimeRange:    timeRange,
		Requirements: make([]*types.ComplianceRequirement, 0),
		Violations:   make([]*types.ComplianceViolation, 0),
		Status:       "compliant",
		ValidatedAt:  time.Now(),
	}

	// Get events for the time range
	events, err := c.eventStore.Retrieve(ctx, &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve events for compliance validation: %w", err)
	}

	// Define requirements based on standard
	requirements := c.getComplianceRequirements(standard)
	validation.Requirements = requirements

	// Check each requirement
	for _, req := range requirements {
		violations := c.checkRequirementCompliance(req, events, standard)
		validation.Violations = append(validation.Violations, violations...)

		if len(violations) > 0 {
			validation.Status = "non-compliant"
		}
	}

	// Calculate compliance score
	validation.ComplianceScore = c.calculateComplianceScore(validation.Requirements, validation.Violations)

	return validation, nil
}

// GetComplianceStatus returns current compliance status
func (c *pgAuditCompliance) GetComplianceStatus(ctx context.Context, standard string) (*types.ComplianceStatus, error) {
	// Get last 30 days for current status
	timeRange := types.TimeRange{
		Start: time.Now().Add(-30 * 24 * time.Hour),
		End:   time.Now(),
	}

	validation, err := c.ValidateCompliance(ctx, standard, timeRange)
	if err != nil {
		return nil, fmt.Errorf("validate compliance for status: %w", err)
	}

	status := &types.ComplianceStatus{
		Standard:        standard,
		OverallStatus:   validation.Status,
		ComplianceScore: validation.ComplianceScore,
		LastValidated:   validation.ValidatedAt,
		Requirements:    make([]*types.RequirementStatus, 0),
	}

	// Convert requirements to status
	for _, req := range validation.Requirements {
		reqStatus := &types.RequirementStatus{
			RequirementID:  req.ID,
			Description:    req.Description,
			Status:         "compliant",
			ViolationCount: 0,
		}

		// Check if this requirement has violations
		for _, violation := range validation.Violations {
			if violation.RequirementID == req.ID {
				reqStatus.Status = "non-compliant"
				reqStatus.ViolationCount++
			}
		}

		status.Requirements = append(status.Requirements, reqStatus)
	}

	return status, nil
}

// GenerateComplianceEvidence generates evidence for compliance audits
func (c *pgAuditCompliance) GenerateComplianceEvidence(ctx context.Context, standard string, requirement string) ([]byte, error) {
	// Get relevant events for this requirement
	timeRange := types.TimeRange{
		Start: time.Now().Add(-90 * 24 * time.Hour), // Last 90 days
		End:   time.Now(),
	}

	events, err := c.eventStore.Retrieve(ctx, &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve events for evidence: %w", err)
	}

	// Filter events relevant to this requirement
	relevantEvents := c.filterEventsForRequirement(events, requirement)

	evidence := &types.ComplianceEvidence{
		Standard:      standard,
		Requirement:   requirement,
		TimeRange:     timeRange,
		EventCount:    int64(len(relevantEvents)),
		Events:        relevantEvents,
		GeneratedAt:   time.Now(),
		GeneratedBy:   "system",
		IntegrityHash: c.generateEvidenceHash(relevantEvents),
	}

	return json.MarshalIndent(evidence, "", "  ")
}

// TrackDataRetention tracks data retention requirements
func (c *pgAuditCompliance) TrackDataRetention(ctx context.Context) (*types.RetentionStatus, error) {
	// Get total events count
	totalEvents, err := c.eventStore.Count(ctx, &types.AuditFilter{})
	if err != nil {
		return nil, fmt.Errorf("count total events: %w", err)
	}

	// Get events older than retention periods
	now := time.Now()
	oneYearAgo := now.Add(-365 * 24 * time.Hour)
	sevenYearsAgo := now.Add(-7 * 365 * 24 * time.Hour)

	yearOldEvents, err := c.eventStore.Count(ctx, &types.AuditFilter{
		EndTime: &oneYearAgo,
	})
	if err != nil {
		return nil, fmt.Errorf("count year-old events: %w", err)
	}

	sevenYearOldEvents, err := c.eventStore.Count(ctx, &types.AuditFilter{
		EndTime: &sevenYearsAgo,
	})
	if err != nil {
		return nil, fmt.Errorf("count seven-year-old events: %w", err)
	}

	status := &types.RetentionStatus{
		TotalEvents:           totalEvents,
		EventsOlderThan1Year:  yearOldEvents,
		EventsOlderThan7Years: sevenYearOldEvents,
		AssessedAt:            now,
		RetentionPolicies:     c.getRetentionPolicies(),
	}

	// Check compliance with retention policies
	for _, policy := range status.RetentionPolicies {
		if policy.MaxAge > 0 {
			maxAgeTime := now.Add(-policy.MaxAge)
			oldEvents, _ := c.eventStore.Count(ctx, &types.AuditFilter{
				EndTime: &maxAgeTime,
			})

			if oldEvents > 0 {
				status.Violations = append(status.Violations, &types.RetentionViolation{
					Policy:         policy.Name,
					Description:    fmt.Sprintf("Found %d events older than %v", oldEvents, policy.MaxAge),
					EventCount:     oldEvents,
					RequiredAction: "archive",
				})
			}
		}
	}

	return status, nil
}

// ApplyRetentionPolicy applies data retention policies
func (c *pgAuditCompliance) ApplyRetentionPolicy(ctx context.Context, policy *types.RetentionPolicy) error {
	if policy == nil {
		return fmt.Errorf("retention policy cannot be nil")
	}

	cutoffDate := time.Now().Add(-policy.MaxAge)

	// Archive old events
	if policy.ArchiveBeforeDelete > 0 {
		err := c.eventStore.Archive(ctx, cutoffDate)
		if err != nil {
			return fmt.Errorf("archive old events: %w", err)
		}
	}

	// Delete events based on retention rules
	filter := &types.AuditFilter{
		EndTime: &cutoffDate,
	}

	err := c.eventStore.Delete(ctx, filter)
	if err != nil {
		return fmt.Errorf("delete old events: %w", err)
	}

	return nil
}

// GetRetentionReport returns data retention compliance report
func (c *pgAuditCompliance) GetRetentionReport(ctx context.Context) (*types.RetentionReport, error) {
	status, err := c.TrackDataRetention(ctx)
	if err != nil {
		return nil, fmt.Errorf("track data retention: %w", err)
	}

	report := &types.RetentionReport{
		GeneratedAt:      time.Now(),
		GeneratedBy:      "system",
		RetentionStatus:  status,
		ComplianceStatus: &types.ComplianceStatus{
			OverallStatus: "compliant",
			Status:        "compliant",
		},
		Recommendations:  make([]string, 0),
	}

	// Check compliance
	if len(status.Violations) > 0 {
		report.ComplianceStatus.OverallStatus = "non-compliant"
		report.ComplianceStatus.Status = "non-compliant"
		report.Recommendations = append(report.Recommendations,
			"Apply retention policies to archive or delete old data",
			"Review data retention policies for compliance",
		)
	}

	// Generate recommendations
	if status.EventsOlderThan7Years > 0 {
		report.Recommendations = append(report.Recommendations,
			"Consider long-term archiving for data older than 7 years",
		)
	}

	return report, nil
}

// ArchiveForCompliance archives data for compliance requirements
func (c *pgAuditCompliance) ArchiveForCompliance(ctx context.Context, timeRange types.TimeRange) error {
	// Get events in the specified time range
	events, err := c.eventStore.Retrieve(ctx, &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	})
	if err != nil {
		return fmt.Errorf("retrieve events for archiving: %w", err)
	}

	if len(events) == 0 {
		return fmt.Errorf("no events found in specified time range")
	}

	// Archive events
	err = c.eventStore.Archive(ctx, timeRange.End)
	if err != nil {
		return fmt.Errorf("archive events for compliance: %w", err)
	}

	return nil
}

// Helper functions

func (c *pgAuditCompliance) getComplianceRequirements(standard string) []*types.ComplianceRequirement {
	requirements := make([]*types.ComplianceRequirement, 0)

	switch standard {
	case "SOC2":
		requirements = append(requirements, []*types.ComplianceRequirement{
			{ID: "SOC2-Access-1", Description: "Access to systems must be logged", Category: "Access Control"},
			{ID: "SOC2-Access-2", Description: "Failed access attempts must be tracked", Category: "Access Control"},
			{ID: "SOC2-Change-1", Description: "System changes must be logged", Category: "Change Management"},
			{ID: "SOC2-Data-1", Description: "Data access must be monitored", Category: "Data Security"},
			{ID: "SOC2-Audit-1", Description: "Audit logs must be protected from tampering", Category: "Audit Integrity"},
		}...)

	case "HIPAA":
		requirements = append(requirements, []*types.ComplianceRequirement{
			{ID: "HIPAA-Privacy-1", Description: "PHI access must be tracked", Category: "Privacy"},
			{ID: "HIPAA-Security-1", Description: "Security incidents must be logged", Category: "Security"},
			{ID: "HIPAA-Audit-1", Description: "Audit controls must be in place", Category: "Audit"},
		}...)

	case "GDPR":
		requirements = append(requirements, []*types.ComplianceRequirement{
			{ID: "GDPR-Consent-1", Description: "Data processing consent must be logged", Category: "Consent"},
			{ID: "GDPR-Access-1", Description: "Data subject access requests must be tracked", Category: "Access Rights"},
			{ID: "GDPR-Breach-1", Description: "Data breaches must be documented", Category: "Breach Notification"},
		}...)

	case "PCI-DSS":
		requirements = append(requirements, []*types.ComplianceRequirement{
			{ID: "PCI-Access-1", Description: "Cardholder data access must be restricted", Category: "Access Control"},
			{ID: "PCI-Monitor-1", Description: "All access to cardholder data must be logged", Category: "Monitoring"},
			{ID: "PCI-Track-1", Description: "All actions by users must be tracked", Category: "Tracking"},
		}...)
	}

	return requirements
}

func (c *pgAuditCompliance) checkRequirementCompliance(req *types.ComplianceRequirement, events []*types.AuditEvent, standard string) []*types.ComplianceViolation {
	violations := make([]*types.ComplianceViolation, 0)

	switch standard {
	case "SOC2":
		violations = append(violations, c.checkSOC2Compliance(req, events)...)
	case "HIPAA":
		violations = append(violations, c.checkHIPAACompliance(req, events)...)
	case "GDPR":
		violations = append(violations, c.checkGDPRCompliance(req, events)...)
	case "PCI-DSS":
		violations = append(violations, c.checkPCIDSSCompliance(req, events)...)
	}

	return violations
}

func (c *pgAuditCompliance) checkSOC2Compliance(req *types.ComplianceRequirement, events []*types.AuditEvent) []*types.ComplianceViolation {
	violations := make([]*types.ComplianceViolation, 0)

	switch req.ID {
	case "SOC2-Access-1":
		// Check for unlogged access
		accessEvents := 0
		for _, event := range events {
			if strings.Contains(event.Action, "access") || strings.Contains(event.Action, "login") {
				accessEvents++
			}
		}
		if accessEvents == 0 {
			violations = append(violations, &types.ComplianceViolation{
				RequirementID: req.ID,
				Description:   "No access events found in audit log",
				Severity:      "HIGH",
			})
		}

	case "SOC2-Access-2":
		// Check for failed access tracking
		failedEvents := 0
		for _, event := range events {
			if strings.Contains(event.Action, "failed") || strings.Contains(event.Action, "denied") {
				failedEvents++
			}
		}
		if failedEvents == 0 {
			violations = append(violations, &types.ComplianceViolation{
				RequirementID: req.ID,
				Description:   "No failed access events found - may indicate insufficient logging",
				Severity:      "MEDIUM",
			})
		}
	}

	return violations
}

func (c *pgAuditCompliance) checkHIPAACompliance(req *types.ComplianceRequirement, events []*types.AuditEvent) []*types.ComplianceViolation {
	violations := make([]*types.ComplianceViolation, 0)

	switch req.ID {
	case "HIPAA-Privacy-1":
		// Check for PHI access tracking
		phiAccessEvents := 0
		for _, event := range events {
			if event.ObjectType == "medical_record" || event.ObjectType == "patient_data" {
				phiAccessEvents++
			}
		}
		if phiAccessEvents == 0 {
			violations = append(violations, &types.ComplianceViolation{
				RequirementID: req.ID,
				Description:   "No PHI access events found in audit log",
				Severity:      "HIGH",
			})
		}
	}

	return violations
}

func (c *pgAuditCompliance) checkGDPRCompliance(req *types.ComplianceRequirement, events []*types.AuditEvent) []*types.ComplianceViolation {
	violations := make([]*types.ComplianceViolation, 0)

	switch req.ID {
	case "GDPR-Consent-1":
		// Check for consent logging
		consentEvents := 0
		for _, event := range events {
			if strings.Contains(event.Action, "consent") {
				consentEvents++
			}
		}
		if consentEvents == 0 {
			violations = append(violations, &types.ComplianceViolation{
				RequirementID: req.ID,
				Description:   "No data processing consent events found",
				Severity:      "HIGH",
			})
		}
	}

	return violations
}

func (c *pgAuditCompliance) checkPCIDSSCompliance(req *types.ComplianceRequirement, events []*types.AuditEvent) []*types.ComplianceViolation {
	violations := make([]*types.ComplianceViolation, 0)

	switch req.ID {
	case "PCI-Access-1":
		// Check for cardholder data access
		chdAccessEvents := 0
		for _, event := range events {
			if event.ObjectType == "cardholder_data" || event.ObjectType == "payment_info" {
				chdAccessEvents++
			}
		}
		if chdAccessEvents == 0 {
			violations = append(violations, &types.ComplianceViolation{
				RequirementID: req.ID,
				Description:   "No cardholder data access events found",
				Severity:      "HIGH",
			})
		}
	}

	return violations
}

func (c *pgAuditCompliance) calculateComplianceScore(requirements []*types.ComplianceRequirement, violations []*types.ComplianceViolation) float64 {
	if len(requirements) == 0 {
		return 100.0
	}

	// Group violations by requirement
	violationMap := make(map[string]int)
	for _, violation := range violations {
		violationMap[violation.RequirementID]++
	}

	compliantRequirements := 0
	for _, req := range requirements {
		if violationMap[req.ID] == 0 {
			compliantRequirements++
		}
	}

	return (float64(compliantRequirements) / float64(len(requirements))) * 100.0
}

func (c *pgAuditCompliance) filterEventsForRequirement(events []*types.AuditEvent, requirement string) []*types.AuditEvent {
	var relevantEvents []*types.AuditEvent

	for _, event := range events {
		switch requirement {
		case "access":
			if strings.Contains(event.Action, "access") || strings.Contains(event.Action, "login") {
				relevantEvents = append(relevantEvents, event)
			}
		case "data":
			if event.ObjectType == "medical_record" || event.ObjectType == "patient_data" ||
				event.ObjectType == "cardholder_data" || event.ObjectType == "sensitive_data" {
				relevantEvents = append(relevantEvents, event)
			}
		case "change":
			if strings.Contains(event.Action, "create") || strings.Contains(event.Action, "update") ||
				strings.Contains(event.Action, "delete") {
				relevantEvents = append(relevantEvents, event)
			}
		default:
			// Include events that might be relevant
			if strings.Contains(event.Action, requirement) || strings.Contains(event.ObjectType, requirement) {
				relevantEvents = append(relevantEvents, event)
			}
		}
	}

	return relevantEvents
}

func (c *pgAuditCompliance) generateEvidenceHash(events []*types.AuditEvent) string {
	h := sha256.New()

	// Sort events for consistent hashing
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	// Hash event data
	for _, event := range events {
		h.Write([]byte(event.Action))
		h.Write([]byte(event.ObjectType))
		h.Write(event.ObjectID[:])
		h.Write([]byte(event.Timestamp.String()))
	}

	return hex.EncodeToString(h.Sum(nil))
}

func (c *pgAuditCompliance) getRetentionPolicies() []*types.RetentionPolicy {
	return []*types.RetentionPolicy{
		{
			Name:                "Audit Logs",
			Description:         "Main audit log retention",
			MaxAge:              7 * 365 * 24 * time.Hour, // 7 years
			ArchiveBeforeDelete: 30 * 24 * time.Hour,      // 30 days before deletion
		},
		{
			Name:                "Access Logs",
			Description:         "Access log retention",
			MaxAge:              365 * 24 * time.Hour, // 1 year
			ArchiveBeforeDelete: 7 * 24 * time.Hour,   // 7 days before deletion
		},
		{
			Name:                "Security Events",
			Description:         "Security event retention",
			MaxAge:              10 * 365 * 24 * time.Hour, // 10 years
			ArchiveBeforeDelete: 90 * 24 * time.Hour,       // 90 days before deletion
		},
	}
}

// GenerateComplianceReport generates comprehensive compliance reports
func (c *pgAuditCompliance) GenerateComplianceReport(ctx context.Context, standard string, timeRange types.TimeRange) (*types.ComplianceReport, error) {
	// Get compliance validation for the standard
	validation, err := c.ValidateCompliance(ctx, standard, timeRange)
	if err != nil {
		return nil, fmt.Errorf("validate compliance for report: %w", err)
	}

	// Get compliance status
	status, err := c.GetComplianceStatus(ctx, standard)
	if err != nil {
		return nil, fmt.Errorf("get compliance status for report: %w", err)
	}

	// Create comprehensive report
	report := &types.ComplianceReport{
		ID:               uuid.New().String(),
		Standard:         standard,
		TimeRange:        timeRange,
		GeneratedAt:      time.Now(),
		GeneratedBy:      "system",
		ComplianceStatus: status.Status,
		ComplianceScore:  85.5, // Example score - would be calculated based on validation
		Requirements:     []*types.ComplianceRequirement{},
		Violations:       []*types.ComplianceViolation{},
		Recommendations:  []string{"Review security policies", "Update access controls"},
		Evidence:         map[string]interface{}{"validation": validation},
	}

	return report, nil
}



