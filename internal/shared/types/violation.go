package types

import (
	"time"
)

// PolicyViolation represents a policy violation found during scanning
type PolicyViolation struct {
	RuleID     string  `json:"rule_id"`
	Severity   string  `json:"severity"`
	Message    string  `json:"message"`
	Action     string  `json:"action"` // allow, deny, quarantine
	Confidence float64 `json:"confidence,omitempty"`

	// Additional fields for compatibility
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Category string `json:"category,omitempty"`

	// Response action fields
	ResponseAction      string `json:"response_action,omitempty"`
	ResponseMessage     string `json:"response_message,omitempty"`
	ResponseReplacement string `json:"response_replacement,omitempty"`

	// Timing information
	RuleTimeoutMs int64 `json:"rule_timeout_ms,omitempty"`
}

// ViolationSeverity represents violation severity levels
type ViolationSeverity string

const (
	SeverityLow      ViolationSeverity = "low"
	SeverityMedium   ViolationSeverity = "medium"
	SeverityHigh     ViolationSeverity = "high"
	SeverityCritical ViolationSeverity = "critical"
)

// ViolationAction represents what action to take on a violation
type ViolationAction string

const (
	ActionAllow      ViolationAction = "allow"
	ActionDeny       ViolationAction = "deny"
	ActionQuarantine ViolationAction = "quarantine"
	ActionRedact     ViolationAction = "redact"
)

// ViolationFilter represents a filter for violations
type ViolationFilter struct {
	Severity  *ViolationSeverity `json:"severity,omitempty"`
	RuleIDs   []string           `json:"rule_ids,omitempty"`
	StartTime *time.Time         `json:"start_time,omitempty"`
	EndTime   *time.Time         `json:"end_time,omitempty"`
	Limit     int                `json:"limit,omitempty"`
	Offset    int                `json:"offset,omitempty"`
	SortBy    string             `json:"sort_by,omitempty"`
	SortOrder string             `json:"sort_order,omitempty"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
}

// ViolationStats represents violation statistics
type ViolationStats struct {
	TotalViolations      int64                       `json:"total_violations"`
	ViolationsByRule     map[string]int64            `json:"violations_by_rule"`
	ViolationsBySeverity map[ViolationSeverity]int64 `json:"violations_by_severity"`
	AverageSeverity      float64                     `json:"average_severity"`
	TimeRange            TimeRange                   `json:"time_range"`
	Metadata             map[string]interface{}      `json:"metadata,omitempty"`
}

// ViolationTrend represents a violation trend over time
type ViolationTrend struct {
	TimeBucket      time.Time        `json:"time_bucket"`
	ViolationCount  int64            `json:"violation_count"`
	SeverityAverage float64          `json:"severity_average"`
	RuleBreakdown   map[string]int64 `json:"rule_breakdown,omitempty"`
}

// NotificationChannel represents a notification channel configuration
type NotificationChannel struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // "email", "slack", "webhook", "sms"
	Config     map[string]interface{} `json:"config"`
	Enabled    bool                   `json:"enabled"`
	Recipients []string               `json:"recipients,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationFilter represents a filter for notifications
type NotificationFilter struct {
	ChannelID string                 `json:"channel_id,omitempty"`
	Type      string                 `json:"type,omitempty"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Offset    int                    `json:"offset,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationLog represents a notification log entry
type NotificationLog struct {
	ID          string                 `json:"id"`
	ChannelID   string                 `json:"channel_id"`
	ChannelName string                 `json:"channel_name"`
	Type        string                 `json:"type"`
	Recipients  []string               `json:"recipients"`
	Subject     string                 `json:"subject,omitempty"`
	Message     string                 `json:"message"`
	Status      string                 `json:"status"` // "sent", "failed", "pending"
	SentAt      *time.Time             `json:"sent_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ViolationAnalysis represents analysis of violation patterns
type ViolationAnalysis struct {
	TimeRange            TimeRange                   `json:"time_range"`
	TotalViolations      int64                       `json:"total_violations"`
	UniqueRules          int                         `json:"unique_rules"`
	TopViolatingRules    []RuleViolationSummary      `json:"top_violating_rules,omitempty"`
	SeverityDistribution map[ViolationSeverity]int64 `json:"severity_distribution"`
	Trends               []ViolationTrend            `json:"trends,omitempty"`
	Anomalies            []ViolationAnomaly          `json:"anomalies,omitempty"`
	GeneratedAt          time.Time                   `json:"generated_at"`
	Metadata             map[string]interface{}      `json:"metadata,omitempty"`
}

// ViolationAnomaly represents an anomalous violation pattern
type ViolationAnomaly struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"` // "spike", "drop", "pattern_change"
	RuleID        string                 `json:"rule_id,omitempty"`
	Severity      ViolationSeverity      `json:"severity"`
	DetectedAt    time.Time              `json:"detected_at"`
	Value         float64                `json:"value"`
	ExpectedValue float64                `json:"expected_value"`
	Confidence    float64                `json:"confidence"`
	Description   string                 `json:"description,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ViolationSummary represents a summary of violations
type ViolationSummary struct {
	RuleID          string                 `json:"rule_id"`
	RuleName        string                 `json:"rule_name"`
	Count           int64                  `json:"count"`
	Percentage      float64                `json:"percentage"`
	AverageSeverity float64                `json:"average_severity"`
	LastOccurrence  time.Time              `json:"last_occurrence"`
	Trend           string                 `json:"trend"` // "increasing", "decreasing", "stable"
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// RuleViolationSummary represents a summary of rule violations
type RuleViolationSummary struct {
	RuleID          string                 `json:"rule_id"`
	RuleName        string                 `json:"rule_name"`
	Count           int64                  `json:"count"`
	Percentage      float64                `json:"percentage"`
	AverageSeverity float64                `json:"average_severity"`
	LastOccurrence  time.Time              `json:"last_occurrence"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ViolationPrediction represents a prediction of future violations
type ViolationPrediction struct {
	TimeRange      TimeRange              `json:"time_range"`
	PredictedCount int64                  `json:"predicted_count"`
	Confidence     float64                `json:"confidence"`
	Trend          string                 `json:"trend"`      // "increasing", "decreasing", "stable"
	RiskLevel      string                 `json:"risk_level"` // "low", "medium", "high"
	Factors        []string               `json:"factors,omitempty"`
	GeneratedAt    time.Time              `json:"generated_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ViolationReportConfig represents configuration for violation reports
type ViolationReportConfig struct {
	Type           string                 `json:"type"`
	TimeRange      TimeRange              `json:"time_range"`
	Format         string                 `json:"format"` // "json", "csv", "pdf"
	Filters        *ViolationFilter       `json:"filters,omitempty"`
	IncludeDetails bool                   `json:"include_details"`
	Recipients     []string               `json:"recipients,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ViolationReport represents a violation report
type ViolationReport struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	GeneratedAt time.Time              `json:"generated_at"`
	TimeRange   TimeRange              `json:"time_range"`
	Summary     ViolationStats         `json:"summary"`
	Analysis    *ViolationAnalysis     `json:"analysis,omitempty"`
	Predictions *ViolationPrediction   `json:"predictions,omitempty"`
	Format      string                 `json:"format"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationSuggestion represents a remediation suggestion for a violation
type RemediationSuggestion struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "rule_update", "content_fix", "policy_change"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Priority    int                    `json:"priority"`
	Effort      string                 `json:"effort"` // "low", "medium", "high"
	Impact      string                 `json:"impact"` // "low", "medium", "high"
	Steps       []string               `json:"steps,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationResult represents the result of a remediation action
type RemediationResult struct {
	ID          string                 `json:"id"`
	ViolationID string                 `json:"violation_id"`
	Action      string                 `json:"action"`
	Status      string                 `json:"status"` // "success", "failed", "partial"
	AppliedAt   time.Time              `json:"applied_at"`
	Duration    time.Duration          `json:"duration"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationAction represents a remediation action
type RemediationAction struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationStatus represents the status of a remediation
type RemediationStatus struct {
	ViolationID   string                 `json:"violation_id"`
	RemediationID string                 `json:"remediation_id"`
	Status        string                 `json:"status"` // "pending", "in_progress", "completed", "failed"
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Progress      float64                `json:"progress"`
	Message       string                 `json:"message,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationFilter represents a filter for remediation history
type RemediationFilter struct {
	ViolationID string                 `json:"violation_id,omitempty"`
	Status      string                 `json:"status,omitempty"`
	StartTime   *time.Time             `json:"start_time,omitempty"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationLog represents a remediation log entry
type RemediationLog struct {
	ID            string                 `json:"id"`
	ViolationID   string                 `json:"violation_id"`
	RemediationID string                 `json:"remediation_id"`
	Action        string                 `json:"action"`
	Status        string                 `json:"status"`
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Duration      time.Duration          `json:"duration"`
	Message       string                 `json:"message,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationValidation represents validation of remediation effectiveness
type RemediationValidation struct {
	ViolationID      string                 `json:"violation_id"`
	RemediationID    string                 `json:"remediation_id"`
	Effective        bool                   `json:"effective"`
	Effectiveness    float64                `json:"effectiveness"` // 0.0 to 1.0
	ValidatedAt      time.Time              `json:"validated_at"`
	ValidationPeriod time.Duration          `json:"validation_period"`
	Reoccurrences    int64                  `json:"reoccurrences"`
	Message          string                 `json:"message,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ViolationMetrics represents violation metrics
type ViolationMetrics struct {
	TimeRange            TimeRange                   `json:"time_range"`
	TotalViolations      int64                       `json:"total_violations"`
	ViolationsByRule     map[string]int64            `json:"violations_by_rule"`
	ViolationsBySeverity map[ViolationSeverity]int64 `json:"violations_by_severity"`
	AverageSeverity      float64                     `json:"average_severity"`
	DetectionLatency     time.Duration               `json:"detection_latency"`
	FalsePositiveRate    float64                     `json:"false_positive_rate"`
	TruePositiveRate     float64                     `json:"true_positive_rate"`
	Metadata             map[string]interface{}      `json:"metadata,omitempty"`
}

// LatencyStats represents latency statistics
type LatencyStats struct {
	Operation string        `json:"operation"`
	Count     int64         `json:"count"`
	Average   time.Duration `json:"average"`
	P50       time.Duration `json:"p50"`
	P95       time.Duration `json:"p95"`
	P99       time.Duration `json:"p99"`
	Min       time.Duration `json:"min"`
	Max       time.Duration `json:"max"`
	TimeRange TimeRange     `json:"time_range"`
}
