// Deprecated: internal observability tracing shim removed in favor of OpenTelemetry.
// File kept to avoid breaking imports during transitional refactors.
package tracing

import "context"

type Attribute struct { Key string; Value any }
type Span interface{ End(error) }
type Tracer interface{ Start(context.Context, string, ...Attribute) (context.Context, Span) }

type noopSpan struct{}
func (noopSpan) End(error) {}

type noopTracer struct{}
func Noop() Tracer { return noopTracer{} }
func (noopTracer) Start(ctx context.Context, _ string, _ ...Attribute) (context.Context, Span) { return ctx, noopSpan{} }
