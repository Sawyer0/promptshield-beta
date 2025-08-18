package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TraceHTTPRequest creates a span for HTTP request tracing
func TraceHTTPRequest(tracer trace.Tracer, ctx context.Context, method, path string) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("%s %s", method, path)
	spanCtx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))

	span.SetAttributes(
		attribute.String("http.method", method),
		attribute.String("http.route", path),
	)

	return spanCtx, span
}

// TraceGRPCRequest creates a span for gRPC request tracing
func TraceGRPCRequest(tracer trace.Tracer, ctx context.Context, method string) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("grpc %s", method)
	spanCtx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))

	span.SetAttributes(
		attribute.String("rpc.system", "grpc"),
		attribute.String("rpc.method", method),
	)

	return spanCtx, span
}

// TraceDatabaseQuery creates a span for database query tracing
func TraceDatabaseQuery(tracer trace.Tracer, ctx context.Context, operation, table string) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("db %s %s", operation, table)
	spanCtx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))

	span.SetAttributes(
		attribute.String("db.operation", operation),
		attribute.String("db.sql.table", table),
	)

	return spanCtx, span
}

// TraceLLMRequest creates a span for LLM request tracing
func TraceLLMRequest(tracer trace.Tracer, ctx context.Context, provider_name, model string) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("llm %s %s", provider_name, model)
	spanCtx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))

	span.SetAttributes(
		attribute.String("llm.provider", provider_name),
		attribute.String("llm.model", model),
	)

	return spanCtx, span
}

// TraceRuleProcessing creates a span for rule processing
func TraceRuleProcessing(tracer trace.Tracer, ctx context.Context, ruleID string) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("rule %s", ruleID)
	spanCtx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindInternal))

	span.SetAttributes(
		attribute.String("rule.id", ruleID),
	)

	return spanCtx, span
}
