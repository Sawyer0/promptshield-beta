package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    PolicyFlushEventsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ps_tool_policy_flush_events_total",
            Help: "Total number of tool policy cache flush events (published/received)",
        },
        []string{"role", "scope"}, // role: publisher|subscriber; scope: global|tenant
    )
    PolicyFlushLatencySeconds = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "ps_tool_policy_flush_latency_seconds",
            Help:    "Propagation latency from publish to subscriber receipt",
            Buckets: prometheus.DefBuckets,
        },
        []string{"scope"}, // scope: global|tenant
    )
    PolicyFlushSubscriberUp = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "ps_tool_policy_subscriber_up",
            Help: "Whether the tool policy flush subscriber is active (1) or not (0)",
        },
    )
    PolicyFlushLastReceiveUnixSeconds = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "ps_tool_policy_last_receive_unix_seconds",
            Help: "Unix timestamp of last received tool policy flush event",
        },
    )
)

func init() {
    if Enabled() {
        prometheus.MustRegister(PolicyFlushEventsTotal, PolicyFlushLatencySeconds, PolicyFlushSubscriberUp, PolicyFlushLastReceiveUnixSeconds)
    }
}
