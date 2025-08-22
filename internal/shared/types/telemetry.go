package types

import "time"

// TelemetryConfig represents telemetry configuration
// From internal/observability/telemetry/telemetry.go
type TelemetryConfig struct {
	Enabled   bool          `json:"enabled"`
	Endpoint  string        `json:"endpoint,omitempty"`
	File      string        `json:"file,omitempty"`
	Sample    float64       `json:"sample"`
	MachineID string        `json:"machine_id,omitempty"`
	Timeout   time.Duration `json:"timeout"`
	Service   string        `json:"service"`
	Version   string        `json:"version"`
}

// TracingConfig represents tracing configuration
// From internal/observability/tracing/tracing.go
type TracingConfig struct {
	Enabled     bool    `json:"enabled"`
	ServiceName string  `json:"service_name"`
	Version     string  `json:"version"`
	Endpoint    string  `json:"endpoint,omitempty"`
	SampleRate  float64 `json:"sample_rate"`
	Insecure    bool    `json:"insecure"`
}

// TraceID represents a trace identifier
type TraceID string

// SpanID represents a span identifier
type SpanID string

// TelemetryEvent represents a telemetry event
type TelemetryEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	MachineID string                 `json:"machine_id,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
}

// MetricsSnapshot represents a point-in-time metrics snapshot
type MetricsSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// PerformanceMetrics represents performance-related metrics
type PerformanceMetrics struct {
	RequestCount      int64         `json:"request_count"`
	AverageLatencyMs  float64       `json:"average_latency_ms"`
	P50LatencyMs      float64       `json:"p50_latency_ms"`
	P95LatencyMs      float64       `json:"p95_latency_ms"`
	P99LatencyMs      float64       `json:"p99_latency_ms"`
	ErrorRate         float64       `json:"error_rate"`
	ThroughputRPS     float64       `json:"throughput_rps"`
	MemoryUsageBytes  int64         `json:"memory_usage_bytes"`
	CPUUsagePercent   float64       `json:"cpu_usage_percent"`
	ActiveConnections int           `json:"active_connections"`
	WindowStart       time.Time     `json:"window_start"`
	WindowEnd         time.Time     `json:"window_end"`
	WindowDuration    time.Duration `json:"window_duration"`
}

// ProviderMetrics represents LLM provider-specific metrics
type ProviderMetrics struct {
	Provider                Provider         `json:"provider"`
	TotalRequests           int64            `json:"total_requests"`
	SuccessfulRequests      int64            `json:"successful_requests"`
	FailedRequests          int64            `json:"failed_requests"`
	AverageLatencyMs        float64          `json:"average_latency_ms"`
	AverageTokensPrompt     float64          `json:"average_tokens_prompt"`
	AverageTokensCompletion float64          `json:"average_tokens_completion"`
	TotalTokensUsed         int64            `json:"total_tokens_used"`
	RateLimitHits           int64            `json:"rate_limit_hits"`
	QuotaExceededCount      int64            `json:"quota_exceeded_count"`
	ErrorsByType            map[string]int64 `json:"errors_by_type"`
	WindowStart             time.Time        `json:"window_start"`
	WindowEnd               time.Time        `json:"window_end"`
}

// HealthStatus represents the health status of a service component
type HealthStatus struct {
	Component    string                 `json:"component"`
	Status       HealthStatusType       `json:"status"`
	Message      string                 `json:"message,omitempty"`
	LastCheck    time.Time              `json:"last_check"`
	ResponseTime time.Duration          `json:"response_time"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// HealthStatusType represents the health status
type HealthStatusType string

const (
	HealthStatusHealthy   HealthStatusType = "healthy"
	HealthStatusDegraded  HealthStatusType = "degraded"
	HealthStatusUnhealthy HealthStatusType = "unhealthy"
	HealthStatusUnknown   HealthStatusType = "unknown"
)

// SystemInfo represents system information for telemetry
type SystemInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Version      string `json:"version"`
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCgoCall   int64  `json:"num_cgo_call"`
}

// TraceSpan represents a distributed tracing span
type TraceSpan struct {
	TraceID       TraceID                `json:"trace_id"`
	SpanID        SpanID                 `json:"span_id"`
	ParentSpanID  SpanID                 `json:"parent_span_id,omitempty"`
	OperationName string                 `json:"operation_name"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      time.Duration          `json:"duration"`
	Tags          map[string]interface{} `json:"tags,omitempty"`
	Logs          []TraceLog             `json:"logs,omitempty"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
}

// TraceLog represents a log entry within a trace span
type TraceLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// AlertRule represents an alerting rule configuration
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metric      string                 `json:"metric"`
	Condition   AlertCondition         `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Window      time.Duration          `json:"window"`
	Severity    ViolationSeverity      `json:"severity"`
	Enabled     bool                   `json:"enabled"`
	Recipients  []string               `json:"recipients,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AlertCondition represents alert condition types
type AlertCondition string

const (
	AlertConditionGreaterThan AlertCondition = "greater_than"
	AlertConditionLessThan    AlertCondition = "less_than"
	AlertConditionEquals      AlertCondition = "equals"
	AlertConditionNotEquals   AlertCondition = "not_equals"
	AlertConditionIncreaseBy  AlertCondition = "increase_by"
	AlertConditionDecreaseBy  AlertCondition = "decrease_by"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusPending  AlertStatus = "pending"
)

// Alert represents a telemetry alert
type Alert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Severity    string                 `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Enabled     bool                   `json:"enabled"`
	Conditions  map[string]interface{} `json:"conditions"`
	Actions     []string               `json:"actions,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NetworkMetrics represents network I/O metrics
type NetworkMetrics struct {
	BytesReceived   int64 `json:"bytes_received"`
	BytesSent       int64 `json:"bytes_sent"`
	PacketsReceived int64 `json:"packets_received"`
	PacketsSent     int64 `json:"packets_sent"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	CPUUsage    float64                `json:"cpu_usage"`
	MemoryUsage int64                  `json:"memory_usage_bytes"`
	DiskUsage   float64                `json:"disk_usage_percent"`
	NetworkIO   NetworkMetrics         `json:"network_io"`
	LoadAverage []float64              `json:"load_average"`
	Uptime      time.Duration          `json:"uptime"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ApplicationMetrics represents application-specific metrics
type ApplicationMetrics struct {
	RequestCount      int64                  `json:"request_count"`
	ErrorCount        int64                  `json:"error_count"`
	ResponseTime      time.Duration          `json:"response_time"`
	Throughput        float64                `json:"throughput"`
	ActiveConnections int                    `json:"active_connections"`
	Custom            map[string]interface{} `json:"custom,omitempty"`
}

// CustomMetric represents a custom metric
type CustomMetric interface {
	GetName() string
	GetValue() interface{}
	GetType() string
	GetTags() map[string]string
}

// Threshold represents a performance threshold
type Threshold struct {
	Value    float64                `json:"value"`
	Operator string                 `json:"operator"` // "gt", "lt", "eq", "gte", "lte"
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ThresholdViolation represents a threshold violation
type ThresholdViolation struct {
	MetricName   string                 `json:"metric_name"`
	Threshold    *Threshold             `json:"threshold"`
	CurrentValue float64                `json:"current_value"`
	ViolatedAt   time.Time              `json:"violated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProfileToken represents a profiling session token
type ProfileToken string

// ProfileData represents profiling data
type ProfileData struct {
	Token     ProfileToken           `json:"token"`
	Operation string                 `json:"operation"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Data      []byte                 `json:"data"`
	Type      string                 `json:"type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ProfileAnalysis represents analysis of profile data
type ProfileAnalysis struct {
	Token           ProfileToken            `json:"token"`
	Operation       string                  `json:"operation"`
	Hotspots        []ProfileHotspot        `json:"hotspots,omitempty"`
	Recommendations []ProfileRecommendation `json:"recommendations,omitempty"`
	Summary         string                  `json:"summary,omitempty"`
	Metadata        map[string]interface{}  `json:"metadata,omitempty"`
}

// ProfileHotspot represents a performance hotspot
type ProfileHotspot struct {
	Function   string        `json:"function"`
	File       string        `json:"file"`
	Line       int           `json:"line"`
	Percentage float64       `json:"percentage"`
	Time       time.Duration `json:"time"`
}

// ProfileRecommendation represents a performance recommendation
type ProfileRecommendation struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // "low", "medium", "high"
	Effort      string `json:"effort"` // "low", "medium", "high"
}

// AlertTrigger represents an alert trigger event
type AlertTrigger struct {
	AlertID     string                 `json:"alert_id"`
	AlertName   string                 `json:"alert_name"`
	TriggeredAt time.Time              `json:"triggered_at"`
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	Message     string                 `json:"message,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertHistory represents alert history
type AlertHistory struct {
	AlertID     string                 `json:"alert_id"`
	Status      AlertStatus            `json:"status"`
	TriggeredAt time.Time              `json:"triggered_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	Message     string                 `json:"message,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TraceEvent represents an event within a trace
type TraceEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Trace represents a complete distributed trace
type Trace struct {
	ID         TraceID                `json:"id"`
	RootSpanID SpanID                 `json:"root_span_id"`
	Spans      []*Span                `json:"spans"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Span represents a span within a trace
type Span struct {
	ID        SpanID                 `json:"id"`
	TraceID   TraceID                `json:"trace_id"`
	ParentID  *SpanID                `json:"parent_id,omitempty"`
	Name      string                 `json:"name"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Tags      map[string]interface{} `json:"tags,omitempty"`
	Events    []*TraceEvent          `json:"events,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TraceQuery represents a trace search query
type TraceQuery struct {
	TraceID     *TraceID               `json:"trace_id,omitempty"`
	ServiceName string                 `json:"service_name,omitempty"`
	Operation   string                 `json:"operation,omitempty"`
	StartTime   *time.Time             `json:"start_time,omitempty"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Tags        map[string]interface{} `json:"tags,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
}

// TraceStatistics represents trace statistics
type TraceStatistics struct {
	TimeRange       TimeRange     `json:"time_range"`
	TotalTraces     int64         `json:"total_traces"`
	TotalSpans      int64         `json:"total_spans"`
	AverageDuration time.Duration `json:"average_duration"`
	ErrorRate       float64       `json:"error_rate"`
	SlowestTraces   []*Trace      `json:"slowest_traces,omitempty"`
}

// ExportFormat represents the format for data export
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatXML  ExportFormat = "xml"
	ExportFormatYAML ExportFormat = "yaml"
)

// ExportSchedule represents a scheduled export configuration
type ExportSchedule struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Format         ExportFormat           `json:"format"`
	Destination    string                 `json:"destination"`
	CronExpression string                 `json:"cron_expression"`
	Enabled        bool                   `json:"enabled"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ExportStatus represents the status of an export operation
type ExportStatus struct {
	ExportID        string                 `json:"export_id"`
	Status          string                 `json:"status"` // "pending", "running", "completed", "failed"
	Progress        float64                `json:"progress"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	RecordsExported int64                  `json:"records_exported"`
	Error           string                 `json:"error,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ExportDestination represents an export destination
type ExportDestination struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // "s3", "gcs", "http", "file"
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Dashboard represents a telemetry dashboard
type Dashboard struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Widgets     []*DashboardWidget     `json:"widgets"`
	Layout      map[string]interface{} `json:"layout,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DashboardWidget represents a widget in a dashboard
type DashboardWidget struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "chart", "metric", "table", "text"
	Title    string                 `json:"title"`
	Config   map[string]interface{} `json:"config"`
	Position map[string]interface{} `json:"position,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DashboardData represents data for dashboard widgets
type DashboardData struct {
	DashboardID string                 `json:"dashboard_id"`
	Timestamp   time.Time              `json:"timestamp"`
	WidgetData  map[string]interface{} `json:"widget_data"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SharingPermissions represents permissions for sharing
type SharingPermissions struct {
	Read    bool       `json:"read"`
	Write   bool       `json:"write"`
	Admin   bool       `json:"admin"`
	Expires *time.Time `json:"expires,omitempty"`
}

// ReportConfig represents report generation configuration
type ReportConfig struct {
	Type      string                 `json:"type"`
	TimeRange TimeRange              `json:"time_range"`
	Format    ExportFormat           `json:"format"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Report represents a generated report
type Report struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	GeneratedAt time.Time              `json:"generated_at"`
	TimeRange   TimeRange              `json:"time_range"`
	Data        map[string]interface{} `json:"data"`
	Format      ExportFormat           `json:"format"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceReport represents a performance report
type PerformanceReport struct {
	ReportID        string                      `json:"report_id"`
	TimeRange       TimeRange                   `json:"time_range"`
	Summary         PerformanceSummary          `json:"summary"`
	TopIssues       []PerformanceIssue          `json:"top_issues,omitempty"`
	Recommendations []PerformanceRecommendation `json:"recommendations,omitempty"`
	GeneratedAt     time.Time                   `json:"generated_at"`
	Metadata        map[string]interface{}      `json:"metadata,omitempty"`
}

// PerformanceSummary represents a performance summary
type PerformanceSummary struct {
	AverageResponseTime time.Duration `json:"average_response_time"`
	P95ResponseTime     time.Duration `json:"p95_response_time"`
	P99ResponseTime     time.Duration `json:"p99_response_time"`
	Throughput          float64       `json:"throughput"`
	ErrorRate           float64       `json:"error_rate"`
	Availability        float64       `json:"availability"`
}

// PerformanceIssue represents a performance issue
type PerformanceIssue struct {
	Type           string                 `json:"type"`
	Severity       string                 `json:"severity"`
	Description    string                 `json:"description"`
	Impact         string                 `json:"impact"`
	Recommendation string                 `json:"recommendation,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceRecommendation represents a performance recommendation
type PerformanceRecommendation struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // "low", "medium", "high"
	Effort      string `json:"effort"` // "low", "medium", "high"
	Priority    int    `json:"priority"`
}

// UsageReport represents a usage report
type UsageReport struct {
	ReportID     string                 `json:"report_id"`
	TimeRange    TimeRange              `json:"time_range"`
	UserCount    int64                  `json:"user_count"`
	RequestCount int64                  `json:"request_count"`
	DataUsage    int64                  `json:"data_usage_bytes"`
	Cost         float64                `json:"cost,omitempty"`
	Breakdown    map[string]interface{} `json:"breakdown,omitempty"`
	GeneratedAt  time.Time              `json:"generated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorReport represents an error analysis report
type ErrorReport struct {
	ReportID    string                 `json:"report_id"`
	TimeRange   TimeRange              `json:"time_range"`
	TotalErrors int64                  `json:"total_errors"`
	ErrorRate   float64                `json:"error_rate"`
	TopErrors   []ErrorSummary         `json:"top_errors,omitempty"`
	Trends      []ErrorTrend           `json:"trends,omitempty"`
	GeneratedAt time.Time              `json:"generated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorSummary represents a summary of errors
type ErrorSummary struct {
	ErrorType      string    `json:"error_type"`
	Count          int64     `json:"count"`
	Percentage     float64   `json:"percentage"`
	LastOccurrence time.Time `json:"last_occurrence"`
}

// ErrorTrend represents an error trend
type ErrorTrend struct {
	TimeBucket time.Time `json:"time_bucket"`
	ErrorCount int64     `json:"error_count"`
	ErrorRate  float64   `json:"error_rate"`
}

// ReportSchedule represents a scheduled report configuration
type ReportSchedule struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	CronExpression string                 `json:"cron_expression"`
	Recipients     []string               `json:"recipients,omitempty"`
	Enabled        bool                   `json:"enabled"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ReportFilter represents a filter for reports
type ReportFilter struct {
	Type      string                 `json:"type,omitempty"`
	TimeRange *TimeRange             `json:"time_range,omitempty"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Anomaly represents an anomaly detected in metrics
type Anomaly struct {
	ID            string                 `json:"id"`
	MetricName    string                 `json:"metric_name"`
	DetectedAt    time.Time              `json:"detected_at"`
	Value         float64                `json:"value"`
	ExpectedValue float64                `json:"expected_value"`
	Confidence    float64                `json:"confidence"`
	Severity      string                 `json:"severity"` // "low", "medium", "high"
	Description   string                 `json:"description,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ModelInfo represents information about an anomaly detection model
type ModelInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Type        string                 `json:"type"`
	Accuracy    float64                `json:"accuracy"`
	LastTrained time.Time              `json:"last_trained"`
	Status      string                 `json:"status"` // "active", "training", "error"
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AnomalyFilter represents a filter for anomalies
type AnomalyFilter struct {
	MetricName string                 `json:"metric_name,omitempty"`
	Severity   string                 `json:"severity,omitempty"`
	StartTime  *time.Time             `json:"start_time,omitempty"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Limit      int                    `json:"limit,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ModelValidation represents validation results for a model
type ModelValidation struct {
	ModelID     string                 `json:"model_id"`
	Accuracy    float64                `json:"accuracy"`
	Precision   float64                `json:"precision"`
	Recall      float64                `json:"recall"`
	F1Score     float64                `json:"f1_score"`
	TestSamples int64                  `json:"test_samples"`
	ValidatedAt time.Time              `json:"validated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
