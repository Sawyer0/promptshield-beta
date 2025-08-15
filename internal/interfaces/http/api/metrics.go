package api

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpBytesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_http_bytes_total",
			Help: "Total HTTP bytes in/out by path",
		},
		[]string{"direction", "path"},
	)

	scanRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ps_scan_request_duration_seconds",
			Help:    "Duration of /v1/scan requests by mode",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"mode"},
	)

	scanEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_scan_events_total",
			Help: "Total NDJSON decision events emitted per route",
		},
		[]string{"route"},
	)

	registerMetricsOnce sync.Once
)

func init() {
	registerMetricsOnce.Do(func() {
		prometheus.MustRegister(httpBytesTotal, scanRequestDuration, scanEventsTotal)
	})
}
