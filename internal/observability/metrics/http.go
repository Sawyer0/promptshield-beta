package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	timeutil "github.com/promptshield/promptshield/internal/util/time"
)

// HTTP API metrics
var (
	HTTPBytesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_http_bytes_total",
			Help: "Total HTTP bytes in/out by path",
		},
		[]string{"direction", "path"},
	)
	ScanRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ps_scan_request_duration_seconds",
			Help:    "Duration of /v1/scan requests by mode",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"mode"},
	)
	ScanEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_scan_events_total",
			Help: "Total NDJSON decision events emitted per route",
		},
		[]string{"route"},
	)
)

func RecordScanDuration(mode string, duration time.Duration) {
	timeutil.ObserveDuration(ScanRequestDuration.WithLabelValues(mode), duration)
}

func init() {
	if Enabled() {
		prometheus.MustRegister(HTTPBytesTotal, ScanRequestDuration, ScanEventsTotal)
	}
}
