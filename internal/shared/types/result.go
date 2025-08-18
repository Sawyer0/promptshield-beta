package types

import "time"

// ScanResult represents the result of a security scan
// Enhanced version of pkg/types.ScanResult with additional enforcement context
type ScanResult struct {
	Input      string             `json:"input"`
	Violations []PolicyViolation  `json:"violations"`
	Metrics    ScanMetrics        `json:"metrics"`
	ScanInfo   ScanInfo           `json:"scan_info"`
	Decision   EnforcementDecision `json:"decision"`
}

// ScanMetrics captures lightweight scan metrics for observability
// Enhanced version of pkg/types.Metrics
type ScanMetrics struct {
	BytesRead int64 `json:"bytes_read"`
	LinesRead int64 `json:"lines_read"`
	
	// Performance counters
	RegexAttempts    int64 `json:"regex_attempts,omitempty"`
	RegexSkipped     int64 `json:"regex_skipped,omitempty"`
	SemanticAttempts int64 `json:"semantic_attempts,omitempty"`
	SemanticSkipped  int64 `json:"semantic_skipped,omitempty"`
	
	// Cache efficiency
	CacheHits   int `json:"cache_hits,omitempty"`
	CacheMisses int `json:"cache_misses,omitempty"`
}

// ScanInfo captures comprehensive scan metadata
// Enhanced version of pkg/types.ScanInfo
type ScanInfo struct {
	// Summary statistics
	TotalViolations  int    `json:"total_violations"`
	ScanStatus       string `json:"scan_status"`        // "success", "timeout", "error", "partial"
	ScanDurationMs   int64  `json:"scan_duration_ms"`
	
	// Rule evaluation statistics
	RulesProcessed   int `json:"rules_processed"`
	RulesSkipped     int `json:"rules_skipped,omitempty"`
	RulesTimedOut    int `json:"rules_timed_out,omitempty"`
	RulesErrored     int `json:"rules_errored,omitempty"`
	
	// Performance breakdown by rule level
	Level1DurationMs int64 `json:"level1_duration_ms,omitempty"` // Keywords
	Level2DurationMs int64 `json:"level2_duration_ms,omitempty"` // Regex
	Level3DurationMs int64 `json:"level3_duration_ms,omitempty"` // Semantic/LLM
	
	// Decision metadata
	ShouldBlock       bool   `json:"should_block"`
	BlockReason       string `json:"block_reason,omitempty"`       // Rule ID or reason for blocking
	HighestSeverity   string `json:"highest_severity,omitempty"`   // Highest severity found
	TriggerRuleCount  int    `json:"trigger_rule_count"`           // Number of rules that triggered
	
	// Resource usage
	PeakMemoryBytes int64 `json:"peak_memory_bytes,omitempty"`
	CPUTimeMs       int64 `json:"cpu_time_ms,omitempty"`
}

// EnforcementDecision represents the result of policy enforcement
// From domain/models.go
type EnforcementDecision struct {
	Allow       bool                   `json:"allow"`
	Reason      string                 `json:"reason,omitempty"`
	Violations  []PolicyViolation      `json:"violations,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ProcessedAt time.Time              `json:"processed_at"`
	Latency     time.Duration          `json:"latency_ms"`
}

// ShouldBlock returns true if the scan result indicates the request should be blocked
func (sr *ScanResult) ShouldBlock() bool {
	return sr.ScanInfo.ShouldBlock || !sr.Decision.Allow
}

// HasViolations returns true if the scan found any violations
func (sr *ScanResult) HasViolations() bool {
	return len(sr.Violations) > 0
}

// HighestSeverityViolation returns the violation with the highest severity
func (sr *ScanResult) HighestSeverityViolation() *PolicyViolation {
	if len(sr.Violations) == 0 {
		return nil
	}
	
	highest := &sr.Violations[0]
	for i := 1; i < len(sr.Violations); i++ {
		if sr.compareSeverity(sr.Violations[i].Severity, highest.Severity) > 0 {
			highest = &sr.Violations[i]
		}
	}
	
	return highest
}

// compareSeverity compares two severity levels
// Returns: 1 if a > b, 0 if a == b, -1 if a < b
func (sr *ScanResult) compareSeverity(a, b string) int {
	severityOrder := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}
	
	aVal, aOk := severityOrder[a]
	bVal, bOk := severityOrder[b]
	
	if !aOk || !bOk {
		return 0 // Unknown severities are equal
	}
	
	if aVal > bVal {
		return 1
	} else if aVal < bVal {
		return -1
	}
	return 0
}