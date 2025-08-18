package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// ScanResultProcessor defines the interface for processing scan results
type ScanResultProcessor interface {
	// ProcessResult processes a single scan result
	ProcessResult(ctx context.Context, result *types.ScanResult) error

	// ProcessBatch processes multiple scan results
	ProcessBatch(ctx context.Context, results []*types.ScanResult) error

	// MergeResults merges multiple scan results into one
	MergeResults(ctx context.Context, results []*types.ScanResult) (*types.ScanResult, error)

	// EnrichResult enriches scan result with additional metadata
	EnrichResult(ctx context.Context, result *types.ScanResult) (*types.ScanResult, error)

	// ValidateResult validates scan result integrity
	ValidateResult(ctx context.Context, result *types.ScanResult) error

	// TransformResult transforms scan result format
	TransformResult(ctx context.Context, result *types.ScanResult, format string) (interface{}, error)
}

// ScanResultStore defines the interface for storing and retrieving scan results
type ScanResultStore interface {
	// Store stores a scan result
	Store(ctx context.Context, result *types.ScanResult) error

	// StoreBatch stores multiple scan results
	StoreBatch(ctx context.Context, results []*types.ScanResult) error

	// GetByID retrieves a scan result by ID
	GetByID(ctx context.Context, resultID string) (*types.ScanResult, error)

	// GetByScanID retrieves all results for a scan
	GetByScanID(ctx context.Context, scanID string) ([]*types.ScanResult, error)

	// Query queries scan results with filters
	Query(ctx context.Context, filter *types.ScanResultFilter) ([]*types.ScanResult, error)

	// Count returns count of scan results matching filter
	Count(ctx context.Context, filter *types.ScanResultFilter) (int64, error)

	// Delete deletes scan results matching filter
	Delete(ctx context.Context, filter *types.ScanResultFilter) error

	// Archive archives old scan results
	Archive(ctx context.Context, olderThan time.Time) error
}

// EnforcementDecisionMaker defines the interface for making enforcement decisions
type EnforcementDecisionMaker interface {
	// MakeDecision makes an enforcement decision based on scan result
	MakeDecision(ctx context.Context, result *types.ScanResult) (*types.EnforcementDecision, error)

	// MakeBatchDecision makes enforcement decisions for multiple results
	MakeBatchDecision(ctx context.Context, results []*types.ScanResult) ([]*types.EnforcementDecision, error)

	// EvaluatePolicy evaluates policy against scan result
	EvaluatePolicy(ctx context.Context, result *types.ScanResult, policy *types.Policy) (*types.PolicyEvaluation, error)

	// GetDecisionReason returns reason for enforcement decision
	GetDecisionReason(ctx context.Context, decision *types.EnforcementDecision) (string, error)

	// OverrideDecision overrides an enforcement decision
	OverrideDecision(ctx context.Context, resultID string, newDecision *types.EnforcementDecision) error

	// GetDecisionHistory returns history of decisions for a result
	GetDecisionHistory(ctx context.Context, resultID string) ([]*types.EnforcementDecision, error)
}

// ScanMetricsCollector defines the interface for collecting scan metrics
type ScanMetricsCollector interface {
	// RecordScanMetrics records metrics for a scan operation
	RecordScanMetrics(ctx context.Context, metrics *types.ScanMetrics) error

	// GetMetrics retrieves scan metrics for a time range
	GetMetrics(ctx context.Context, timeRange types.TimeRange) (*types.ScanMetricsAggregated, error)

	// GetPerformanceMetrics returns performance metrics for scanning
	GetPerformanceMetrics(ctx context.Context, timeRange types.TimeRange) (*types.ScanPerformanceMetrics, error)

	// GetAccuracyMetrics returns accuracy metrics for scanning
	GetAccuracyMetrics(ctx context.Context, timeRange types.TimeRange) (*types.ScanAccuracyMetrics, error)

	// RecordLatency records scan operation latency
	RecordLatency(ctx context.Context, operation string, duration time.Duration) error

	// RecordThroughput records scan throughput metrics
	RecordThroughput(ctx context.Context, bytesProcessed int64, duration time.Duration) error

	// GetThroughputStats returns throughput statistics
	GetThroughputStats(ctx context.Context, timeRange types.TimeRange) (*types.ThroughputStats, error)
}

// ResultAnalyzer defines the interface for analyzing scan results
type ResultAnalyzer interface {
	// AnalyzePatterns analyzes patterns in scan results
	AnalyzePatterns(ctx context.Context, results []*types.ScanResult) (*types.PatternAnalysis, error)

	// DetectAnomalies detects anomalous scan results
	DetectAnomalies(ctx context.Context, results []*types.ScanResult) ([]*types.ResultAnomaly, error)

	// GetTrendAnalysis returns trend analysis for scan results
	GetTrendAnalysis(ctx context.Context, timeRange types.TimeRange) (*types.ResultTrendAnalysis, error)

	// GetFalsePositiveRate calculates false positive rate
	GetFalsePositiveRate(ctx context.Context, timeRange types.TimeRange) (float64, error)

	// GetFalseNegativeRate calculates false negative rate
	GetFalseNegativeRate(ctx context.Context, timeRange types.TimeRange) (float64, error)

	// CompareScanners compares performance of different scanners
	CompareScanners(ctx context.Context, scannerIDs []string, timeRange types.TimeRange) (*types.ScannerComparison, error)

	// GetConfidenceDistribution returns distribution of confidence scores
	GetConfidenceDistribution(ctx context.Context, timeRange types.TimeRange) (*types.ConfidenceDistribution, error)
}

// ResultNotifier defines the interface for result-based notifications
type ResultNotifier interface {
	// NotifyResult sends notification for a scan result
	NotifyResult(ctx context.Context, result *types.ScanResult) error

	// NotifyHighRiskResult sends notification for high-risk results
	NotifyHighRiskResult(ctx context.Context, result *types.ScanResult) error

	// NotifyBatch sends notifications for multiple results
	NotifyBatch(ctx context.Context, results []*types.ScanResult) error

	// ConfigureNotifications configures result-based notifications
	ConfigureNotifications(ctx context.Context, config *types.ResultNotificationConfig) error

	// SetThresholds sets notification thresholds for results
	SetThresholds(ctx context.Context, thresholds map[string]float64) error

	// GetNotificationHistory returns notification history
	GetNotificationHistory(ctx context.Context, filter *types.NotificationFilter) ([]*types.ResultNotification, error)
}

// ResultReporter defines the interface for scan result reporting
type ResultReporter interface {
	// GenerateReport generates a scan result report
	GenerateReport(ctx context.Context, config *types.ResultReportConfig) (*types.ResultReport, error)

	// GenerateSummaryReport generates a summary report
	GenerateSummaryReport(ctx context.Context, timeRange types.TimeRange) (*types.ResultSummaryReport, error)

	// GenerateDetectionReport generates a detection effectiveness report
	GenerateDetectionReport(ctx context.Context, timeRange types.TimeRange) (*types.DetectionReport, error)

	// GetTopThreats returns most frequently detected threats
	GetTopThreats(ctx context.Context, timeRange types.TimeRange, limit int) ([]*types.ThreatSummary, error)

	// GetScannerEffectiveness returns scanner effectiveness metrics
	GetScannerEffectiveness(ctx context.Context, scannerID string, timeRange types.TimeRange) (*types.ScannerEffectiveness, error)

	// ExportResults exports scan results in specified format
	ExportResults(ctx context.Context, filter *types.ScanResultFilter, format string) ([]byte, error)

	// ScheduleReport schedules periodic result report generation
	ScheduleReport(ctx context.Context, config *types.ResultReportSchedule) error
}

// ResultValidator defines the interface for validating scan results
type ResultValidator interface {
	// ValidateResult validates a scan result for correctness
	ValidateResult(ctx context.Context, result *types.ScanResult) (*types.ValidationResult, error)

	// ValidateBatch validates multiple scan results
	ValidateBatch(ctx context.Context, results []*types.ScanResult) ([]*types.ValidationResult, error)

	// CheckConsistency checks consistency between related results
	CheckConsistency(ctx context.Context, results []*types.ScanResult) (*types.ConsistencyCheck, error)

	// ValidateConfidence validates confidence scores in results
	ValidateConfidence(ctx context.Context, result *types.ScanResult) error

	// ValidateMetadata validates result metadata
	ValidateMetadata(ctx context.Context, result *types.ScanResult) error

	// GetValidationRules returns current validation rules
	GetValidationRules() []*types.ValidationRule

	// SetValidationRules sets validation rules
	SetValidationRules(rules []*types.ValidationRule) error
}

// ResultEnricher defines the interface for enriching scan results
type ResultEnricher interface {
	// EnrichWithThreatIntel enriches result with threat intelligence
	EnrichWithThreatIntel(ctx context.Context, result *types.ScanResult) (*types.ScanResult, error)

	// EnrichWithContext enriches result with contextual information
	EnrichWithContext(ctx context.Context, result *types.ScanResult) (*types.ScanResult, error)

	// EnrichWithRiskScore enriches result with risk scoring
	EnrichWithRiskScore(ctx context.Context, result *types.ScanResult) (*types.ScanResult, error)

	// EnrichWithRemediation enriches result with remediation suggestions
	EnrichWithRemediation(ctx context.Context, result *types.ScanResult) (*types.ScanResult, error)

	// EnrichBatch enriches multiple results
	EnrichBatch(ctx context.Context, results []*types.ScanResult) ([]*types.ScanResult, error)

	// GetEnrichmentSources returns available enrichment sources
	GetEnrichmentSources() []string

	// ConfigureEnrichment configures enrichment settings
	ConfigureEnrichment(ctx context.Context, config *types.EnrichmentConfig) error
}
