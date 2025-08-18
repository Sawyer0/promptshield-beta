package types

// PolicyViolation represents a policy violation found during scanning
// Consolidates PolicyViolation from proxy.go, domain.Violation, and pkg/types.Violation
type PolicyViolation struct {
	RuleID     string  `json:"rule_id"`
	Severity   string  `json:"severity"`
	Message    string  `json:"message"`
	Action     string  `json:"action"` // allow, deny, quarantine
	Confidence float64 `json:"confidence,omitempty"`
	
	// Additional fields from pkg/types for compatibility
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Category string `json:"category,omitempty"`
	
	// Response action fields from pkg/types
	ResponseAction      string `json:"response_action,omitempty"`
	ResponseMessage     string `json:"response_message,omitempty"`
	ResponseReplacement string `json:"response_replacement,omitempty"`
	
	// Timing information
	RuleTimeoutMs int64 `json:"rule_timeout_ms,omitempty"`
}

// ViolationSeverity represents violation severity levels
type ViolationSeverity string

const (
	SeverityLow      ViolationSeverity = "low"
	SeverityMedium   ViolationSeverity = "medium"
	SeverityHigh     ViolationSeverity = "high"
	SeverityCritical ViolationSeverity = "critical"
)

// ViolationAction represents what action to take on a violation
type ViolationAction string

const (
	ActionAllow      ViolationAction = "allow"
	ActionDeny       ViolationAction = "deny"
	ActionQuarantine ViolationAction = "quarantine"
	ActionRedact     ViolationAction = "redact"
)