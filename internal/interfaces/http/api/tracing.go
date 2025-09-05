package api

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// tracingMiddleware extracts tracing context from incoming requests and creates spans
func tracingMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("promptshield/http")
	propagator := otel.GetTextMapPropagator()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract tracing context from headers
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Create span for the HTTP request
		spanName := r.Method + " " + r.URL.Path
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.route", r.URL.Path),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.remote_addr", r.RemoteAddr),
				attribute.String("net.peer.ip", r.Header.Get("X-Forwarded-For")),
				attribute.String("http.referer", r.Referer()),
				attribute.String("http.content_type", r.Header.Get("Content-Type")),
				attribute.Int64("http.content_length", r.ContentLength),
				attribute.String("ps.tenant_id", r.Header.Get("X-PS-Tenant-ID")),
				attribute.String("ps.request_id", r.Header.Get("X-Request-ID")),
				attribute.String("ps.tool_id", r.Header.Get("X-PS-Tool-ID")),
				attribute.String("ps.lane", r.Header.Get("X-PS-Lane")),
				attribute.String("ps.plan_hash", r.Header.Get("X-PS-Plan-Hash")),
				attribute.String("ps.conversation_id", r.Header.Get("X-PS-Conversation-ID")),
			),
		)
		defer span.End()

		// Add trace context to response headers for downstream services
		propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))

		// Wrap the response writer to capture status code
		wrapped := &tracingResponseWriter{
			ResponseWriter: w,
			statusCode:     200, // default status
		}

		// Continue with request processing
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Add response attributes to span
		span.SetAttributes(
			attribute.Int("http.status_code", wrapped.statusCode),
		)

		// Set span status based on HTTP status code
		if wrapped.statusCode >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	})
}

// tracingResponseWriter wraps http.ResponseWriter to capture status code
type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *tracingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *tracingResponseWriter) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}
