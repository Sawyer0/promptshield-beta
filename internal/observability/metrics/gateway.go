package metrics

import (
    "os"
    "sort"
    "strconv"
    "strings"
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
        []string{"method", "status", "endpoint"},
    )
    RequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "ps_gateway_request_duration_seconds",
            Help:    "Duration of gateway HTTP requests by status and endpoint",
            Buckets: requestDurationBuckets,
        },
        []string{"method", "status", "endpoint"},
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
        prometheus.MustRegister(RequestsTotal, RequestDuration, RequestQueueDepth, CacheOperations, ResourceUtilization, TimeToFirstDecision)
    }
}

// Customizable histogram buckets for request duration.
// Override via PS_GATEWAY_REQ_BUCKETS (comma-separated seconds), e.g. "0.01,0.05,0.1,0.25,0.5,1,2.5,5,10".
var requestDurationBuckets = func() []float64 {
    if s := strings.TrimSpace(os.Getenv("PS_GATEWAY_REQ_BUCKETS")); s != "" {
        parts := strings.Split(s, ",")
        bucks := make([]float64, 0, len(parts))
        for _, p := range parts {
            if f, err := strconv.ParseFloat(strings.TrimSpace(p), 64); err == nil && f > 0 {
                bucks = append(bucks, f)
            }
        }
        if len(bucks) >= 2 {
            sort.Float64s(bucks)
            return bucks
        }
    }
    // Sensible default buckets from 5ms to 30s
    return []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.25, 0.35, 0.5, 0.75, 1.0, 1.5, 2.5, 5, 7.5, 10, 15, 30}
}()
