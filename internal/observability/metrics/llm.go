package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	timeutil "github.com/promptshield/promptshield/internal/util/time"
)

// LLM Token Usage Metrics for billing/observability
var (
	TokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_llm_tokens_total",
			Help: "Total LLM tokens consumed by provider, model, and type",
		},
		[]string{"provider", "model", "token_type", "tenant"},
	)
	LLMRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_llm_requests_total",
			Help: "Total LLM requests by provider, model, and decision",
		},
		[]string{"provider", "model", "decision", "tenant"},
	)
	LLMLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ps_llm_request_duration_seconds",
			Help:    "LLM request latency by provider and model",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
		},
		[]string{"provider", "model"},
	)
	LLMErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_llm_errors_total",
			Help: "Total LLM provider errors by provider, error type, and retryable status",
		},
		[]string{"provider", "error_code", "retryable"},
	)
	LLMRetries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_llm_retries_total",
			Help: "Total LLM request retries by provider and attempt number",
		},
		[]string{"provider", "attempt"},
	)
	LLMTimeouts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_llm_timeouts_total",
			Help: "Total LLM request timeouts by provider and model",
		},
		[]string{"provider", "model"},
	)
	LLMRetrySuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ps_llm_retry_success_total",
			Help: "Total successful LLM requests after retries by provider and final attempt",
		},
		[]string{"provider", "model", "final_attempt"},
	)
)

func RecordLLMLatency(provider, model string, duration time.Duration) {
	timeutil.ObserveDuration(LLMLatency.WithLabelValues(provider, model), duration)
}

func init() {
	prometheus.MustRegister(TokensTotal, LLMRequestsTotal, LLMLatency, LLMErrors, LLMRetries, LLMTimeouts, LLMRetrySuccess)
}