package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Enforcer-specific metrics
var (
	EnforcerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_enforcer_requests_total",
			Help: "Total HTTP requests to enforcer",
		},
		[]string{"path", "code"},
	)
	EnforcerDecisions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_enforcer_decisions_total",
			Help: "Total decisions made by enforcer",
		},
		[]string{"decision"},
	)
	EnforcerRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ps_enforcer_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "decision"},
	)
	PolicyBypass = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_policy_bypass_total",
			Help: "Total requests served in policy bypass mode",
		},
		[]string{"reason"},
	)
)

func init() {
	// Only register if metrics are enabled
	if Enabled() {
		prometheus.MustRegister(EnforcerRequests, EnforcerDecisions, EnforcerRequestDuration, PolicyBypass)
	}
}
