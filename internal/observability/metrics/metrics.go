package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Gateway Prometheus Metrics - All metrics for HTTP API and gRPC enforcer
var (
	// HTTP API metrics
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

	// gRPC ext_proc enforcer metrics
	ExtprocStreams = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ps_extproc_streams_total", Help: "Total gRPC ext_proc streams by decision"},
		[]string{"decision"},
	)
	ExtprocBytes = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_extproc_bytes_total", Help: "Total bytes observed in gRPC ext_proc streams"},
	)
	ExtprocStreamDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "ps_extproc_stream_duration_seconds", Help: "Duration of gRPC ext_proc streams", Buckets: prometheus.DefBuckets},
		[]string{"decision"},
	)
	ExtprocRuleHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ps_extproc_rule_hits_total", Help: "Rule hits counted by id and severity"},
		[]string{"rule_id", "severity"},
	)
	ExtprocRedactions = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_extproc_redactions_total", Help: "Total body chunk redactions applied"},
	)

	// Rulepack and stream management metrics
	RulepackActivations = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_rulepack_activations_total", Help: "Total rulepack activations"},
	)
	RulepackValidationFailures = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_rulepack_validation_failures_total", Help: "Total rulepack validation failures"},
	)
	StreamLagSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ps_stream_lag_seconds",
			Help: "Lag in seconds for message streams",
		},
		[]string{"stream"},
	)
	PendingCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ps_pending_messages",
			Help: "Number of pending messages in streams",
		},
		[]string{"stream"},
	)
	ConsumerRestartsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ps_consumer_restarts_total",
			Help: "Total consumer restarts",
		},
	)

	// Gateway operational metrics
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
	RuleCompilationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ps_rule_compilation_duration_seconds",
			Help:    "Time to compile rules from rulepacks",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"status"},
	)
	TimeToFirstDecision = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ps_time_to_first_decision_seconds",
			Help:    "Time from request start to first enforcement decision",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
		},
	)
)

// Helper functions for common metric operations
func IncRulepackActivations()        { RulepackActivations.Inc() }
func IncRulepackValidationFailures() { RulepackValidationFailures.Inc() }

// init registers all Gateway metrics with Prometheus
func init() {
	// HTTP API metrics
	prometheus.MustRegister(HTTPBytesTotal, ScanRequestDuration, ScanEventsTotal)
	
	// gRPC ext_proc enforcer metrics  
	prometheus.MustRegister(ExtprocStreams, ExtprocBytes, ExtprocStreamDuration, ExtprocRuleHits, ExtprocRedactions)
	
	// Rulepack and stream management metrics
	prometheus.MustRegister(RulepackActivations, RulepackValidationFailures)
	prometheus.MustRegister(StreamLagSeconds, PendingCount, ConsumerRestartsTotal)
	
	// Gateway operational metrics
	prometheus.MustRegister(RequestsTotal, RequestQueueDepth, CacheOperations)
	prometheus.MustRegister(ResourceUtilization, RuleCompilationDuration, TimeToFirstDecision)
}
