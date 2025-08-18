package metrics

import "github.com/prometheus/client_golang/prometheus"

// Stream management metrics
var (
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
)

func init() {
	prometheus.MustRegister(StreamLagSeconds, PendingCount, ConsumerRestartsTotal)
}