package metrics

import "github.com/prometheus/client_golang/prometheus"

// Rulepack and stream management metrics
var (
	RulepackActivations = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_rulepack_activations_total", Help: "Total rulepack activations"},
	)
	RulepackValidationFailures = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "ps_rulepack_validation_failures_total", Help: "Total rulepack validation failures"},
	)
	RuleCompilationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ps_rule_compilation_duration_seconds",
			Help:    "Time to compile rules from rulepacks",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"status"},
	)
)

func IncRulepackActivations() { 
	RulepackActivations.Inc() 
}

func IncRulepackValidationFailures() { 
	RulepackValidationFailures.Inc() 
}

func init() {
	prometheus.MustRegister(RulepackActivations, RulepackValidationFailures, RuleCompilationDuration)
}