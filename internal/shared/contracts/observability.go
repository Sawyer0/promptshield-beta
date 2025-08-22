package contracts

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Tracer defines the interface for distributed tracing
type Tracer interface {
	// Start starts a new trace span
	Start(ctx context.Context, name string, attrs ...types.TraceAttribute) (context.Context, Span)
	
	// Extract extracts trace context from headers
	Extract(ctx context.Context, headers map[string]string) context.Context
	
	// Inject injects trace context into headers
	Inject(ctx context.Context, headers map[string]string)
}

// Span defines the interface for trace spans
type Span interface {
	// End finishes the span
	End(err error)
	
	// SetAttribute sets an attribute on the span
	SetAttribute(key string, value interface{})
	
	// SetStatus sets the span status
	SetStatus(code types.SpanStatusCode, description string)
	
	// AddEvent adds an event to the span
	AddEvent(name string, attrs ...types.TraceAttribute)
	
	// GetTraceID returns the trace ID
	GetTraceID() string
	
	// GetSpanID returns the span ID
	GetSpanID() string
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	
	// LogWithContext logs an audit event with context
	LogWithContext(ctx context.Context, event types.AuditEvent) error
	
	// Flush flushes any pending audit logs
	Flush() error
	
	// Close closes the audit logger
	Close() error
}

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// Counter increments a counter metric
	Counter(name string, value float64, tags ...types.MetricTag)
	
	// Gauge sets a gauge metric
	Gauge(name string, value float64, tags ...types.MetricTag)
	
	// Histogram records a histogram metric
	Histogram(name string, value float64, tags ...types.MetricTag)
	
	// Timer records a timing metric
	Timer(name string, duration time.Duration, tags ...types.MetricTag)
	
	// Flush flushes metrics to the backend
	Flush() error
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	// Check performs a health check
	Check(ctx context.Context) *types.HealthStatus
	
	// CheckComponent performs a health check for a specific component
	CheckComponent(ctx context.Context, component string) *types.HealthStatus
	
	// RegisterCheck registers a health check function
	RegisterCheck(name string, check HealthCheckFunc)
	
	// GetStatus returns the overall health status
	GetStatus(ctx context.Context) map[string]*types.HealthStatus
}

// HealthCheckFunc defines a health check function
type HealthCheckFunc func(ctx context.Context) *types.HealthStatus

// EventBroadcaster defines the interface for event broadcasting (SSE, webhooks)
type EventBroadcaster interface {
	// Subscribe subscribes to events
	Subscribe(ctx context.Context, filter types.EventFilter) (<-chan *types.BroadcastEvent, error)
	
	// Publish publishes an event to all subscribers
	Publish(ctx context.Context, event *types.BroadcastEvent) error
	
	// Unsubscribe removes a subscription
	Unsubscribe(subscription string) error
	
	// GetSubscriberCount returns the number of active subscribers
	GetSubscriberCount() int
}

// AlertManager defines the interface for alert management
type AlertManager interface {
	// TriggerAlert triggers an alert
	TriggerAlert(ctx context.Context, alert *types.Alert) error
	
	// ResolveAlert resolves an alert
	ResolveAlert(ctx context.Context, alertID string) error
	
	// GetActiveAlerts returns active alerts
	GetActiveAlerts(ctx context.Context) ([]*types.Alert, error)
	
	// GetAlertHistory returns alert history
	GetAlertHistory(ctx context.Context, filter types.AlertFilter) ([]*types.Alert, error)
}

// LogAggregator defines the interface for log aggregation and analysis
type LogAggregator interface {
	// CollectLogs collects logs from various sources
	CollectLogs(ctx context.Context, filter types.LogFilter) ([]*types.LogEntry, error)
	
	// SearchLogs searches logs with query
	SearchLogs(ctx context.Context, query types.LogQuery) ([]*types.LogEntry, error)
	
	// GetLogMetrics returns metrics about logs
	GetLogMetrics(ctx context.Context, timeRange types.TimeRange) (*types.LogMetrics, error)
}

// TelemetryProvider defines the interface for telemetry provider setup (no global state)
type TelemetryProvider interface {
	// CreateMeterProvider creates a new meter provider with the given config
	CreateMeterProvider(ctx context.Context, config *types.TelemetryConfig) (metric.MeterProvider, error)
	
	// CreateTracerProvider creates a new tracer provider with the given config
	CreateTracerProvider(ctx context.Context, config *types.TelemetryConfig) (trace.TracerProvider, error)
	
	// CreateMeter creates a meter from the provider
	CreateMeter(provider metric.MeterProvider, name string) metric.Meter
	
	// CreateTracer creates a tracer from the provider
	CreateTracer(provider trace.TracerProvider, name string) trace.Tracer
	
	// Shutdown cleanly shuts down providers
	Shutdown(ctx context.Context) error
}

// TelemetryExporter defines the interface for exporting telemetry data
type TelemetryExporter interface {
	// ExportEvent exports a telemetry event to configured destinations
	ExportEvent(ctx context.Context, event *types.TelemetryEvent) error
	
	// ExportToFile exports event to file
	ExportToFile(ctx context.Context, event *types.TelemetryEvent, filename string) error
	
	// ExportToOTel exports event to OpenTelemetry collector
	ExportToOTel(ctx context.Context, event *types.TelemetryEvent) error
	
	// Flush flushes all pending exports
	Flush(ctx context.Context) error
}

// TelemetryUtilities defines the interface for utility functions
type TelemetryUtilities interface {
	// SanitizeEndpoint sanitizes network endpoints
	SanitizeEndpoint(endpoint string) string
	
	// Coalesce returns first non-empty string
	Coalesce(values ...string) string
	
	// ToAttributes converts map to OpenTelemetry attributes
	ToAttributes(data map[string]interface{}, allowedKeys []string) []interface{}
}

