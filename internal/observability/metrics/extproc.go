package metrics

import "github.com/prometheus/client_golang/prometheus"

// gRPC ext_proc enforcer metrics
var (
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
)

func init() {
	if Enabled() {
		prometheus.MustRegister(ExtprocStreams, ExtprocBytes, ExtprocStreamDuration, ExtprocRuleHits, ExtprocRedactions)
	}
}
