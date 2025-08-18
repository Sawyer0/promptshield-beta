package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	contextutil "github.com/promptshield/promptshield/internal/util/context"
)

// Level represents log levels
type Level = slog.Level

// Common log levels
const (
	LevelDebug Level = slog.LevelDebug
	LevelInfo  Level = slog.LevelInfo
	LevelWarn  Level = slog.LevelWarn
	LevelError Level = slog.LevelError
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	component string
}

// New creates a new logger with component name
func New(component string) *Logger {
	return &Logger{
		Logger:    slog.Default(),
		component: component,
	}
}

// NewWithHandler creates a new logger with custom handler
func NewWithHandler(component string, handler slog.Handler) *Logger {
	return &Logger{
		Logger:    slog.New(handler),
		component: component,
	}
}

// Component returns the component name
func (l *Logger) Component() string {
	return l.component
}

// WithComponent creates a new logger with different component
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger:    l.Logger,
		component: component,
	}
}

// WithContext creates a logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	attrs := l.extractContextAttrs(ctx)
	return &Logger{
		Logger:    l.Logger.With(attrs...),
		component: l.component,
	}
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	attrs := make([]slog.Attr, 0, len(fields))
	for key, value := range fields {
		attrs = append(attrs, slog.Any(key, value))
	}
	return &Logger{
		Logger:    l.Logger.With(attrs...),
		component: l.component,
	}
}

// WithField creates a logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		Logger:    l.Logger.With(slog.Any(key, value)),
		component: l.component,
	}
}

// WithError creates a logger with error field
func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err.Error())
}

// extractContextAttrs extracts logging attributes from context
func (l *Logger) extractContextAttrs(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr
	
	// Add component
	if l.component != "" {
		attrs = append(attrs, slog.String("component", l.component))
	}
	
	// Extract common context values
	if requestID, ok := contextutil.GetRequestID(ctx); ok {
		attrs = append(attrs, slog.String("request_id", requestID))
	}
	if tenantID, ok := contextutil.GetTenantID(ctx); ok {
		attrs = append(attrs, slog.String("tenant_id", tenantID))
	}
	if userID, ok := contextutil.GetUserID(ctx); ok {
		attrs = append(attrs, slog.String("user_id", userID))
	}
	if correlationID, ok := contextutil.GetCorrelationID(ctx); ok {
		attrs = append(attrs, slog.String("correlation_id", correlationID))
	}
	if traceID, ok := contextutil.GetTraceID(ctx); ok {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}
	if spanID, ok := contextutil.GetSpanID(ctx); ok {
		attrs = append(attrs, slog.String("span_id", spanID))
	}
	if attempt, ok := contextutil.GetRetryAttempt(ctx); ok {
		attrs = append(attrs, slog.Int("retry_attempt", attempt))
	}
	
	return attrs
}

// Debug logs at debug level
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(msg, args...))
}

// DebugContext logs at debug level with context
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Debug(msg, args...)
}

// Info logs at info level
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(msg, args...))
}

// InfoContext logs at info level with context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Info(msg, args...)
}

// Warn logs at warn level
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(msg, args...))
}

// WarnContext logs at warn level with context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Warn(msg, args...)
}

// Error logs at error level
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(msg, args...))
}

// ErrorContext logs at error level with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Error(msg, args...)
}

// Fatal logs at error level and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(msg, args...))
	os.Exit(1)
}

// FatalContext logs at error level with context and exits
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Error(msg, args...)
	os.Exit(1)
}

// Panic logs at error level and panics
func (l *Logger) Panic(msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	l.Logger.Error(message)
	panic(message)
}

// PanicContext logs at error level with context and panics
func (l *Logger) PanicContext(ctx context.Context, msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	l.WithContext(ctx).Error(message)
	panic(message)
}

// LogRequest logs HTTP request information
func (l *Logger) LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	l.WithContext(ctx).Info("HTTP request completed",
		slog.String("method", method),
		slog.String("path", path),
		slog.Int("status_code", statusCode),
		slog.Duration("duration", duration),
	)
}

// LogError logs error with additional context
func (l *Logger) LogError(ctx context.Context, err error, msg string, args ...interface{}) {
	l.WithContext(ctx).WithError(err).Error(msg, args...)
}

// LogPerformance logs performance metrics
func (l *Logger) LogPerformance(ctx context.Context, operation string, duration time.Duration, metadata map[string]interface{}) {
	logger := l.WithContext(ctx).WithFields(metadata)
	logger.Info("Performance metric",
		slog.String("operation", operation),
		slog.Duration("duration", duration),
	)
}

// LogSecurity logs security-related events
func (l *Logger) LogSecurity(ctx context.Context, event string, severity string, details map[string]interface{}) {
	logger := l.WithContext(ctx).WithFields(details)
	logger.Error("Security event",
		slog.String("event", event),
		slog.String("severity", severity),
	)
}

// LogAudit logs audit trail events
func (l *Logger) LogAudit(ctx context.Context, action string, resource string, details map[string]interface{}) {
	logger := l.WithContext(ctx).WithFields(details)
	logger.Info("Audit event",
		slog.String("action", action),
		slog.String("resource", resource),
	)
}

// Configuration for logger setup
type Config struct {
	Level     Level
	Format    string // "json" or "text"
	AddSource bool
	Output    string // "stdout", "stderr", or file path
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:     LevelInfo,
		Format:    "json",
		AddSource: false,
		Output:    "stdout",
	}
}

// Setup configures the global logger
func Setup(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}
	
	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     config.Level,
		AddSource: config.AddSource,
	}
	
	// Determine output
	var output *os.File
	switch strings.ToLower(config.Output) {
	case "stdout", "":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// File output
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}
	
	// Create handler
	var handler slog.Handler
	switch strings.ToLower(config.Format) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}
	
	// Set as default
	slog.SetDefault(slog.New(handler))
	
	return nil
}

// GetLogger gets a logger for a component
func GetLogger(component string) *Logger {
	return New(component)
}

// SetLevel sets the log level for the default logger
func SetLevel(level Level) {
	// Note: slog doesn't provide a direct way to change level after creation
	// This would typically require recreating the handler
	Setup(&Config{
		Level:  level,
		Format: "json",
		Output: "stdout",
	})
}

// IsEnabled checks if a log level is enabled
func IsEnabled(level Level) bool {
	return slog.Default().Enabled(context.Background(), level)
}

// GetLevel returns the current log level
func GetLevel() Level {
	// slog doesn't provide a direct way to get current level
	// This is a simplified implementation
	if IsEnabled(LevelDebug) {
		return LevelDebug
	}
	if IsEnabled(LevelInfo) {
		return LevelInfo
	}
	if IsEnabled(LevelWarn) {
		return LevelWarn
	}
	return LevelError
}

// Flush flushes any buffered log entries
func Flush() {
	// slog handlers typically don't need explicit flushing
	// but if using a custom handler, this could be implemented
}

// Global logger functions for convenience
var defaultLogger = New("default")

// Debug logs at debug level using default logger
func Debug(msg string, args ...interface{}) {
	defaultLogger.Debug(msg, args...)
}

// DebugContext logs at debug level with context using default logger
func DebugContext(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.DebugContext(ctx, msg, args...)
}

// Info logs at info level using default logger
func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

// InfoContext logs at info level with context using default logger
func InfoContext(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

// Warn logs at warn level using default logger
func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

// WarnContext logs at warn level with context using default logger
func WarnContext(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.WarnContext(ctx, msg, args...)
}

// Error logs at error level using default logger
func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}

// ErrorContext logs at error level with context using default logger
func ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}

// Fatal logs at error level and exits using default logger
func Fatal(msg string, args ...interface{}) {
	defaultLogger.Fatal(msg, args...)
}

// FatalContext logs at error level with context and exits using default logger
func FatalContext(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.FatalContext(ctx, msg, args...)
}

// WithField creates a logger with an additional field
func WithField(key string, value interface{}) *Logger {
	return defaultLogger.WithField(key, value)
}

// WithFields creates a logger with additional fields
func WithFields(fields map[string]interface{}) *Logger {
	return defaultLogger.WithFields(fields)
}

// WithError creates a logger with error field
func WithError(err error) *Logger {
	return defaultLogger.WithError(err)
}

// WithContext creates a logger with context values
func WithContext(ctx context.Context) *Logger {
	return defaultLogger.WithContext(ctx)
}