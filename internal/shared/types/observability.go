package types

import "time"

// TraceAttribute represents a trace attribute
type TraceAttribute struct {
	Key   string
	Value interface{}
}

// SpanStatusCode represents span status codes
type SpanStatusCode int

const (
	SpanStatusCodeUnset SpanStatusCode = 0
	SpanStatusCodeOK    SpanStatusCode = 1
	SpanStatusCodeError SpanStatusCode = 2
)

// MetricTag represents a metric tag
type MetricTag struct {
	Key   string
	Value string
}

// EventFilter defines filtering criteria for event subscriptions
type EventFilter struct {
	EventTypes []string `json:"event_types,omitempty"`
	TenantID   string   `json:"tenant_id,omitempty"`
	Source     string   `json:"source,omitempty"`
}

// BroadcastEvent represents an event for broadcasting
type BroadcastEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// AlertFilter defines filtering for alert queries
type AlertFilter struct {
	Severity  ViolationSeverity `json:"severity,omitempty"`
	RuleID    string            `json:"rule_id,omitempty"`
	TimeRange TimeRange         `json:"time_range,omitempty"`
	Status    string            `json:"status,omitempty"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// LogFilter defines filtering for log collection
type LogFilter struct {
	Level     LogLevel  `json:"level,omitempty"`
	Source    string    `json:"source,omitempty"`
	TenantID  string    `json:"tenant_id,omitempty"`
	TimeRange TimeRange `json:"time_range"`
}

// LogQuery represents a log search query
type LogQuery struct {
	Query     string    `json:"query"`
	TimeRange TimeRange `json:"time_range"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LogLevel represents log levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LogMetrics represents log-based metrics
type LogMetrics struct {
	TotalLogs    int64              `json:"total_logs"`
	LogsByLevel  map[LogLevel]int64 `json:"logs_by_level"`
	LogsBySource map[string]int64   `json:"logs_by_source"`
	ErrorRate    float64            `json:"error_rate"`
	TimeRange    TimeRange          `json:"time_range"`
}
