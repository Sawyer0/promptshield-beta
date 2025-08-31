package api

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "strings"
    "time"

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
		startTime := time.Now()
		
		// Use structured logging with slog
		logger := slog.Default().With(
			"correlation_id", correlationID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
		
		logger.Info("request_started")
		
		next.ServeHTTP(ww, r)
		
		// Log the response
		duration := time.Since(startTime)
		logger.With(
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", duration.Milliseconds(),
		).Info("request_completed")
	})
}

// rateLimitMiddleware applies simple rate limiting for Security Gateway
func rateLimitMiddleware(quotaStore interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security Gateway uses simple environment-based rate limiting
			// Rate limiting is handled by PS_ENFORCER_RPS environment variable
			// Complex per-tenant quota management removed for simplicity
			next.ServeHTTP(w, r)
		})
	}
}

// getCorrelationID extracts correlation ID from request context
func getCorrelationID(r *http.Request) string {
	if r == nil {
		return ""
	}
	
	if v := r.Context().Value(correlationIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	
	// Fallback to header
	if id := r.Header.Get("X-PS-Correlation-ID"); id != "" {
		return id
	}
	
	return ""
}

// corsMiddleware handles Cross-Origin Resource Sharing for frontend access
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        
        // Allow requests from frontend development server
        allowedOrigins := []string{
            // Common dev ports
            "http://localhost:3000",
            "http://127.0.0.1:3000",
            "https://localhost:3000",
            "http://localhost:5173",
            "http://127.0.0.1:5173",
            "http://localhost:4173",
            "http://127.0.0.1:4173",
        }

        // Allow override via env: PS_CORS_ALLOWED_ORIGINS=origin1,origin2
        if v := strings.TrimSpace(os.Getenv("PS_CORS_ALLOWED_ORIGINS")); v != "" {
            for _, o := range strings.Split(v, ",") {
                if o = strings.TrimSpace(o); o != "" {
                    allowedOrigins = append(allowedOrigins, o)
                }
            }
        }
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		
        if allowed {
            // Inform caches that response may vary by Origin
            w.Header().Add("Vary", "Origin")
            w.Header().Set("Access-Control-Allow-Origin", origin)
        }
        
        // Set CORS headers
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-PS-Frontend-Auth, X-Tenant-ID, X-PS-Tenant-ID, X-PS-User-ID, X-PS-User-Name")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}
