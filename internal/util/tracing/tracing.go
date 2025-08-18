package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	tracingpkg "github.com/promptshield/promptshield/internal/observability/tracing"
)

// TraceHTTPRequest creates a span for HTTP request tracing
func TraceHTTPRequest(provider *tracingpkg.TracingProvider, ctx context.Context, method, path string) (context.Context, *tracingpkg.Span) {
	spanName := fmt.Sprintf("%s %s", method, path)
	spanCtx, span := provider.StartSpan(ctx, spanName, trace.WithSpanKind(tracingpkg.SpanKindServer))
	
	span.SetAttribute("http.method", method)
	span.SetAttribute("http.route", path)
	
	return spanCtx, span
}

// TraceGRPCRequest creates a span for gRPC request tracing
func TraceGRPCRequest(provider *tracingpkg.TracingProvider, ctx context.Context, method string) (context.Context, *tracingpkg.Span) {
	spanName := fmt.Sprintf("grpc %s", method)
	spanCtx, span := provider.StartSpan(ctx, spanName, trace.WithSpanKind(tracingpkg.SpanKindServer))
	
	span.SetAttribute("rpc.system", "grpc")
	span.SetAttribute("rpc.method", method)
	
	return spanCtx, span
}

// TraceDatabaseQuery creates a span for database query tracing
func TraceDatabaseQuery(provider *tracingpkg.TracingProvider, ctx context.Context, operation, table string) (context.Context, *tracingpkg.Span) {
	spanName := fmt.Sprintf("db %s %s", operation, table)
	spanCtx, span := provider.StartSpan(ctx, spanName, trace.WithSpanKind(tracingpkg.SpanKindClient))
	
	span.SetAttribute("db.operation", operation)
	span.SetAttribute("db.sql.table", table)
	
	return spanCtx, span
}

// TraceLLMRequest creates a span for LLM request tracing
func TraceLLMRequest(provider *tracingpkg.TracingProvider, ctx context.Context, provider_name, model string) (context.Context, *tracingpkg.Span) {
	spanName := fmt.Sprintf("llm %s %s", provider_name, model)
	spanCtx, span := provider.StartSpan(ctx, spanName, trace.WithSpanKind(tracingpkg.SpanKindClient))
	
	span.SetAttribute("llm.provider", provider_name)
	span.SetAttribute("llm.model", model)
	
	return spanCtx, span
}

// TraceRuleProcessing creates a span for rule processing
func TraceRuleProcessing(provider *tracingpkg.TracingProvider, ctx context.Context, ruleID string) (context.Context, *tracingpkg.Span) {
	spanName := fmt.Sprintf("rule %s", ruleID)
	spanCtx, span := provider.StartSpan(ctx, spanName, trace.WithSpanKind(tracingpkg.SpanKindInternal))
	
	span.SetAttribute("rule.id", ruleID)
	
	return spanCtx, span
}