package severity

import "strings"

// MeetsThreshold returns true if sev is at or above the threshold.
// Accepted severities: INFO < WARNING < HIGH < ERROR < CRITICAL
func MeetsThreshold(sev, threshold string) bool {
	order := map[string]int{"INFO": 1, "WARNING": 2, "HIGH": 3, "ERROR": 4, "CRITICAL": 5}
	s := order[strings.ToUpper(sev)]
	t := order[strings.ToUpper(threshold)]
	if t == 0 { // unknown threshold -> treat as no threshold
		return false
	}
	return s >= t
}
