package telemetry

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "os"
    "strings"

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
    grpcCredentials "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/credentials/insecure"

    "github.com/promptshield/promptshield/internal/shared/types"
    stringutil "github.com/promptshield/promptshield/internal/util/strings"
)

// createMeterProvider creates a meter provider instance
func createMeterProvider(ctx context.Context, config *types.TelemetryConfig) (metric.MeterProvider, *grpc.ClientConn, error) {
    if !config.Enabled || config.Endpoint == "" {
        return nil, nil, nil
    }

    // Build transport security from environment variables (kept consistent with enforcer HTTP)
    // PS_OTEL_INSECURE (default false), PS_OTEL_CA_FILE, PS_OTEL_CLIENT_CERT_FILE, PS_OTEL_CLIENT_KEY_FILE, PS_OTEL_SERVER_NAME
    insecureEnv := strings.TrimSpace(os.Getenv("PS_OTEL_INSECURE"))
    insecureConn := strings.EqualFold(insecureEnv, "true") || insecureEnv == "1"
    var dialOpt grpc.DialOption
    if insecureConn {
        dialOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
    } else {
        tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
        if sni := strings.TrimSpace(os.Getenv("PS_OTEL_SERVER_NAME")); sni != "" { tlsCfg.ServerName = sni }
        if caFile := strings.TrimSpace(os.Getenv("PS_OTEL_CA_FILE")); caFile != "" {
            if pem, err := os.ReadFile(caFile); err == nil {
                pool := x509.NewCertPool()
                if pool.AppendCertsFromPEM(pem) { tlsCfg.RootCAs = pool }
            }
        }
        certFile := strings.TrimSpace(os.Getenv("PS_OTEL_CLIENT_CERT_FILE"))
        keyFile := strings.TrimSpace(os.Getenv("PS_OTEL_CLIENT_KEY_FILE"))
        if certFile != "" && keyFile != "" {
            if crt, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
                tlsCfg.Certificates = []tls.Certificate{crt}
            }
        }
        creds := grpcCredentials.NewTLS(tlsCfg)
        dialOpt = grpc.WithTransportCredentials(creds)
    }

    conn, err := grpc.DialContext(ctx, config.Endpoint, dialOpt)
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
