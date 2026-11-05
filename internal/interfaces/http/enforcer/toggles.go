package enforcerhttp

import (
	"os"
	"strings"
)

// Optional performance toggles (env-driven)
func metricsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_DISABLE_METRICS")))
	return !(v == "1" || v == "true" || v == "yes")
}

func tracingEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_DISABLE_TRACING")))
	return !(v == "1" || v == "true" || v == "yes")
}

func fastMode() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_FAST")))
	return v == "1" || v == "true" || v == "yes"
}

// envBool returns true when the provided env var is set to 1, true, yes, or on (case-insensitive).
func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
