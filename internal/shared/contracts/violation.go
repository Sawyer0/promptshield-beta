package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// ViolationDetector defines the interface for detecting policy violations
type ViolationDetector interface {
	// Detect detects violations in content
	Detect(ctx context.Context, content []byte) ([]*types.PolicyViolation, error)

	// DetectStream detects violations in streaming content
	DetectStream(ctx context.Context, content <-chan []byte) (<-chan *types.PolicyViolation, error)

	// DetectWithRules detects violations using specific rules
	DetectWithRules(ctx context.Context, content []byte, ruleIDs []string) ([]*types.PolicyViolation, error)

	// SetSeverityThreshold sets minimum severity for detection
	SetSeverityThreshold(severity types.ViolationSeverity) error

	// GetSupportedRules returns list of supported rule types
	GetSupportedRules() []string

	// ValidateRules validates rule configuration
	ValidateRules(ctx context.Context, rules []interface{}) error
}

// ViolationLogger defines the interface for logging policy violations
type ViolationLogger interface {
	// LogViolation logs a single violation
	LogViolation(ctx context.Context, violation *types.PolicyViolation) error

	// LogViolations logs multiple violations
	LogViolations(ctx context.Context, violations []*types.PolicyViolation) error

	// GetViolations retrieves violations with filtering
	GetViolations(ctx context.Context, filter *types.ViolationFilter) ([]*types.PolicyViolation, error)

	// GetViolationStats returns violation statistics
	GetViolationStats(ctx context.Context, timeRange types.TimeRange) (*types.ViolationStats, error)

	// GetViolationTrends returns violation trends over time
	GetViolationTrends(ctx context.Context, timeRange types.TimeRange, granularity time.Duration) ([]*types.ViolationTrend, error)

	// ExportViolations exports violations to external format
	ExportViolations(ctx context.Context, filter *types.ViolationFilter, format string) ([]byte, error)
}

// ViolationNotifier defines the interface for violation notifications
type ViolationNotifier interface {
	// NotifyViolation sends notification for a violation
	NotifyViolation(ctx context.Context, violation *types.PolicyViolation) error

	// NotifyBatch sends notifications for multiple violations
	NotifyBatch(ctx context.Context, violations []*types.PolicyViolation) error

	// SetNotificationThreshold sets minimum severity for notifications
	SetNotificationThreshold(severity types.ViolationSeverity) error

	// ConfigureChannels configures notification channels
	ConfigureChannels(ctx context.Context, channels []*types.NotificationChannel) error

	// TestNotification tests notification configuration
	TestNotification(ctx context.Context, channel string) error

	// GetNotificationHistory returns notification history
	GetNotificationHistory(ctx context.Context, filter *types.NotificationFilter) ([]*types.NotificationLog, error)
}

// ViolationAnalyzer defines the interface for analyzing violation patterns
type ViolationAnalyzer interface {
	// AnalyzePatterns analyzes violation patterns
	AnalyzePatterns(ctx context.Context, violations []*types.PolicyViolation) (*types.ViolationAnalysis, error)

	// DetectAnomalies detects anomalous violation patterns
	DetectAnomalies(ctx context.Context, timeRange types.TimeRange) ([]*types.ViolationAnomaly, error)

	// GetTopViolations returns most frequent violations
	GetTopViolations(ctx context.Context, timeRange types.TimeRange, limit int) ([]*types.ViolationSummary, error)

	// GetViolationsByRule returns violations grouped by rule
	GetViolationsByRule(ctx context.Context, timeRange types.TimeRange) (map[string][]*types.PolicyViolation, error)

	// GetViolationsBySeverity returns violations grouped by severity
	GetViolationsBySeverity(ctx context.Context, timeRange types.TimeRange) (map[types.ViolationSeverity][]*types.PolicyViolation, error)

	// PredictViolations predicts future violation trends
	PredictViolations(ctx context.Context, timeRange types.TimeRange) (*types.ViolationPrediction, error)

	// GenerateReport generates violation analysis report
	GenerateReport(ctx context.Context, config *types.ViolationReportConfig) (*types.ViolationReport, error)
}

// ViolationRemediator defines the interface for violation remediation
type ViolationRemediator interface {
	// SuggestRemediation suggests remediation actions for violations
	SuggestRemediation(ctx context.Context, violation *types.PolicyViolation) ([]*types.RemediationSuggestion, error)

	// AutoRemediate automatically remediates violations
	AutoRemediate(ctx context.Context, violation *types.PolicyViolation) (*types.RemediationResult, error)

	// ApplyRemediation applies a specific remediation action
	ApplyRemediation(ctx context.Context, violation *types.PolicyViolation, action *types.RemediationAction) error

	// TrackRemediation tracks remediation status
	TrackRemediation(ctx context.Context, violationID string) (*types.RemediationStatus, error)

	// GetRemediationHistory returns remediation history
	GetRemediationHistory(ctx context.Context, filter *types.RemediationFilter) ([]*types.RemediationLog, error)

	// ValidateRemediation validates remediation effectiveness
	ValidateRemediation(ctx context.Context, violationID string, remediationID string) (*types.RemediationValidation, error)
}

// ViolationMetrics defines the interface for violation metrics collection
type ViolationMetrics interface {
	// RecordViolation records a violation metric
	RecordViolation(ctx context.Context, violation *types.PolicyViolation) error

	// IncrementCounter increments a violation counter
	IncrementCounter(ctx context.Context, metric string, labels map[string]string) error

	// RecordLatency records violation detection latency
	RecordLatency(ctx context.Context, operation string, duration time.Duration) error

	// GetMetrics returns violation metrics
	GetMetrics(ctx context.Context, timeRange types.TimeRange) (*types.ViolationMetrics, error)

	// GetCounters returns violation counters
	GetCounters(ctx context.Context) (map[string]int64, error)

	// GetLatencyStats returns latency statistics
	GetLatencyStats(ctx context.Context, operation string) (*types.LatencyStats, error)

	// ExportMetrics exports metrics in Prometheus format
	ExportMetrics(ctx context.Context) ([]byte, error)
}
