package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Context keys
type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	tenantIDKey      contextKey = "tenant_id"
	userIDKey        contextKey = "user_id"
)

// correlationIDMiddleware adds correlation ID to requests with tracing support
func correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-PS-Correlation-ID")
		if correlationID == "" {
			// Try to get from trace context if available
			if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
				correlationID = span.SpanContext().TraceID().String()
			} else {
				correlationID = uuid.New().String()
			}
		}
		
		// Add to response header
		w.Header().Set("X-PS-Correlation-ID", correlationID)
		
		// Add to span attributes if tracing is active
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
			span.SetAttributes(attribute.String("correlation.id", correlationID))
		}
		
		// Add to context
		ctx := context.WithValue(r.Context(), correlationIDKey, correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// tenantContextMiddleware extracts and validates tenant context with tracing support
func tenantContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-PS-Tenant-ID")
		
		// For admin endpoints, tenant ID might be optional or in the URL
		// For user endpoints, it should be required and validated against token claims
		
		if tenantID != "" {
			// Add tenant to span attributes if tracing is active
			if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
				span.SetAttributes(attribute.String("tenant.id", tenantID))
			}
			
			ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// requestLoggerMiddleware logs all requests with correlation ID
func requestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		
		// Log the request with correlation ID
		correlationID := getCorrelationID(r)
		
		// TODO: Add proper structured logging here
		// log.Info("request_started",
		//     "correlation_id", correlationID,
		//     "method", r.Method,
		//     "path", r.URL.Path,
		//     "remote_addr", r.RemoteAddr,
		// )
		
		next.ServeHTTP(ww, r)
		
		// Log the response
		// TODO: Add proper structured logging here
		// log.Info("request_completed",
		//     "correlation_id", correlationID,
		//     "status", ww.Status(),
		//     "bytes", ww.BytesWritten(),
		// )
		_ = correlationID // silence unused variable warning for now
	})
}

// rateLimitMiddleware applies rate limiting per tenant
func rateLimitMiddleware(quotaStore interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting if no quota store
			if quotaStore == nil {
				next.ServeHTTP(w, r)
				return
			}
			
			// Get tenant ID from context
			tenantID := getTenantID(r)
			if tenantID == "" {
				// No tenant context, allow request
				next.ServeHTTP(w, r)
				return
			}
			
			// Check rate limit
			// TODO: Implement actual rate limiting check
			// if !quotaStore.Allow(tenantID) {
			//     writeErrorJSON(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", 
			//         "Too many requests", nil, r)
			//     return
			// }
			
			next.ServeHTTP(w, r)
		})
	}
}