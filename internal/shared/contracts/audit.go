package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// AuditEventStore defines the interface for storing and retrieving audit events
type AuditEventStore interface {
	// Store stores a single audit event
	Store(ctx context.Context, event *types.AuditEvent) error

	// StoreBatch stores multiple audit events
	StoreBatch(ctx context.Context, events []*types.AuditEvent) error

	// Retrieve retrieves audit events by filter
	Retrieve(ctx context.Context, filter *types.AuditFilter) ([]*types.AuditEvent, error)

	// GetByID retrieves an audit event by ID
	GetByID(ctx context.Context, eventID string) (*types.AuditEvent, error)

	// Count returns the count of audit events matching filter
	Count(ctx context.Context, filter *types.AuditFilter) (int64, error)

	// Delete deletes audit events matching filter (for compliance retention)
	Delete(ctx context.Context, filter *types.AuditFilter) error

	// Archive archives old audit events
	Archive(ctx context.Context, olderThan time.Time) error

	// Verify verifies audit log integrity
	Verify(ctx context.Context, timeRange types.TimeRange) (*types.AuditVerification, error)
}

// AuditTrailManager defines the interface for managing audit trails
type AuditTrailManager interface {
	// CreateTrail creates a new audit trail for an entity
	CreateTrail(ctx context.Context, entityID string, entityType string) error

	// GetTrail retrieves the audit trail for an entity
	GetTrail(ctx context.Context, entityID string, entityType string) ([]*types.AuditEvent, error)

	// AppendToTrail appends an event to an audit trail
	AppendToTrail(ctx context.Context, entityID string, event *types.AuditEvent) error

	// GetTrailSummary returns a summary of the audit trail
	GetTrailSummary(ctx context.Context, entityID string) (*types.AuditTrailSummary, error)

	// ExportTrail exports an audit trail
	ExportTrail(ctx context.Context, entityID string, format string) ([]byte, error)

	// ValidateTrail validates audit trail integrity
	ValidateTrail(ctx context.Context, entityID string) (*types.TrailValidation, error)

	// MergeTrails merges multiple audit trails
	MergeTrails(ctx context.Context, entityIDs []string) ([]*types.AuditEvent, error)
}

// AuditReporter defines the interface for audit reporting
type AuditReporter interface {
	// GenerateReport generates an audit report
	GenerateReport(ctx context.Context, config *types.AuditReportConfig) (*types.AuditReport, error)

	// GenerateComplianceReport generates a compliance-specific report
	GenerateComplianceReport(ctx context.Context, standard string, timeRange types.TimeRange) (*types.ComplianceReport, error)

	// GetActivitySummary returns activity summary for a time period
	GetActivitySummary(ctx context.Context, timeRange types.TimeRange) (*types.ActivitySummary, error)

	// GetUserActivity returns activity for a specific user
	GetUserActivity(ctx context.Context, userID string, timeRange types.TimeRange) ([]*types.AuditEvent, error)

	// GetResourceActivity returns activity for a specific resource
	GetResourceActivity(ctx context.Context, resourceID string, timeRange types.TimeRange) ([]*types.AuditEvent, error)

	// ExportReport exports a report in specified format
	ExportReport(ctx context.Context, reportID string, format string) ([]byte, error)

	// ScheduleReport schedules periodic report generation
	ScheduleReport(ctx context.Context, config *types.AuditReportSchedule) error

	// GetReportHistory returns previously generated reports
	GetReportHistory(ctx context.Context, filter *types.ReportFilter) ([]*types.AuditReport, error)
}

// AuditAnalyzer defines the interface for audit data analysis
type AuditAnalyzer interface {
	// AnalyzePatterns analyzes patterns in audit data
	AnalyzePatterns(ctx context.Context, timeRange types.TimeRange) (*types.AuditPatternAnalysis, error)

	// DetectAnomalies detects anomalous activities in audit data
	DetectAnomalies(ctx context.Context, timeRange types.TimeRange) ([]*types.AuditAnomaly, error)

	// GetTrendAnalysis returns trend analysis for audit events
	GetTrendAnalysis(ctx context.Context, timeRange types.TimeRange, granularity time.Duration) (*types.AuditTrendAnalysis, error)

	// GetRiskAssessment returns risk assessment based on audit data
	GetRiskAssessment(ctx context.Context, entityID string) (*types.RiskAssessment, error)

	// CorrelateEvents correlates related audit events
	CorrelateEvents(ctx context.Context, events []*types.AuditEvent) ([]*types.EventCorrelation, error)

	// GetBehaviorBaseline establishes behavioral baseline from audit data
	GetBehaviorBaseline(ctx context.Context, userID string, timeRange types.TimeRange) (*types.BehaviorBaseline, error)

	// CompareToBaseline compares current activity to established baseline
	CompareToBaseline(ctx context.Context, userID string, activity []*types.AuditEvent) (*types.BaselineComparison, error)
}

// AuditCompliance defines the interface for compliance-related audit operations
type AuditCompliance interface {
	// ValidateCompliance validates audit data against compliance requirements
	ValidateCompliance(ctx context.Context, standard string, timeRange types.TimeRange) (*types.ComplianceValidation, error)

	// GetComplianceStatus returns current compliance status
	GetComplianceStatus(ctx context.Context, standard string) (*types.ComplianceStatus, error)

	// GenerateComplianceEvidence generates evidence for compliance audits
	GenerateComplianceEvidence(ctx context.Context, standard string, requirement string) ([]byte, error)

	// TrackDataRetention tracks data retention requirements
	TrackDataRetention(ctx context.Context) (*types.RetentionStatus, error)

	// ApplyRetentionPolicy applies data retention policies
	ApplyRetentionPolicy(ctx context.Context, policy *types.RetentionPolicy) error

	// GetRetentionReport returns data retention compliance report
	GetRetentionReport(ctx context.Context) (*types.RetentionReport, error)

	// ArchiveForCompliance archives data for compliance requirements
	ArchiveForCompliance(ctx context.Context, timeRange types.TimeRange) error
}

// AuditNotifier defines the interface for audit-based notifications
type AuditNotifier interface {
	// NotifySecurityEvent sends notification for security events
	NotifySecurityEvent(ctx context.Context, event *types.SecurityEvent) error

	// NotifyComplianceViolation sends notification for compliance violations
	NotifyComplianceViolation(ctx context.Context, violation *types.ComplianceViolation) error

	// NotifyAnomalousActivity sends notification for anomalous activities
	NotifyAnomalousActivity(ctx context.Context, anomaly *types.AuditAnomaly) error

	// ConfigureAlerts configures audit-based alerts
	ConfigureAlerts(ctx context.Context, alerts []*types.AuditAlert) error

	// TestAlert tests alert configuration
	TestAlert(ctx context.Context, alertID string) error

	// GetAlertHistory returns alert notification history
	GetAlertHistory(ctx context.Context, filter *types.AlertFilter) ([]*types.AlertNotification, error)

	// SetNotificationThresholds sets thresholds for audit notifications
	SetNotificationThresholds(ctx context.Context, thresholds map[string]interface{}) error
}

// AuditHashChain defines the interface for audit log integrity using hash chains
type AuditHashChain interface {
	// AppendEvent appends an event to the hash chain
	AppendEvent(ctx context.Context, event *types.AuditEvent) (string, error)

	// VerifyChain verifies the integrity of the hash chain
	VerifyChain(ctx context.Context, startHash string, endHash string) (*types.ChainVerification, error)

	// GetChainInfo returns information about the hash chain
	GetChainInfo(ctx context.Context) (*types.ChainInfo, error)

	// RepairChain attempts to repair a broken hash chain
	RepairChain(ctx context.Context, fromEventID string) error

	// ExportChain exports the hash chain for verification
	ExportChain(ctx context.Context, timeRange types.TimeRange) ([]byte, error)

	// ValidateEvent validates an event against the hash chain
	ValidateEvent(ctx context.Context, eventID string) (*types.EventValidation, error)
}
