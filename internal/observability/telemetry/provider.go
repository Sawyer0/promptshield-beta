package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/promptshield/promptshield/internal/shared/types"
	stringutil "github.com/promptshield/promptshield/internal/util/strings"
)

// createMeterProvider creates a meter provider instance
func createMeterProvider(ctx context.Context, config *types.TelemetryConfig) (metric.MeterProvider, *grpc.ClientConn, error) {
	if !config.Enabled || config.Endpoint == "" {
		return nil, nil, nil
	}

	conn, err := grpc.DialContext(ctx, config.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	res, err := createResource(config)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
		sdkmetric.WithResource(res),
	)

	return mp, conn, nil
}

// createTracerProvider creates a tracer provider instance using telemetry config
func createTracerProvider(ctx context.Context, config *types.TelemetryConfig, conn *grpc.ClientConn) (trace.TracerProvider, error) {
	if !config.Enabled || config.Endpoint == "" {
		return nil, nil
	}

	texp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	sample := config.Sample
	if sample <= 0 {
		sample = 1.0
	}
	if sample > 1 {
		sample = 1.0
	}

	res, err := createResource(config)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(texp),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sample))),
		sdktrace.WithResource(res),
	)

	return tp, nil
}

// createResource creates OTel resource attributes
func createResource(config *types.TelemetryConfig) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(stringutil.Coalesce(config.Service, "promptshield")),
		semconv.ServiceVersionKey.String(stringutil.Coalesce(config.Version, "dev")),
	}
	if config.MachineID != "" {
		attrs = append(attrs, semconv.ServiceInstanceIDKey.String(config.MachineID))
	}

	return resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		attrs...,
	))
}
