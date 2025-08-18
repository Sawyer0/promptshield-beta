package context

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ContextKey represents a type for context keys to avoid collisions
type ContextKey string

// Common context keys
const (
	KeyRequestID     ContextKey = "request_id"
	KeyTenantID      ContextKey = "tenant_id"
	KeyUserID        ContextKey = "user_id"
	KeyCorrelationID ContextKey = "correlation_id"
	KeySessionID     ContextKey = "session_id"
	KeyTraceID       ContextKey = "trace_id"
	KeySpanID        ContextKey = "span_id"
	KeyStartTime     ContextKey = "start_time"
	KeyTimeout       ContextKey = "timeout"
	KeyRetryAttempt  ContextKey = "retry_attempt"
)

// WithRequestID adds a request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, KeyRequestID, requestID)
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(KeyRequestID).(string)
	return value, ok
}

// WithTenantID adds a tenant ID to context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, KeyTenantID, tenantID)
}

// GetTenantID retrieves tenant ID from context
func GetTenantID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(KeyTenantID).(string)
	return value, ok
}

// WithUserID adds a user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, KeyUserID, userID)
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(KeyUserID).(string)
	return value, ok
}

// WithCorrelationID adds a correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, KeyCorrelationID, correlationID)
}

// GetCorrelationID retrieves correlation ID from context
func GetCorrelationID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(KeyCorrelationID).(string)
	return value, ok
}

// WithSessionID adds a session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, KeySessionID, sessionID)
}

// GetSessionID retrieves session ID from context
func GetSessionID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(KeySessionID).(string)
	return value, ok
}

// WithTraceID adds a trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, KeyTraceID, traceID)
}

// GetTraceID retrieves trace ID from context
func GetTraceID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(KeyTraceID).(string)
	return value, ok
}

// WithSpanID adds a span ID to context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, KeySpanID, spanID)
}

// GetSpanID retrieves span ID from context
func GetSpanID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(KeySpanID).(string)
	return value, ok
}

// WithStartTime adds a start time to context
func WithStartTime(ctx context.Context, startTime time.Time) context.Context {
	return context.WithValue(ctx, KeyStartTime, startTime)
}

// GetStartTime retrieves start time from context
func GetStartTime(ctx context.Context) (time.Time, bool) {
	value, ok := ctx.Value(KeyStartTime).(time.Time)
	return value, ok
}

// WithRetryAttempt adds retry attempt number to context
func WithRetryAttempt(ctx context.Context, attempt int) context.Context {
	return context.WithValue(ctx, KeyRetryAttempt, attempt)
}

// GetRetryAttempt retrieves retry attempt number from context
func GetRetryAttempt(ctx context.Context) (int, bool) {
	value, ok := ctx.Value(KeyRetryAttempt).(int)
	return value, ok
}

// WithTimeout creates a context with timeout
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithDeadline creates a context with deadline
func WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(parent, deadline)
}

// WithCancel creates a cancelable context
func WithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// Background returns a background context
func Background() context.Context {
	return context.Background()
}

// TODO returns a TODO context
func TODO() context.Context {
	return context.TODO()
}

// IsCanceled checks if context is canceled
func IsCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// IsTimeout checks if context error is due to timeout
func IsTimeout(ctx context.Context) bool {
	return ctx.Err() == context.DeadlineExceeded
}

// WaitForCancel waits for context cancellation
func WaitForCancel(ctx context.Context) {
	<-ctx.Done()
}

// WaitForCancelWithTimeout waits for context cancellation or timeout
func WaitForCancelWithTimeout(ctx context.Context, timeout time.Duration) bool {
	select {
	case <-ctx.Done():
		return true
	case <-time.After(timeout):
		return false
	}
}

// CombineContext combines multiple contexts into one
func CombineContext(contexts ...context.Context) (context.Context, context.CancelFunc) {
	if len(contexts) == 0 {
		return WithCancel(Background())
	}
	if len(contexts) == 1 {
		return WithCancel(contexts[0])
	}
	
	ctx, cancel := WithCancel(Background())
	
	var wg sync.WaitGroup
	for _, c := range contexts {
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			<-ctx.Done()
			cancel()
		}(c)
	}
	
	return ctx, cancel
}

// MergeValues merges values from multiple contexts
func MergeValues(parent context.Context, sources ...context.Context) context.Context {
	ctx := parent
	
	for _, source := range sources {
		// This is a simplified merge - in practice, you'd need reflection
		// to iterate over all values in a context, which isn't directly possible
		// This demonstrates the pattern for known keys
		
		if requestID, ok := GetRequestID(source); ok {
			ctx = WithRequestID(ctx, requestID)
		}
		if tenantID, ok := GetTenantID(source); ok {
			ctx = WithTenantID(ctx, tenantID)
		}
		if userID, ok := GetUserID(source); ok {
			ctx = WithUserID(ctx, userID)
		}
		if correlationID, ok := GetCorrelationID(source); ok {
			ctx = WithCorrelationID(ctx, correlationID)
		}
		if sessionID, ok := GetSessionID(source); ok {
			ctx = WithSessionID(ctx, sessionID)
		}
		if traceID, ok := GetTraceID(source); ok {
			ctx = WithTraceID(ctx, traceID)
		}
		if spanID, ok := GetSpanID(source); ok {
			ctx = WithSpanID(ctx, spanID)
		}
		if startTime, ok := GetStartTime(source); ok {
			ctx = WithStartTime(ctx, startTime)
		}
		if attempt, ok := GetRetryAttempt(source); ok {
			ctx = WithRetryAttempt(ctx, attempt)
		}
	}
	
	return ctx
}

// ExtractValues extracts common values to a map
func ExtractValues(ctx context.Context) map[string]interface{} {
	values := make(map[string]interface{})
	
	if requestID, ok := GetRequestID(ctx); ok {
		values["request_id"] = requestID
	}
	if tenantID, ok := GetTenantID(ctx); ok {
		values["tenant_id"] = tenantID
	}
	if userID, ok := GetUserID(ctx); ok {
		values["user_id"] = userID
	}
	if correlationID, ok := GetCorrelationID(ctx); ok {
		values["correlation_id"] = correlationID
	}
	if sessionID, ok := GetSessionID(ctx); ok {
		values["session_id"] = sessionID
	}
	if traceID, ok := GetTraceID(ctx); ok {
		values["trace_id"] = traceID
	}
	if spanID, ok := GetSpanID(ctx); ok {
		values["span_id"] = spanID
	}
	if startTime, ok := GetStartTime(ctx); ok {
		values["start_time"] = startTime
	}
	if attempt, ok := GetRetryAttempt(ctx); ok {
		values["retry_attempt"] = attempt
	}
	
	return values
}

// NewRequestContext creates a new context with request tracking
func NewRequestContext(parent context.Context) context.Context {
	requestID := uuid.New().String()
	correlationID := uuid.New().String()
	startTime := time.Now()
	
	ctx := WithRequestID(parent, requestID)
	ctx = WithCorrelationID(ctx, correlationID)
	ctx = WithStartTime(ctx, startTime)
	
	return ctx
}

// NewTraceContext creates a new context with tracing information
func NewTraceContext(parent context.Context, traceID, spanID string) context.Context {
	ctx := WithTraceID(parent, traceID)
	ctx = WithSpanID(ctx, spanID)
	return ctx
}

// ContextWithValues creates a context with multiple values
func ContextWithValues(parent context.Context, values map[ContextKey]interface{}) context.Context {
	ctx := parent
	for key, value := range values {
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}

// GetValue gets a value from context with type assertion
func GetValue[T any](ctx context.Context, key ContextKey) (T, bool) {
	value, ok := ctx.Value(key).(T)
	return value, ok
}

// SetValue sets a value in context
func SetValue[T any](ctx context.Context, key ContextKey, value T) context.Context {
	return context.WithValue(ctx, key, value)
}

// TimeoutContext creates a context that times out after specified duration
func TimeoutContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// DeadlineContext creates a context that expires at specified time
func DeadlineContext(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(parent, deadline)
}

// CancelableContext creates a cancelable context
func CancelableContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// ForEach executes a function for each context value (limited to known keys)
func ForEach(ctx context.Context, fn func(key string, value interface{})) {
	values := ExtractValues(ctx)
	for key, value := range values {
		fn(key, value)
	}
}

// Copy creates a copy of context without deadline/cancellation
func Copy(ctx context.Context) context.Context {
	return MergeValues(Background(), ctx)
}

// Detach creates a detached context that preserves values but removes cancellation
func Detach(ctx context.Context) context.Context {
	return Copy(ctx)
}

// IsValid checks if context is valid (not nil and not canceled)
func IsValid(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	return ctx.Err() == nil
}

// RemainingTime returns remaining time until context deadline
func RemainingTime(ctx context.Context) (time.Duration, bool) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0, false
	}
	return time.Until(deadline), true
}

// HasDeadline checks if context has a deadline
func HasDeadline(ctx context.Context) bool {
	_, ok := ctx.Deadline()
	return ok
}

// GetDeadline returns context deadline
func GetDeadline(ctx context.Context) (time.Time, bool) {
	return ctx.Deadline()
}

// GetError returns context error
func GetError(ctx context.Context) error {
	return ctx.Err()
}

// ExecuteWithTimeout executes a function with timeout
func ExecuteWithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	errChan := make(chan error, 1)
	go func() {
		errChan <- fn(timeoutCtx)
	}()
	
	select {
	case err := <-errChan:
		return err
	case <-timeoutCtx.Done():
		return timeoutCtx.Err()
	}
}

// ExecuteWithCancel executes a function with cancellation
func ExecuteWithCancel(ctx context.Context, fn func(context.Context) error) (error, context.CancelFunc) {
	cancelCtx, cancel := context.WithCancel(ctx)
	
	errChan := make(chan error, 1)
	go func() {
		errChan <- fn(cancelCtx)
	}()
	
	go func() {
		select {
		case <-errChan:
			// Function completed
		case <-cancelCtx.Done():
			// Context was canceled
		}
	}()
	
	return <-errChan, cancel
}