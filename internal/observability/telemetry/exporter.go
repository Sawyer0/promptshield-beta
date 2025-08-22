package telemetry

import (
	"context"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/promptshield/promptshield/internal/encoding/jsonx"
	"github.com/promptshield/promptshield/internal/shared/types"
	otelutil "github.com/promptshield/promptshield/internal/util/otel"
)

// eventExporter handles telemetry event exporting
type eventExporter struct {
	mu      sync.Mutex
	config  *types.TelemetryConfig
	cStart  metric.Int64Counter
	cScan   metric.Int64Counter
	cErr    metric.Int64Counter
}

// newEventExporter creates a new event exporter
func newEventExporter(config *types.TelemetryConfig, meter metric.Meter) *eventExporter {
	exporter := &eventExporter{
		config: config,
	}

	if meter != nil {
		exporter.cStart, _ = meter.Int64Counter("ps_startups_total")
		exporter.cScan, _ = meter.Int64Counter("ps_scans_total")
		exporter.cErr, _ = meter.Int64Counter("ps_errors_total")
	}

	return exporter
}

// exportEvent exports a telemetry event to configured destinations
func (e *eventExporter) exportEvent(ctx context.Context, event *types.TelemetryEvent) error {
	if e == nil || e.config == nil || !e.config.Enabled {
		return nil
	}

	// Check sampling
	if e.config.Sample < 1 {
		if (time.Now().UnixNano() % 1000) > int64(e.config.Sample*1000.0) {
			return nil
		}
	}

	// Record to OTel counters
	if err := e.recordToMetrics(ctx, event); err != nil {
		return err
	}

	// Write to file if configured
	if e.config.File != "" {
		return e.exportToFile(event)
	}

	return nil
}

// recordToMetrics records event to OTel metrics
func (e *eventExporter) recordToMetrics(ctx context.Context, event *types.TelemetryEvent) error {
	attrs := otelutil.ToAttributes(event.Payload, []string{"version", "os", "arch", "ci", "json_out", "output_format", "workers", "fail_on", "kind"})
	
	switch event.Type {
	case "startup", "setup":
		if e.cStart != nil {
			e.cStart.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	case "scan_summary":
		if e.cScan != nil {
			e.cScan.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	case "error_summary":
		if e.cErr != nil {
			e.cErr.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	}

	return nil
}

// exportToFile exports event to file
func (e *eventExporter) exportToFile(event *types.TelemetryEvent) error {
	evt := map[string]any{
		"type":       event.Type,
		"timestamp":  event.Timestamp.Unix(),
		"machine_id": event.MachineID,
		"payload":    event.Payload,
	}
	
	b, err := jsonx.Marshal(evt)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	f, err := os.OpenFile(e.config.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(b, '\n'))
	return err
}