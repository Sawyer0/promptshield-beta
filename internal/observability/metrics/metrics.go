package metrics

import (
	"os"
)

// Registry initialization package for PromptShield metrics.
// Domain-specific metrics are defined in their respective subpackages:
// - enforcer.go: Enforcer-specific metrics
// - extproc.go: External processor (Envoy ext_proc) metrics
// - gateway.go: Gateway operational metrics
// - http.go: HTTP API metrics
// - llm.go: LLM provider and token usage metrics
// - rulepack.go: Rulepack management metrics
// - stream.go: Stream processing metrics

// Enabled returns true if metrics collection is enabled
// This is controlled by the PS_DISABLE_METRICS environment variable
func Enabled() bool {
	return os.Getenv("PS_DISABLE_METRICS") == ""
}

// init is called when the package is imported, ensuring all domain-specific
// metrics are registered with Prometheus via their respective init() functions.
func init() {
	// Metrics registration happens via init() functions in:
	// - enforcer.go
	// - extproc.go
	// - gateway.go
	// - http.go
	// - llm.go
	// - rulepack.go
	// - stream.go
	// Each init() function now checks metrics.Enabled() before registering
}
