package postgres

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// convertIntMapToInterface converts map[string]int to map[string]interface{}
func convertIntMapToInterface(m map[string]int) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// pgAuditReporter implements contracts.AuditReporter with PostgreSQL backend
type pgAuditReporter struct {
	eventStore contracts.AuditEventStore
}

// NewAuditReporter creates a new PostgreSQL audit reporter
func NewAuditReporter(eventStore contracts.AuditEventStore) contracts.AuditReporter {
	return &pgAuditReporter{
		eventStore: eventStore,
	}
}

// GenerateReport generates an audit report
func (r *pgAuditReporter) GenerateReport(ctx context.Context, config *types.AuditReportConfig) (*types.AuditReport, error) {
	// Get events based on config
	events, err := r.eventStore.Retrieve(ctx, config.Filters)
	if err != nil {
		return nil, fmt.Errorf("retrieve events for report: %w", err)
	}

	report := &types.AuditReport{
		ID:          uuid.New().String(),
		Type:        config.ReportType,
		Title:       r.generateReportTitle(config),
		Description: r.generateReportDescription(config),
		TimeRange:   config.TimeRange,
		GeneratedAt: time.Now(),
		GeneratedBy: "system", // TODO: Get from context
		TotalEvents: int64(len(events)),
		Summary:     make(map[string]interface{}),
		Details:     events,
		Format:      config.Format,
	}

	// Generate summary based on grouping
	if len(config.GroupBy) > 0 {
		report.Summary = r.generateGroupedSummary(events, config.GroupBy)
	} else {
		report.Summary = r.generateBasicSummary(events)
	}

	// Generate charts data if requested
	if config.IncludeDetails {
		report.Charts = r.generateCharts(events, config.GroupBy)
	}

	return report, nil
}

// GenerateComplianceReport generates a compliance-specific report
func (r *pgAuditReporter) GenerateComplianceReport(ctx context.Context, standard string, timeRange types.TimeRange) (*types.ComplianceReport, error) {
	// Get all events in time range
	filter := &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	}

	events, err := r.eventStore.Retrieve(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("retrieve events for compliance report: %w", err)
	}

	report := &types.ComplianceReport{
		ID:           uuid.New().String(),
		Standard:     standard,
		TimeRange:    timeRange,
		GeneratedAt:  time.Now(),
		GeneratedBy:  "system",
		Requirements: r.getComplianceRequirements(standard),
		Evidence:     make(map[string]interface{}),
	}

	// Analyze compliance based on standard
	violations, status := r.analyzeCompliance(standard, events)

	report.ComplianceStatus = status
	report.Violations = violations
	report.Recommendations = r.generateComplianceRecommendations(standard, violations)

	return report, nil
}

// GetActivitySummary returns activity summary for a time period
func (r *pgAuditReporter) GetActivitySummary(ctx context.Context, timeRange types.TimeRange) (*types.ActivitySummary, error) {
	filter := &types.AuditFilter{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	}

	events, err := r.eventStore.Retrieve(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("retrieve events for activity summary: %w", err)
	}

	summary := &types.ActivitySummary{
		TimeRange:    timeRange,
		TotalEvents:  int64(len(events)),
		EventTypes:   make(map[string]int64),
		TopUsers:     make([]*types.UserActivity, 0),
		TopResources: make([]*types.ResourceActivity, 0),
		TopActions:   make([]*types.ActionActivity, 0),
		RiskMetrics:  &types.RiskMetrics{},
	}

	// Analyze events
	userActivity := make(map[string]*types.UserActivity)
	resourceActivity := make(map[string]*types.ResourceActivity)
	actionActivity := make(map[string]*types.ActionActivity)

	for _, event := range events {
		// Count event types
		summary.EventTypes[event.Action]++

		// Track user activity
		userID := "unknown"
		if event.ActorID != nil {
			userID = event.ActorID.String()
		}
		if userActivity[userID] == nil {
			userActivity[userID] = &types.UserActivity{
				UserID:       userID,
				UserEmail:    event.ActorEmail,
				EventCount:   0,
				LastActivity: event.Timestamp,
			}
		}
		userActivity[userID].EventCount++
		if event.Timestamp.After(userActivity[userID].LastActivity) {
			userActivity[userID].LastActivity = event.Timestamp
		}

		// Track resource activity
		resourceID := event.ObjectID.String()
		if resourceActivity[resourceID] == nil {
			resourceActivity[resourceID] = &types.ResourceActivity{
				ResourceID:   resourceID,
				ResourceType: event.ObjectType,
				EventCount:   0,
				LastActivity: event.Timestamp,
			}
		}
		resourceActivity[resourceID].EventCount++
		if event.Timestamp.After(resourceActivity[resourceID].LastActivity) {
			resourceActivity[resourceID].LastActivity = event.Timestamp
		}

		// Track action activity
		if actionActivity[event.Action] == nil {
			actionActivity[event.Action] = &types.ActionActivity{
				Action:       event.Action,
				EventCount:   0,
				LastActivity: event.Timestamp,
			}
		}
		actionActivity[event.Action].EventCount++
		if event.Timestamp.After(actionActivity[event.Action].LastActivity) {
			actionActivity[event.Action].LastActivity = event.Timestamp
		}
	}

	// Convert to slices and sort
	for _, activity := range userActivity {
		summary.TopUsers = append(summary.TopUsers, activity)
	}
	for _, activity := range resourceActivity {
		summary.TopResources = append(summary.TopResources, activity)
	}
	for _, activity := range actionActivity {
		summary.TopActions = append(summary.TopActions, activity)
	}

	// Calculate risk metrics
	summary.RiskMetrics = r.calculateRiskMetrics(events)

	return summary, nil
}

// GetUserActivity returns activity for a specific user
func (r *pgAuditReporter) GetUserActivity(ctx context.Context, userID string, timeRange types.TimeRange) ([]*types.AuditEvent, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	filter := &types.AuditFilter{
		ActorID:   userUUID.String(),
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	}

	return r.eventStore.Retrieve(ctx, filter)
}

// GetResourceActivity returns activity for a specific resource
func (r *pgAuditReporter) GetResourceActivity(ctx context.Context, resourceID string, timeRange types.TimeRange) ([]*types.AuditEvent, error) {
	resourceUUID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid resource ID: %w", err)
	}

	filter := &types.AuditFilter{
		ObjectID:  &resourceUUID,
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
	}

	return r.eventStore.Retrieve(ctx, filter)
}

// ExportReport exports a report in specified format
func (r *pgAuditReporter) ExportReport(ctx context.Context, reportID string, format string) ([]byte, error) {
	// For now, generate a simple report and export it
	// In a real implementation, you'd store reports and retrieve by ID
	config := &types.AuditReportConfig{
		ReportType: "activity",
		TimeRange: types.TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
		Format: format,
	}

	report, err := r.GenerateReport(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("generate report for export: %w", err)
	}

	return r.exportReportInFormat(report, format)
}

// ScheduleReport schedules periodic report generation
func (r *pgAuditReporter) ScheduleReport(ctx context.Context, config *types.AuditReportSchedule) error {
	// This would typically store the schedule in a database
	// For now, we'll just validate the configuration
	if config.ReportType == "" {
		return fmt.Errorf("report type is required")
	}
	if config.Schedule == "" {
		return fmt.Errorf("schedule is required")
	}
	if config.Config == nil {
		return fmt.Errorf("report configuration is required")
	}

	// TODO: Implement actual scheduling mechanism
	return fmt.Errorf("report scheduling not yet implemented")
}

// GetReportHistory returns previously generated reports
func (r *pgAuditReporter) GetReportHistory(ctx context.Context, filter *types.ReportFilter) ([]*types.AuditReport, error) {
	// TODO: Implement report storage and retrieval
	// For now, return empty list
	return []*types.AuditReport{}, nil
}

// Helper functions

func (r *pgAuditReporter) generateReportTitle(config *types.AuditReportConfig) string {
	switch config.ReportType {
	case "activity":
		return "User Activity Report"
	case "security":
		return "Security Events Report"
	case "compliance":
		return "Compliance Report"
	case "performance":
		return "Performance Report"
	default:
		return "Audit Report"
	}
}

func (r *pgAuditReporter) generateReportDescription(config *types.AuditReportConfig) string {
	timeRange := fmt.Sprintf("from %s to %s",
		config.TimeRange.Start.Format("2006-01-02"),
		config.TimeRange.End.Format("2006-01-02"))

	switch config.ReportType {
	case "activity":
		return fmt.Sprintf("Summary of user activities %s", timeRange)
	case "security":
		return fmt.Sprintf("Security-related events and violations %s", timeRange)
	case "compliance":
		return fmt.Sprintf("Compliance status and requirements %s", timeRange)
	case "performance":
		return fmt.Sprintf("System performance metrics %s", timeRange)
	default:
		return fmt.Sprintf("Audit events %s", timeRange)
	}
}

func (r *pgAuditReporter) generateBasicSummary(events []*types.AuditEvent) map[string]interface{} {
	summary := make(map[string]interface{})

	// Basic statistics
	summary["total_events"] = len(events)
	summary["unique_users"] = r.countUniqueUsers(events)
	summary["unique_resources"] = r.countUniqueResources(events)
	summary["time_span_hours"] = r.calculateTimeSpan(events)

	// Event type breakdown
	eventTypes := make(map[string]int)
	for _, event := range events {
		eventTypes[event.Action]++
	}
	summary["event_types"] = eventTypes

	return summary
}

func (r *pgAuditReporter) generateGroupedSummary(events []*types.AuditEvent, groupBy []string) map[string]interface{} {
	// For now, implement basic grouping by first field
	if len(groupBy) == 0 {
		return r.generateBasicSummary(events)
	}

	grouped := make(map[string]interface{})

	switch groupBy[0] {
	case "action":
		grouped = r.groupByAction(events)
	case "user":
		grouped = r.groupByUser(events)
	case "resource":
		grouped = r.groupByResource(events)
	default:
		grouped = r.generateBasicSummary(events)
	}

	return grouped
}

func (r *pgAuditReporter) generateCharts(events []*types.AuditEvent, _ []string) []*types.ReportChart {
	charts := make([]*types.ReportChart, 0)

	// Event types pie chart
	eventTypeData := make(map[string]int)
	for _, event := range events {
		eventTypeData[event.Action]++
	}

	charts = append(charts, &types.ReportChart{
		Title: "Events by Type",
		Type:  "pie",
		Data:  convertIntMapToInterface(eventTypeData),
	})

	// Timeline chart (simplified)
	timelineData := r.generateTimelineData(events)
	charts = append(charts, &types.ReportChart{
		Title: "Events Over Time",
		Type:  "line",
		Data:  timelineData,
	})

	return charts
}

func (r *pgAuditReporter) getComplianceRequirements(standard string) []*types.ComplianceRequirement {
	requirements := make([]*types.ComplianceRequirement, 0)

	switch standard {
	case "SOC2":
		requirements = append(requirements, &types.ComplianceRequirement{
			ID:          "SOC2-1",
			Description: "Access controls are properly implemented",
			Category:    "Security",
		})
		requirements = append(requirements, &types.ComplianceRequirement{
			ID:          "SOC2-2",
			Description: "Audit logs are maintained and protected",
			Category:    "Security",
		})
	case "HIPAA":
		requirements = append(requirements, &types.ComplianceRequirement{
			ID:          "HIPAA-1",
			Description: "PHI data is properly protected",
			Category:    "Privacy",
		})
	case "GDPR":
		requirements = append(requirements, &types.ComplianceRequirement{
			ID:          "GDPR-1",
			Description: "Data processing has lawful basis",
			Category:    "Privacy",
		})
	}

	return requirements
}

func (r *pgAuditReporter) analyzeCompliance(standard string, events []*types.AuditEvent) ([]*types.ComplianceViolation, string) {
	violations := make([]*types.ComplianceViolation, 0)

	switch standard {
	case "SOC2":
		for _, event := range events {
			if event.Action == "violation.detected" {
				violations = append(violations, &types.ComplianceViolation{
					RequirementID: "SOC2-1",
					Description:   "Access control violation detected",
					Severity:      "HIGH",
					Event:         event,
				})
			}
		}
	case "HIPAA":
		// Check for PHI-related violations
		for _, event := range events {
			if strings.Contains(event.Action, "data") && event.ObjectType == "medical_record" {
				violations = append(violations, &types.ComplianceViolation{
					RequirementID: "HIPAA-1",
					Description:   "PHI data access without proper authorization",
					Severity:      "CRITICAL",
					Event:         event,
				})
			}
		}
	}

	status := "compliant"
	if len(violations) > 0 {
		status = "non-compliant"
	}

	return violations, status
}

func (r *pgAuditReporter) generateComplianceRecommendations(standard string, violations []*types.ComplianceViolation) []string {
	recommendations := make([]string, 0)

	if len(violations) > 0 {
		recommendations = append(recommendations, "Review and strengthen access controls")
		recommendations = append(recommendations, "Implement additional monitoring and alerting")
		recommendations = append(recommendations, "Conduct regular compliance audits")
	}

	switch standard {
	case "SOC2":
		recommendations = append(recommendations, "Ensure all access is logged and monitored")
	case "HIPAA":
		recommendations = append(recommendations, "Implement proper PHI data handling procedures")
	}

	return recommendations
}

func (r *pgAuditReporter) calculateRiskMetrics(events []*types.AuditEvent) *types.RiskMetrics {
	metrics := &types.RiskMetrics{
		HighRiskEvents:   0,
		MediumRiskEvents: 0,
		LowRiskEvents:    0,
		TotalRiskScore:   0,
	}

	for _, event := range events {
		switch event.Action {
		case "violation.detected", "request.blocked":
			metrics.HighRiskEvents++
			metrics.TotalRiskScore += 10
		case "policy.delete", "provider_key.revoke":
			metrics.MediumRiskEvents++
			metrics.TotalRiskScore += 5
		default:
			metrics.LowRiskEvents++
			metrics.TotalRiskScore += 1
		}
	}

	return metrics
}

func (r *pgAuditReporter) exportReportInFormat(report *types.AuditReport, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(report, "", "  ")
	case "csv":
		return r.reportToCSV(report)
	default:
		return json.Marshal(report)
	}
}

// Utility functions
func (r *pgAuditReporter) countUniqueUsers(events []*types.AuditEvent) int {
	users := make(map[string]bool)
	for _, event := range events {
		userID := "unknown"
		if event.ActorID != nil {
			userID = event.ActorID.String()
		}
		users[userID] = true
	}
	return len(users)
}

func (r *pgAuditReporter) countUniqueResources(events []*types.AuditEvent) int {
	resources := make(map[string]bool)
	for _, event := range events {
		resources[event.ObjectID.String()] = true
	}
	return len(resources)
}

func (r *pgAuditReporter) calculateTimeSpan(events []*types.AuditEvent) float64 {
	if len(events) < 2 {
		return 0
	}

	earliest := events[0].Timestamp
	latest := events[0].Timestamp

	for _, event := range events[1:] {
		if event.Timestamp.Before(earliest) {
			earliest = event.Timestamp
		}
		if event.Timestamp.After(latest) {
			latest = event.Timestamp
		}
	}

	return latest.Sub(earliest).Hours()
}

func (r *pgAuditReporter) groupByAction(events []*types.AuditEvent) map[string]interface{} {
	grouped := make(map[string]interface{})

	for _, event := range events {
		action := event.Action
		if grouped[action] == nil {
			grouped[action] = make([]*types.AuditEvent, 0)
		}
		grouped[action] = append(grouped[action].([]*types.AuditEvent), event)
	}

	return grouped
}

func (r *pgAuditReporter) groupByUser(events []*types.AuditEvent) map[string]interface{} {
	grouped := make(map[string]interface{})

	for _, event := range events {
		userID := "unknown"
		if event.ActorID != nil {
			userID = event.ActorID.String()
		}

		if grouped[userID] == nil {
			grouped[userID] = make([]*types.AuditEvent, 0)
		}
		grouped[userID] = append(grouped[userID].([]*types.AuditEvent), event)
	}

	return grouped
}

func (r *pgAuditReporter) groupByResource(events []*types.AuditEvent) map[string]interface{} {
	grouped := make(map[string]interface{})

	for _, event := range events {
		resourceID := event.ObjectID.String()

		if grouped[resourceID] == nil {
			grouped[resourceID] = make([]*types.AuditEvent, 0)
		}
		grouped[resourceID] = append(grouped[resourceID].([]*types.AuditEvent), event)
	}

	return grouped
}

func (r *pgAuditReporter) generateTimelineData(events []*types.AuditEvent) map[string]interface{} {
	// Group events by hour
	hourly := make(map[string]int)

	for _, event := range events {
		hour := event.Timestamp.Format("2006-01-02 15:04")
		hourly[hour]++
	}

	return convertIntMapToInterface(hourly)
}

func (r *pgAuditReporter) reportToCSV(report *types.AuditReport) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	writer.Write([]string{"timestamp", "action", "actor_type", "actor_email", "object_type", "object_id"})

	// Write events
	for _, event := range report.Details {
		writer.Write([]string{
			event.Timestamp.Format(time.RFC3339),
			event.Action,
			string(event.ActorType),
			event.ActorEmail,
			event.ObjectType,
			event.ObjectID.String(),
		})
	}

	writer.Flush()
	return []byte(buf.String()), writer.Error()
}



