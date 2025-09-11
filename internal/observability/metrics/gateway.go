package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	timeutil "github.com/promptshield/promptshield/internal/util/time"
)

// Gateway operational metrics
var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_gateway_requests_total",
			Help: "Total gateway requests by status",
		},
		[]string{"status", "endpoint"},
	)
	RequestQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ps_gateway_queue_depth",
			Help: "Current queue depth for async processing",
		},
		[]string{"queue_type"},
	)
	CacheOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_gateway_cache_operations_total",
			Help: "Cache hit/miss operations",
		},
		[]string{"operation", "cache_type"},
	)
	ResourceUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ps_gateway_resource_utilization",
			Help: "Resource utilization percentages",
		},
		[]string{"resource_type"},
	)
	TimeToFirstDecision = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ps_time_to_first_decision_seconds",
			Help:    "Time from request start to first enforcement decision",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
		},
	)
)

func RecordDuration(histogram prometheus.Observer, duration time.Duration) {
	timeutil.ObserveDuration(histogram, duration)
}

func init() {
	if Enabled() {
		prometheus.MustRegister(RequestsTotal, RequestQueueDepth, CacheOperations, ResourceUtilization, TimeToFirstDecision)
	}
}
