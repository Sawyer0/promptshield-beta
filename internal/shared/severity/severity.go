package severity

import "strings"

// MeetsThreshold returns true if sev is at or above the threshold.
// Accepted severities: INFO/LOW < WARNING < MEDIUM < HIGH < ERROR < CRITICAL
func MeetsThreshold(sev, threshold string) bool {
	order := map[string]int{
		"INFO": 1, "LOW": 1,
		"WARNING": 2,
		"MEDIUM": 3,
		"HIGH": 4,
		"ERROR": 5,
		"CRITICAL": 6,
	}
	s := order[strings.ToUpper(sev)]
	t := order[strings.ToUpper(threshold)]
	if t == 0 { // unknown threshold -> treat as no threshold
		return false
	}
	return s >= t
}
