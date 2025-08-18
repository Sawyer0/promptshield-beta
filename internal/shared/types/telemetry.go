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
	RequestCount        int64         `json:"request_count"`
	AverageLatencyMs    float64       `json:"average_latency_ms"`
	P50LatencyMs        float64       `json:"p50_latency_ms"`
	P95LatencyMs        float64       `json:"p95_latency_ms"`
	P99LatencyMs        float64       `json:"p99_latency_ms"`
	ErrorRate           float64       `json:"error_rate"`
	ThroughputRPS       float64       `json:"throughput_rps"`
	MemoryUsageBytes    int64         `json:"memory_usage_bytes"`
	CPUUsagePercent     float64       `json:"cpu_usage_percent"`
	ActiveConnections   int           `json:"active_connections"`
	WindowStart         time.Time     `json:"window_start"`
	WindowEnd           time.Time     `json:"window_end"`
	WindowDuration      time.Duration `json:"window_duration"`
}

// ScanMetrics represents scanning-specific performance metrics
type ScanMetrics struct {
	TotalScans           int64   `json:"total_scans"`
	Level1Scans          int64   `json:"level1_scans"`
	Level2Scans          int64   `json:"level2_scans"`
	Level3Scans          int64   `json:"level3_scans"`
	AverageLevel1Ms      float64 `json:"average_level1_ms"`
	AverageLevel2Ms      float64 `json:"average_level2_ms"`
	AverageLevel3Ms      float64 `json:"average_level3_ms"`
	ViolationsDetected   int64   `json:"violations_detected"`
	RequestsBlocked      int64   `json:"requests_blocked"`
	CacheHitRate         float64 `json:"cache_hit_rate"`
	SemanticAnalysisCalls int64  `json:"semantic_analysis_calls"`
	SemanticErrorRate    float64 `json:"semantic_error_rate"`
}

// ProviderMetrics represents LLM provider-specific metrics
type ProviderMetrics struct {
	Provider              Provider      `json:"provider"`
	TotalRequests         int64         `json:"total_requests"`
	SuccessfulRequests    int64         `json:"successful_requests"`
	FailedRequests        int64         `json:"failed_requests"`
	AverageLatencyMs      float64       `json:"average_latency_ms"`
	AverageTokensPrompt   float64       `json:"average_tokens_prompt"`
	AverageTokensCompletion float64     `json:"average_tokens_completion"`
	TotalTokensUsed       int64         `json:"total_tokens_used"`
	RateLimitHits         int64         `json:"rate_limit_hits"`
	QuotaExceededCount    int64         `json:"quota_exceeded_count"`
	ErrorsByType          map[string]int64 `json:"errors_by_type"`
	WindowStart           time.Time     `json:"window_start"`
	WindowEnd             time.Time     `json:"window_end"`
}

// HealthStatus represents the health status of a service component
type HealthStatus struct {
	Component   string                 `json:"component"`
	Status      HealthStatusType       `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastCheck   time.Time              `json:"last_check"`
	ResponseTime time.Duration         `json:"response_time"`
	Details     map[string]interface{} `json:"details,omitempty"`
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
	TraceID      TraceID                `json:"trace_id"`
	SpanID       SpanID                 `json:"span_id"`
	ParentSpanID SpanID                 `json:"parent_span_id,omitempty"`
	OperationName string                `json:"operation_name"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	Tags         map[string]interface{} `json:"tags,omitempty"`
	Logs         []TraceLog             `json:"logs,omitempty"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
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
	Severity    Severity               `json:"severity"`
	Enabled     bool                   `json:"enabled"`
	Recipients  []string               `json:"recipients,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AlertCondition represents alert condition types
type AlertCondition string

const (
	AlertConditionGreaterThan    AlertCondition = "greater_than"
	AlertConditionLessThan       AlertCondition = "less_than"
	AlertConditionEquals         AlertCondition = "equals"
	AlertConditionNotEquals      AlertCondition = "not_equals"
	AlertConditionIncreaseBy     AlertCondition = "increase_by"
	AlertConditionDecreaseBy     AlertCondition = "decrease_by"
)

// Alert represents a triggered alert
type Alert struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Status      AlertStatus            `json:"status"`
	Severity    Severity               `json:"severity"`
	Message     string                 `json:"message"`
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	TriggerTime time.Time              `json:"trigger_time"`
	ResolveTime *time.Time             `json:"resolve_time,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"
)