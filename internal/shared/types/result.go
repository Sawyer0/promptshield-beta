package types

import (
	"time"

	"github.com/google/uuid"
)

// ScanResult represents the result of a security scan
type ScanResult struct {
	ID         string                 `json:"id"`
	Input      string                 `json:"input"`
	Violations []*PolicyViolation     `json:"violations"`
	Metrics    *ScanMetrics           `json:"metrics"`
	Decision   *EnforcementDecision   `json:"decision"`
	ScanInfo   *ScanInfo              `json:"scan_info"`
	TenantID   *uuid.UUID             `json:"tenant_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ScanMetrics represents metrics for a scan operation
type ScanMetrics struct {
	BytesRead        int64                  `json:"bytes_read"`
	LinesRead        int64                  `json:"lines_read"`
	RegexAttempts    int64                  `json:"regex_attempts,omitempty"`
	RegexSkipped     int64                  `json:"regex_skipped,omitempty"`
	SemanticAttempts int64                  `json:"semantic_attempts,omitempty"`
	SemanticSkipped  int64                  `json:"semantic_skipped,omitempty"`
	CacheHits        int                    `json:"cache_hits,omitempty"`
	CacheMisses      int                    `json:"cache_misses,omitempty"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ScanInfo represents information about a scan
type ScanInfo struct {
	TotalViolations  int                    `json:"total_violations"`
	ScanStatus       string                 `json:"scan_status"` // "success", "timeout", "error", "partial"
	ScanDurationMs   int64                  `json:"scan_duration_ms"`
	RulesProcessed   int                    `json:"rules_processed"`
	RulesSkipped     int                    `json:"rules_skipped,omitempty"`
	RulesTimedOut    int                    `json:"rules_timed_out,omitempty"`
	RulesErrored     int                    `json:"rules_errored,omitempty"`
	Level1DurationMs int64                  `json:"level1_duration_ms,omitempty"`
	Level2DurationMs int64                  `json:"level2_duration_ms,omitempty"`
	Level3DurationMs int64                  `json:"level3_duration_ms,omitempty"`
	ShouldBlock      bool                   `json:"should_block"`
	BlockReason      string                 `json:"block_reason,omitempty"`
	HighestSeverity  string                 `json:"highest_severity,omitempty"`
	TriggerRuleCount int                    `json:"trigger_rule_count"`
	PeakMemoryBytes  int64                  `json:"peak_memory_bytes,omitempty"`
	CPUTimeMs        int64                  `json:"cpu_time_ms,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// EnforcementDecision represents the result of policy enforcement
type EnforcementDecision struct {
	Allow       bool                   `json:"allow"`
	Reason      string                 `json:"reason,omitempty"`
	Violations  []*PolicyViolation     `json:"violations,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ProcessedAt time.Time              `json:"processed_at"`
	Latency     time.Duration          `json:"latency_ms"`
}

// ScanResultFilter represents a filter for querying scan results
type ScanResultFilter struct {
	ScanID    string                 `json:"scan_id,omitempty"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	Decision  string                 `json:"decision,omitempty"`
	Severity  *ViolationSeverity     `json:"severity,omitempty"`
	RuleIDs   []string               `json:"rule_ids,omitempty"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Offset    int                    `json:"offset,omitempty"`
	SortBy    string                 `json:"sort_by,omitempty"`
	SortOrder string                 `json:"sort_order,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyEvaluation represents the evaluation of a policy against a scan result
type PolicyEvaluation struct {
	PolicyID    string                 `json:"policy_id"`
	ResultID    string                 `json:"result_id"`
	Decision    string                 `json:"decision"`
	Reason      string                 `json:"reason"`
	Confidence  float64                `json:"confidence"`
	Violations  []*PolicyViolation     `json:"violations,omitempty"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ScanMetricsAggregated represents aggregated scan metrics
type ScanMetricsAggregated struct {
	TimeRange        TimeRange              `json:"time_range"`
	TotalScans       int64                  `json:"total_scans"`
	AllowedScans     int64                  `json:"allowed_scans"`
	DeniedScans      int64                  `json:"denied_scans"`
	QuarantinedScans int64                  `json:"quarantined_scans"`
	TotalViolations  int64                  `json:"total_violations"`
	AverageLatency   time.Duration          `json:"average_latency"`
	Throughput       float64                `json:"throughput"`
	ErrorRate        float64                `json:"error_rate"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ScanPerformanceMetrics represents performance metrics for scanning
type ScanPerformanceMetrics struct {
	TimeRange      TimeRange              `json:"time_range"`
	AverageLatency time.Duration          `json:"average_latency"`
	P50Latency     time.Duration          `json:"p50_latency"`
	P95Latency     time.Duration          `json:"p95_latency"`
	P99Latency     time.Duration          `json:"p99_latency"`
	Throughput     float64                `json:"throughput"`
	Concurrency    int                    `json:"concurrency"`
	MemoryUsage    int64                  `json:"memory_usage_bytes"`
	CPUUsage       float64                `json:"cpu_usage_percent"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ScanAccuracyMetrics represents accuracy metrics for scanning
type ScanAccuracyMetrics struct {
	TimeRange      TimeRange              `json:"time_range"`
	TruePositives  int64                  `json:"true_positives"`
	FalsePositives int64                  `json:"false_positives"`
	TrueNegatives  int64                  `json:"true_negatives"`
	FalseNegatives int64                  `json:"false_negatives"`
	Precision      float64                `json:"precision"`
	Recall         float64                `json:"recall"`
	F1Score        float64                `json:"f1_score"`
	Accuracy       float64                `json:"accuracy"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ThroughputStats represents throughput statistics
type ThroughputStats struct {
	TimeRange         TimeRange              `json:"time_range"`
	TotalBytes        int64                  `json:"total_bytes"`
	TotalRequests     int64                  `json:"total_requests"`
	AverageThroughput float64                `json:"average_throughput"`
	PeakThroughput    float64                `json:"peak_throughput"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// PatternAnalysis represents analysis of patterns in scan results
type PatternAnalysis struct {
	TimeRange TimeRange              `json:"time_range"`
	Patterns  []*Pattern             `json:"patterns"`
	Trends    []*Trend               `json:"trends"`
	Anomalies []*Anomaly             `json:"anomalies"`
	Summary   string                 `json:"summary"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ResultAnomaly represents an anomaly detected in scan results
type ResultAnomaly struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Description   string                 `json:"description"`
	Severity      string                 `json:"severity"`
	DetectedAt    time.Time              `json:"detected_at"`
	Value         float64                `json:"value"`
	ExpectedValue float64                `json:"expected_value"`
	Confidence    float64                `json:"confidence"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ResultTrendAnalysis represents trend analysis for scan results
type ResultTrendAnalysis struct {
	TimeRange   TimeRange              `json:"time_range"`
	Trends      []*Trend               `json:"trends"`
	Seasonality []*Seasonality         `json:"seasonality,omitempty"`
	Forecast    []*Forecast            `json:"forecast,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Trend represents a trend in data
type Trend struct {
	Metric     string                 `json:"metric"`
	Direction  string                 `json:"direction"` // "increasing", "decreasing", "stable"
	Slope      float64                `json:"slope"`
	Confidence float64                `json:"confidence"`
	StartValue float64                `json:"start_value"`
	EndValue   float64                `json:"end_value"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Seasonality represents seasonal patterns
type Seasonality struct {
	Period     time.Duration          `json:"period"`
	Amplitude  float64                `json:"amplitude"`
	Phase      float64                `json:"phase"`
	Confidence float64                `json:"confidence"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Forecast represents a forecast prediction
type Forecast struct {
	Metric     string                 `json:"metric"`
	Value      float64                `json:"value"`
	Confidence float64                `json:"confidence"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ScannerComparison represents comparison of different scanners
type ScannerComparison struct {
	TimeRange  TimeRange              `json:"time_range"`
	Scanners   []*ScannerMetrics      `json:"scanners"`
	Comparison map[string]interface{} `json:"comparison"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ScannerMetrics represents metrics for a specific scanner
type ScannerMetrics struct {
	ScannerID  string                 `json:"scanner_id"`
	Accuracy   float64                `json:"accuracy"`
	Precision  float64                `json:"precision"`
	Recall     float64                `json:"recall"`
	F1Score    float64                `json:"f1_score"`
	Latency    time.Duration          `json:"latency"`
	Throughput float64                `json:"throughput"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ConfidenceDistribution represents distribution of confidence scores
type ConfidenceDistribution struct {
	TimeRange TimeRange              `json:"time_range"`
	Buckets   []*ConfidenceBucket    `json:"buckets"`
	Mean      float64                `json:"mean"`
	Median    float64                `json:"median"`
	StdDev    float64                `json:"std_dev"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ConfidenceBucket represents a bucket in confidence distribution
type ConfidenceBucket struct {
	Min        float64                `json:"min"`
	Max        float64                `json:"max"`
	Count      int64                  `json:"count"`
	Percentage float64                `json:"percentage"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ResultNotificationConfig represents configuration for result notifications
type ResultNotificationConfig struct {
	Enabled    bool                   `json:"enabled"`
	Channels   []string               `json:"channels"`
	Thresholds map[string]float64     `json:"thresholds"`
	Recipients []string               `json:"recipients"`
	Template   string                 `json:"template"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ResultNotification represents a result notification
type ResultNotification struct {
	ID        string                 `json:"id"`
	ResultID  string                 `json:"result_id"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	SentAt    time.Time              `json:"sent_at"`
	Delivered bool                   `json:"delivered"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ResultReportConfig represents configuration for result reports
type ResultReportConfig struct {
	Type       string                 `json:"type"`
	TimeRange  TimeRange              `json:"time_range"`
	Format     string                 `json:"format"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
	Recipients []string               `json:"recipients,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ResultReport represents a generated result report
type ResultReport struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	GeneratedAt time.Time              `json:"generated_at"`
	TimeRange   TimeRange              `json:"time_range"`
	Data        map[string]interface{} `json:"data"`
	Format      string                 `json:"format"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResultSummaryReport represents a summary report of scan results
type ResultSummaryReport struct {
	ReportID      string                 `json:"report_id"`
	TimeRange     TimeRange              `json:"time_range"`
	Summary       *ScanSummary           `json:"summary"`
	TopViolations []*ViolationSummary    `json:"top_violations,omitempty"`
	GeneratedAt   time.Time              `json:"generated_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ScanSummary represents a summary of scan results
type ScanSummary struct {
	TotalScans       int64                  `json:"total_scans"`
	AllowedScans     int64                  `json:"allowed_scans"`
	DeniedScans      int64                  `json:"denied_scans"`
	QuarantinedScans int64                  `json:"quarantined_scans"`
	TotalViolations  int64                  `json:"total_violations"`
	AverageLatency   time.Duration          `json:"average_latency"`
	ErrorRate        float64                `json:"error_rate"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// DetectionReport represents a detection effectiveness report
type DetectionReport struct {
	ReportID      string                  `json:"report_id"`
	TimeRange     TimeRange               `json:"time_range"`
	Effectiveness *DetectionEffectiveness `json:"effectiveness"`
	TopThreats    []*ThreatSummary        `json:"top_threats,omitempty"`
	GeneratedAt   time.Time               `json:"generated_at"`
	Metadata      map[string]interface{}  `json:"metadata,omitempty"`
}

// DetectionEffectiveness represents detection effectiveness metrics
type DetectionEffectiveness struct {
	Precision         float64                `json:"precision"`
	Recall            float64                `json:"recall"`
	F1Score           float64                `json:"f1_score"`
	Accuracy          float64                `json:"accuracy"`
	FalsePositiveRate float64                `json:"false_positive_rate"`
	FalseNegativeRate float64                `json:"false_negative_rate"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ThreatSummary represents a summary of threats
type ThreatSummary struct {
	ThreatType string                 `json:"threat_type"`
	Count      int64                  `json:"count"`
	Percentage float64                `json:"percentage"`
	Severity   ViolationSeverity      `json:"severity"`
	LastSeen   time.Time              `json:"last_seen"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ScannerEffectiveness represents scanner effectiveness metrics
type ScannerEffectiveness struct {
	ScannerID     string                  `json:"scanner_id"`
	TimeRange     TimeRange               `json:"time_range"`
	Effectiveness *DetectionEffectiveness `json:"effectiveness"`
	Performance   *ScanPerformanceMetrics `json:"performance"`
	Metadata      map[string]interface{}  `json:"metadata,omitempty"`
}

// ResultReportSchedule represents a scheduled result report
type ResultReportSchedule struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	CronExpression string                 `json:"cron_expression"`
	Config         *ResultReportConfig    `json:"config"`
	Enabled        bool                   `json:"enabled"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ConsistencyCheck represents a consistency check between results
type ConsistencyCheck struct {
	CheckID    string                 `json:"check_id"`
	Results    []string               `json:"results"`
	Consistent bool                   `json:"consistent"`
	Issues     []*ConsistencyIssue    `json:"issues,omitempty"`
	CheckedAt  time.Time              `json:"checked_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ConsistencyIssue represents a consistency issue
type ConsistencyIssue struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	ResultIDs   []string               `json:"result_ids"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EnrichmentConfig represents configuration for result enrichment
type EnrichmentConfig struct {
	Enabled      bool                   `json:"enabled"`
	Sources      []string               `json:"sources"`
	Timeout      time.Duration          `json:"timeout"`
	MaxRetries   int                    `json:"max_retries"`
	CacheEnabled bool                   `json:"cache_enabled"`
	CacheTTL     time.Duration          `json:"cache_ttl"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
