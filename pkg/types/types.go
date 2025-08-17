package types

// Violation represents a single rule violation found during scanning.
type Violation struct {
	RuleID   string `json:"rule_id"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	// Optional time budget applied to this rule during evaluation, in milliseconds
	RuleTimeoutMs int64 `json:"rule_timeout_ms,omitempty"`
	// Optional metadata from rule definition
	Category            string `json:"category,omitempty"`
	ResponseAction      string `json:"response_action,omitempty"`
	ResponseMessage     string `json:"response_message,omitempty"`
	ResponseReplacement string `json:"response_replacement,omitempty"`
}

// Metrics captures lightweight scan metrics for observability.
type Metrics struct {
	BytesRead int64 `json:"bytes_read"`
	LinesRead int64 `json:"lines_read"`
	// Optional performance counters (omitted when zero)
	RegexAttempts    int64 `json:"regex_attempts,omitempty"`
	RegexSkipped     int64 `json:"regex_skipped,omitempty"`
	SemanticAttempts int64 `json:"semantic_attempts,omitempty"`
	SemanticSkipped  int64 `json:"semantic_skipped,omitempty"`
}

// ScanInfo captures comprehensive scan metadata for security gateways.
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
	
	// Cache efficiency
	CacheHits   int `json:"cache_hits,omitempty"`
	CacheMisses int `json:"cache_misses,omitempty"`
	
	// Decision metadata
	ShouldBlock       bool   `json:"should_block"`
	BlockReason       string `json:"block_reason,omitempty"`       // Rule ID or reason for blocking
	HighestSeverity   string `json:"highest_severity,omitempty"`   // Highest severity found
	TriggerRuleCount  int    `json:"trigger_rule_count"`           // Number of rules that triggered
	
	// Resource usage
	PeakMemoryBytes int64 `json:"peak_memory_bytes,omitempty"`
	CPUTimeMs       int64 `json:"cpu_time_ms,omitempty"`
}

// ScanResult is the output of a scan for a single input source.
type ScanResult struct {
	Input      string      `json:"input"`
	Violations []Violation `json:"violations"`
	Metrics    Metrics     `json:"metrics"`
	ScanInfo   ScanInfo    `json:"scan_info"`
	DurationMs int64       `json:"duration_ms,omitempty"` // Deprecated: use ScanInfo.ScanDurationMs
}

// ShouldBlock returns true if the scan result indicates the request should be blocked.
// This is a convenience method that checks the ScanInfo.ShouldBlock field.
func (sr *ScanResult) ShouldBlock() bool {
	return sr.ScanInfo.ShouldBlock
}

// Report is a higher-level aggregation used by some report renderers and tests.
// It is not currently used by the core scanning pipeline but remains for
// compatibility with streaming NDJSON tests that build aggregated reports.
type Report struct {
	FileReports []FileReport `json:"file_reports"`
	Summary     Summary      `json:"summary"`
}

// FileReport groups violations for a single file path.
type FileReport struct {
	Path       string      `json:"path"`
	Violations []Violation `json:"violations"`
}

// Summary provides simple aggregate counts for a scan run.
type Summary struct {
	FilesScanned   int `json:"files_scanned"`
	ViolationCount int `json:"violation_count"`
}
