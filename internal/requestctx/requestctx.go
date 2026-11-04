// Package requestctx stores request-scoped identifiers (request, tenant,
// user and correlation IDs) on a context.Context under private keys.
package requestctx

import "context"

type key string

const (
	keyRequestID     key = "request_id"
	keyTenantID      key = "tenant_id"
	keyUserID        key = "user_id"
	keyCorrelationID key = "correlation_id"
)

// WithRequestID returns a copy of ctx carrying the request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyRequestID, id)
}

// GetRequestID returns the request ID stored on ctx, if any.
func GetRequestID(ctx context.Context) (string, bool) { return get(ctx, keyRequestID) }

// WithTenantID returns a copy of ctx carrying the tenant ID.
func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyTenantID, id)
}

// GetTenantID returns the tenant ID stored on ctx, if any.
func GetTenantID(ctx context.Context) (string, bool) { return get(ctx, keyTenantID) }

// WithUserID returns a copy of ctx carrying the user ID.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyUserID, id)
}

// GetUserID returns the user ID stored on ctx, if any.
func GetUserID(ctx context.Context) (string, bool) { return get(ctx, keyUserID) }

// WithCorrelationID returns a copy of ctx carrying the correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyCorrelationID, id)
}

// GetCorrelationID returns the correlation ID stored on ctx, if any.
func GetCorrelationID(ctx context.Context) (string, bool) { return get(ctx, keyCorrelationID) }

func get(ctx context.Context, k key) (string, bool) {
	v, ok := ctx.Value(k).(string)
	return v, ok
}
