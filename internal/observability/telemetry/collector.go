package telemetry

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
	netutil "github.com/promptshield/promptshield/internal/util/net"
)

// Collector implements contracts.TelemetryCollector with tracing capabilities (no global state)
type Collector struct {
	config *types.TelemetryConfig
	mu     sync.Mutex
	
	// Instance-based providers (no globals)
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
	meter          metric.Meter
	tracer         trace.Tracer
	conn           *grpc.ClientConn
	
	// Event exporter
	exporter *eventExporter
}

// NewCollector creates a new telemetry collector with dependency injection
func NewCollector(config *types.TelemetryConfig) *Collector {
	return &Collector{
		config: config,
	}
}

// Ensure Collector implements TelemetryCollector interface
var _ contracts.TelemetryCollector = (*Collector)(nil)

// Initialize initializes the telemetry collector with the given configuration
func (c *Collector) Initialize(ctx context.Context, config *types.TelemetryConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = config
	if !config.Enabled {
		return nil
	}

	if config.Timeout <= 0 {
		config.Timeout = 2 * time.Second
	}

	ep := netutil.SanitizeEndpoint(config.Endpoint)
	if ep == "" {
		return nil
	}

	// Create instance-based providers (no global state)
	mp, conn, err := createMeterProvider(ctx, config)
	if err != nil {
		return err
	}
	c.meterProvider = mp
	c.conn = conn

	if mp != nil {
		c.meter = mp.Meter("promptshield/telemetry")
	}

	// Create tracer provider if meter provider succeeded
	if conn != nil {
		tp, err := createTracerProvider(ctx, config, conn)
		if err != nil {
			return err
		}
		c.tracerProvider = tp
		if tp != nil {
			c.tracer = tp.Tracer("promptshield/telemetry")
		}
	}

	// Initialize event exporter
	c.exporter = newEventExporter(config, c.meter)

	return nil
}

// RecordEvent records a telemetry event
func (c *Collector) RecordEvent(ctx context.Context, event *types.TelemetryEvent) error {
	if c.exporter == nil {
		return nil
	}
	return c.exporter.exportEvent(ctx, event)
}

// CollectMetrics collects system and application metrics
func (c *Collector) CollectMetrics(ctx context.Context) (*types.PerformanceMetrics, error) {
	// Implementation would collect current performance metrics
	return &types.PerformanceMetrics{
		WindowStart: time.Now(),
		WindowEnd:   time.Now(),
	}, nil
}

// RecordLatency records latency measurements
func (c *Collector) RecordLatency(ctx context.Context, operation string, duration time.Duration, tags map[string]string) error {
	if c == nil || c.config == nil || !c.config.Enabled {
		return nil
	}
	// Implementation would record latency metrics
	return nil
}

// RecordThroughput records throughput measurements
func (c *Collector) RecordThroughput(ctx context.Context, operation string, count int64, tags map[string]string) error {
	if c == nil || c.config == nil || !c.config.Enabled {
		return nil
	}
	// Implementation would record throughput metrics
	return nil
}

// RecordErrorRate records error rate measurements
func (c *Collector) RecordErrorRate(ctx context.Context, operation string, errors int64, total int64, tags map[string]string) error {
	if c == nil || c.config == nil || !c.config.Enabled {
		return nil
	}
	// Implementation would record error rate metrics
	return nil
}

// GetMetrics returns collected metrics for a time range
func (c *Collector) GetMetrics(ctx context.Context, timeRange types.TimeRange) (*types.MetricsSnapshot, error) {
	if c == nil {
		return nil, nil
	}
	
	return &types.MetricsSnapshot{
		Timestamp: time.Now(),
		Service:   c.config.Service,
		Version:   c.config.Version,
		Metrics:   make(map[string]interface{}),
	}, nil
}

// GetConfig returns the current telemetry configuration
func (c *Collector) GetConfig() *types.TelemetryConfig {
	if c == nil {
		return nil
	}
	return c.config
}

// Flush flushes all pending telemetry data
func (c *Collector) Flush(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	if c.meterProvider != nil {
		if mp, ok := c.meterProvider.(*sdkmetric.MeterProvider); ok {
			if flushErr := mp.ForceFlush(ctx); flushErr != nil {
				err = flushErr
			}
		}
	}
	if c.tracerProvider != nil {
		if tp, ok := c.tracerProvider.(*sdktrace.TracerProvider); ok {
			if flushErr := tp.ForceFlush(ctx); flushErr != nil && err == nil {
				err = flushErr
			}
		}
	}
	return err
}

// Close closes the telemetry collector
func (c *Collector) Close() error {
	if c == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Shutdown(ctx)
}

// Shutdown flushes and closes telemetry providers
func (c *Collector) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err1, err2, err3 error
	
	if c.meterProvider != nil {
		if mp, ok := c.meterProvider.(*sdkmetric.MeterProvider); ok {
			err1 = mp.Shutdown(ctx)
		}
	}
	
	if c.tracerProvider != nil {
		if tp, ok := c.tracerProvider.(*sdktrace.TracerProvider); ok {
			err2 = tp.Shutdown(ctx)
		}
	}
	
	if c.conn != nil {
		err3 = c.conn.Close()
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return err3
}

// GetTracer returns the OpenTelemetry tracer for distributed tracing
func (c *Collector) GetTracer() trace.Tracer {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tracer
}

// StartSpan starts a new distributed trace span
func (c *Collector) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if c.tracer == nil {
		// Return no-op span if tracer not initialized
		return ctx, trace.SpanFromContext(ctx)
	}
	
	return c.tracer.Start(ctx, name, opts...)
}