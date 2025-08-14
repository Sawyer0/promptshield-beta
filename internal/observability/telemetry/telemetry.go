package telemetry

import (
	"context"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/promptshield/promptshield/internal/encoding/jsonx"
)

// Options configure the telemetry collector.
type Options struct {
	Enabled   bool
	Endpoint  string
	File      string
	Sample    float64
	MachineID string
	Timeout   time.Duration
	Service   string
	Version   string
}

// Collector is a coarse, privacy-first telemetry emitter backed by OpenTelemetry metrics.
type Collector struct {
	enabled   bool
	file      string
	sample    float64
	machineID string
	mu        sync.Mutex
	// OTel
	meter  metric.Meter
	mp     *sdkmetric.MeterProvider
	tp     *sdktrace.TracerProvider
	cStart metric.Int64Counter
	cScan  metric.Int64Counter
	cErr   metric.Int64Counter
}

func New(opts Options) *Collector {
	if !opts.Enabled {
		return &Collector{enabled: false}
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Second
	}
	ep := sanitizeEndpoint(opts.Endpoint)
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	var mp *sdkmetric.MeterProvider
	var tp *sdktrace.TracerProvider
	if ep != "" {
		conn, err := grpc.DialContext(ctx, ep, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
			if err == nil {
				// Resource: include service.name, version, and optional instance id
				attrs := []attribute.KeyValue{
					semconv.ServiceNameKey.String(coalesce(opts.Service, "promptshield")),
					semconv.ServiceVersionKey.String(coalesce(opts.Version, "dev")),
				}
				if opts.MachineID != "" {
					attrs = append(attrs, semconv.ServiceInstanceIDKey.String(opts.MachineID))
				}
				res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
					semconv.SchemaURL,
					attrs...,
				))
				mp = sdkmetric.NewMeterProvider(
					sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
					sdkmetric.WithResource(res),
				)
				otel.SetMeterProvider(mp)
				// Traces
				texp, err2 := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
				if err2 == nil {
					// Clamp sample rate to [0,1]
					sample := opts.Sample
					if sample <= 0 {
						sample = 1.0
					}
					if sample > 1 {
						sample = 1.0
					}
					tp = sdktrace.NewTracerProvider(
						sdktrace.WithBatcher(texp),
						sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sample))),
						sdktrace.WithResource(res),
					)
					otel.SetTracerProvider(tp)
					otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
				}
			}
		}
	}
	m := otel.Meter("promptshield/telemetry")
	cStart, _ := m.Int64Counter("ps_startups_total")
	cScan, _ := m.Int64Counter("ps_scans_total")
	cErr, _ := m.Int64Counter("ps_errors_total")
	return &Collector{
		enabled:   true,
		file:      opts.File,
		sample:    opts.Sample,
		machineID: opts.MachineID,
		meter:     m,
		mp:        mp,
		tp:        tp,
		cStart:    cStart,
		cScan:     cScan,
		cErr:      cErr,
	}
}

// Collect emits one event using OTel counters; payload must be coarse and secret-free.
func (c *Collector) Collect(eventType string, payload map[string]any) {
	if c == nil || !c.enabled {
		return
	}
	if c.sample < 1 {
		if (time.Now().UnixNano() % 1000) > int64(c.sample*1000.0) {
			return
		}
	}
	attrs := toAttrs(payload, []string{"version", "os", "arch", "ci", "json_out", "output_format", "workers", "fail_on", "kind"})
	switch eventType {
	case "startup", "setup":
		c.cStart.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	case "scan_summary":
		c.cScan.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	case "error_summary":
		c.cErr.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	}
	if c.file != "" {
		evt := map[string]any{"type": eventType, "ts": time.Now().UTC().Unix(), "machine_id": c.machineID, "payload": payload}
		b, _ := jsonx.Marshal(evt)
		c.mu.Lock()
		f, err := os.OpenFile(c.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = f.Write(append(b, '\n'))
			_ = f.Close()
		}
		c.mu.Unlock()
	}
}

// Helpers
func sanitizeEndpoint(s string) string {
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return strings.TrimPrefix(strings.TrimPrefix(s, "http://"), "https://")
	}
	return u.Host
}

func coalesce(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func toAttrs(m map[string]any, allow []string) []attribute.KeyValue {
	allowed := map[string]struct{}{}
	for _, k := range allow {
		allowed[k] = struct{}{}
	}
	var kvs []attribute.KeyValue
	for k, v := range m {
		if _, ok := allowed[k]; !ok {
			continue
		}
		switch x := v.(type) {
		case string:
			kvs = append(kvs, attribute.String(k, x))
		case bool:
			kvs = append(kvs, attribute.Bool(k, x))
		case int:
			kvs = append(kvs, attribute.Int(k, x))
		case int64:
			kvs = append(kvs, attribute.Int64(k, x))
		case float64:
			kvs = append(kvs, attribute.Float64(k, x))
		}
	}
	return kvs
}

// Shutdown flushes and closes telemetry providers.
func (c *Collector) Shutdown(ctx context.Context) error {
	if c == nil {
		return nil
	}
	var err1 error
	var err2 error
	if c.mp != nil {
		err1 = c.mp.Shutdown(ctx)
	}
	if c.tp != nil {
		err2 = c.tp.Shutdown(ctx)
	}
	if err1 != nil {
		return err1
	}
	return err2
}
