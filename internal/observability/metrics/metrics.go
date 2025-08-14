package metrics

import (
	"io"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/promptshield/promptshield/internal/encoding/jsonx"
)

// Summary captures aggregate metrics across a scan run.
type Summary struct {
	Files            int   `json:"files"`
	Violations       int   `json:"violations"`
	BytesRead        int64 `json:"bytes_read"`
	LinesRead        int64 `json:"lines_read"`
	DurationMsMin    int64 `json:"duration_ms_min,omitempty"`
	DurationMsMax    int64 `json:"duration_ms_max,omitempty"`
	DurationMsSum    int64 `json:"duration_ms_sum,omitempty"`
	RegexAttempts    int64 `json:"regex_attempts,omitempty"`
	RegexSkipped     int64 `json:"regex_skipped,omitempty"`
	SemanticAttempts int64 `json:"semantic_attempts,omitempty"`
	SemanticSkipped  int64 `json:"semantic_skipped,omitempty"`
}

// NDJSONWriter writes one JSON object (summary) to the provided writer.
type NDJSONWriter struct {
	w  io.Writer
	mu sync.Mutex
}

func NewNDJSONWriter(w io.Writer) *NDJSONWriter { return &NDJSONWriter{w: w} }

func (n *NDJSONWriter) WriteSummary(s Summary) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	enc := jsonx.NewEncoder(n.w)
	return enc.Encode(map[string]interface{}{
		"type": "metrics",
		"run":  s,
	})
}

// Write writes an arbitrary JSON object (one per line). Intended for generic event metrics.
func (n *NDJSONWriter) Write(obj map[string]any) error {
	if n == nil || n.w == nil {
		return io.ErrClosedPipe
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	enc := jsonx.NewEncoder(n.w)
	return enc.Encode(obj)
}

// TelemetryWriter writes coarse, privacy-first telemetry events to a writer.
// Event schema: { "type": string, "ts": int64, "payload": object }
type TelemetryWriter struct {
	w  io.Writer
	mu sync.Mutex
}

func NewTelemetryWriter(w io.Writer) *TelemetryWriter { return &TelemetryWriter{w: w} }

func (t *TelemetryWriter) WriteEvent(eventType string, payload map[string]any) error {
	if t == nil || t.w == nil {
		return io.ErrClosedPipe
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	enc := jsonx.NewEncoder(t.w)
	obj := map[string]any{"type": eventType, "payload": payload}
	return enc.Encode(obj)
}

// Prometheus registry and metrics for CLI scans (optional usage)
var (
	scanDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ps_scan_duration_seconds",
			Help:    "Duration of per-file scans",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"level", "status"},
	)
	filesScanned = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_files_scanned_total", Help: "Total files scanned by CLI"},
	)
	violationsFound = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_violations_total", Help: "Total violations found by CLI"},
	)
)

// RegisterCLI registers Prometheus metrics for CLI if a registry is provided.
func RegisterCLI(reg prometheus.Registerer) {
	if reg == nil {
		return
	}
	reg.MustRegister(scanDuration, filesScanned, violationsFound)
}

// Expose helpers so callers can instrument without importing prometheus symbols.
func ObserveScanDuration(level, status string, seconds float64) {
	scanDuration.WithLabelValues(level, status).Observe(seconds)
}
func IncFilesScanned()    { filesScanned.Inc() }
func AddViolations(n int) { violationsFound.Add(float64(n)) }
