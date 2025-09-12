package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	otelslogbridge "go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/trace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
)

// LogConfig configures the logging system
type LogConfig struct {
	Level  string `json:"level"`  // debug, info, warn, error
	Format string `json:"format"` // json, text
	Output string `json:"output"` // stdout, stderr, file path
}

// CorrelationHandler adds correlation IDs and trace context to logs
type CorrelationHandler struct {
	handler slog.Handler
}

// NewCorrelationHandler creates a handler that adds correlation context
func NewCorrelationHandler(handler slog.Handler) *CorrelationHandler {
	return &CorrelationHandler{
		handler: handler,
	}
}

// Enabled implements slog.Handler
func (h *CorrelationHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle implements slog.Handler
func (h *CorrelationHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add trace context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	
	// Add correlation ID if available
	if corrID := ctx.Value("correlation_id"); corrID != nil {
		if id, ok := corrID.(string); ok {
			r.AddAttrs(slog.String("correlation_id", id))
		}
	}
	
	// Add tenant context if available
	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		if id, ok := tenantID.(string); ok {
			r.AddAttrs(slog.String("tenant_id", id))
		}
	}
	
	// Add request ID if available
	if reqID := ctx.Value("request_id"); reqID != nil {
		if id, ok := reqID.(string); ok {
			r.AddAttrs(slog.String("request_id", id))
		}
	}
	
	return h.handler.Handle(ctx, r)
}

// WithAttrs implements slog.Handler
func (h *CorrelationHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CorrelationHandler{
		handler: h.handler.WithAttrs(attrs),
	}
}

// WithGroup implements slog.Handler
func (h *CorrelationHandler) WithGroup(name string) slog.Handler {
	return &CorrelationHandler{
		handler: h.handler.WithGroup(name),
	}
}

// SetupLogging configures structured logging for the service
func SetupLogging(config LogConfig) (*slog.Logger, error) {
	// Parse log level
	var level slog.Level
	switch config.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	
	// Configure output
	var output io.Writer = os.Stdout
	if config.Output == "stderr" {
		output = os.Stderr
	} else if config.Output != "" && config.Output != "stdout" {
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		output = file
	}
	
	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: level <= slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Format time in ISO8601
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339Nano)),
				}
			}
			return a
		},
	}
	
	// Create base handler
	var baseHandler slog.Handler
	if config.Format == "json" {
		baseHandler = slog.NewJSONHandler(output, opts)
	} else {
		baseHandler = slog.NewTextHandler(output, opts)
	}

	// Wrap with correlation handler
	corr := NewCorrelationHandler(baseHandler)

	// Optionally create OTel logs handler and tee both
	var tee slog.Handler = corr
	if enableOTelLogs() {
		if h, err := newOTelLogsHandler(); err == nil && h != nil {
			tee = TeeHandler(corr, h)
		}
	}

	// Create logger with service context
	logger := slog.New(tee).With(
		"service", "promptshield",
		"version", GetVersion(),
		"pid", os.Getpid(),
	)

	// Set as default logger
	slog.SetDefault(logger)

	return logger, nil
}

// LogViolation logs a security violation with structured data
func LogViolation(ctx context.Context, tenantID, ruleID, severity string, matched string) {
	slog.InfoContext(ctx, "Security violation detected",
		"event_type", "security_violation",
		"tenant_id", tenantID,
		"rule_id", ruleID,
		"severity", severity,
		"matched_content", redactSensitive(matched),
		"timestamp", time.Now().UTC(),
	)
}

// LogRequest logs an HTTP request with timing
func LogRequest(ctx context.Context, method, path string, status int, duration time.Duration, size int64) {
	level := slog.LevelInfo
	if status >= 400 {
		level = slog.LevelWarn
	}
	if status >= 500 {
		level = slog.LevelError
	}
	
	slog.Log(ctx, level, "HTTP request processed",
		"event_type", "http_request",
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"response_size_bytes", size,
	)
}

// LogAlert logs an alert event
func LogAlert(ctx context.Context, alertType, severity, message string, metadata map[string]interface{}) {
	level := slog.LevelWarn
	if severity == "CRITICAL" {
		level = slog.LevelError
	}
	
	attrs := []slog.Attr{
		slog.String("event_type", "alert"),
		slog.String("alert_type", alertType),
		slog.String("severity", severity),
		slog.String("message", message),
	}
	
	// Add metadata as attributes
	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}
	
	slog.LogAttrs(ctx, level, "Alert triggered", attrs...)
}

// LogSystemEvent logs system events like startup, shutdown, configuration changes
func LogSystemEvent(ctx context.Context, event, message string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("event_type", "system"),
		slog.String("event", event),
		slog.String("message", message),
	}
	
	allAttrs := append(baseAttrs, attrs...)
	slog.LogAttrs(ctx, slog.LevelInfo, "System event", allAttrs...)
}

// LogPerformanceMetric logs performance-related events
func LogPerformanceMetric(ctx context.Context, metric string, value float64, unit string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("event_type", "performance"),
		slog.String("metric", metric),
		slog.Float64("value", value),
		slog.String("unit", unit),
		slog.Time("timestamp", time.Now().UTC()),
	}
	
	allAttrs := append(baseAttrs, attrs...)
	slog.LogAttrs(ctx, slog.LevelInfo, "Performance metric", allAttrs...)
}

// LogAuditEvent logs audit events for compliance
func LogAuditEvent(ctx context.Context, userID, action, resource string, success bool, attrs ...slog.Attr) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelWarn
	}
	
	baseAttrs := []slog.Attr{
		slog.String("event_type", "audit"),
		slog.String("user_id", userID),
		slog.String("action", action),
		slog.String("resource", resource),
		slog.Bool("success", success),
		slog.Time("timestamp", time.Now().UTC()),
	}
	
	allAttrs := append(baseAttrs, attrs...)
	slog.LogAttrs(ctx, level, "Audit event", allAttrs...)
}

// Helper functions

// TeeHandler tees two slog handlers.
func TeeHandler(a, b slog.Handler) slog.Handler { return &teeHandler{a: a, b: b} }

type teeHandler struct{ a, b slog.Handler }

func (h *teeHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.a.Enabled(ctx, l) || h.b.Enabled(ctx, l)
}
func (h *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	_ = h.a.Handle(ctx, r)
	_ = h.b.Handle(ctx, r)
	return nil
}
func (h *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &teeHandler{a: h.a.WithAttrs(attrs), b: h.b.WithAttrs(attrs)}
}
func (h *teeHandler) WithGroup(name string) slog.Handler {
	return &teeHandler{a: h.a.WithGroup(name), b: h.b.WithGroup(name)}
}

// OTel logs
func enableOTelLogs() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_OTEL_LOGS")))
	return v == "1" || v == "true" || v == "yes"
}

func newOTelLogsHandler() (slog.Handler, error) {
	endpoint := strings.TrimSpace(os.Getenv("PS_TELEMETRY_ENDPOINT"))
	if endpoint == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	exp, err := otlploggrpc.New(ctx, otlploggrpc.WithEndpoint(endpoint), otlploggrpc.WithInsecure())
	if err != nil { return nil, err }
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)))
	// Create an OTel log-capable slog handler via bridge (v0.13.0 API)
	h := otelslogbridge.NewHandler("promptshield",
		otelslogbridge.WithLoggerProvider(lp),
		otelslogbridge.WithSource(false),
	)
	return h, nil
}

func redactSensitive(content string) string {
	// Redact sensitive patterns while preserving structure for analysis
	if len(content) > 100 {
		return content[:50] + "...[REDACTED]..." + content[len(content)-20:]
	}
	return content
}

func GetVersion() string {
	// This would be injected at build time via ldflags
	return "0.2.0"
}

// SecurityLogger provides methods for security-specific logging
type SecurityLogger struct {
	logger *slog.Logger
}

// NewSecurityLogger creates a security-focused logger
func NewSecurityLogger() *SecurityLogger {
	logger := slog.With(
		"component", "security",
		"service", "promptshield",
	)
	return &SecurityLogger{logger: logger}
}

// LogThreatDetected logs when a threat is detected
func (s *SecurityLogger) LogThreatDetected(ctx context.Context, threat ThreatInfo) {
	s.logger.InfoContext(ctx, "Threat detected",
		"threat_type", threat.Type,
		"severity", threat.Severity,
		"source_ip", threat.SourceIP,
		"user_agent", threat.UserAgent,
		"rule_id", threat.RuleID,
		"confidence", threat.Confidence,
		"action_taken", threat.ActionTaken,
	)
}

// LogSecurityEvent logs general security events
func (s *SecurityLogger) LogSecurityEvent(ctx context.Context, event SecurityEvent) {
	level := slog.LevelInfo
	if event.Severity == "HIGH" || event.Severity == "CRITICAL" {
		level = slog.LevelWarn
	}
	
	s.logger.Log(ctx, level, event.Message,
		"event_type", "security_event",
		"category", event.Category,
		"severity", event.Severity,
		"tenant_id", event.TenantID,
		"metadata", event.Metadata,
	)
}

// ThreatInfo contains information about a detected threat
type ThreatInfo struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	SourceIP    string  `json:"source_ip,omitempty"`
	UserAgent   string  `json:"user_agent,omitempty"`
	RuleID      string  `json:"rule_id"`
	Confidence  float64 `json:"confidence"`
	ActionTaken string  `json:"action_taken"`
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	Category string                 `json:"category"`
	Severity string                 `json:"severity"`
	Message  string                 `json:"message"`
	TenantID string                 `json:"tenant_id"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
