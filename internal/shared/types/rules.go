package types

// RulePack models a user-authored YAML rule pack per v2 schema
// From internal/rules/types.go
type RulePack struct {
	APIVersion  string            `yaml:"apiVersion" json:"apiVersion"`
	Kind        string            `yaml:"kind" json:"kind"`
	Metadata    RulePackMetadata  `yaml:"metadata" json:"metadata"`
	Rules       []Rule            `yaml:"rules" json:"rules"`
	Extends     []string          `yaml:"extends" json:"extends,omitempty"`
	Imports     []string          `yaml:"imports" json:"imports,omitempty"`
	Composition *Composition      `yaml:"composition" json:"composition,omitempty"`
	Performance *Performance      `yaml:"performance" json:"performance,omitempty"`
	Overrides   []Override        `yaml:"overrides" json:"overrides,omitempty"`
	Context     map[string]string `yaml:"context" json:"context,omitempty"`
	// SourcePath is the filesystem path the pack was loaded from (not serialized)
	SourcePath string `yaml:"-" json:"-"`
}

// RulePackMetadata contains metadata about a rule pack
type RulePackMetadata struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	Description string   `yaml:"description" json:"description,omitempty"`
	Author      string   `yaml:"author" json:"author,omitempty"`
	License     string   `yaml:"license" json:"license,omitempty"`
	Homepage    string   `yaml:"homepage" json:"homepage,omitempty"`
	Repository  string   `yaml:"repository" json:"repository,omitempty"`
	Tags        []string `yaml:"tags" json:"tags,omitempty"`
}

// Rule represents a single scanning rule
type Rule struct {
	ID       string `yaml:"id" json:"id"`
	Name     string `yaml:"name" json:"name"`
	Level    int    `yaml:"level" json:"level"`       // 1=keyword, 2=regex, 3=semantic
	Severity string `yaml:"severity" json:"severity"` // INFO|WARNING|ERROR|CRITICAL
	Category string `yaml:"category" json:"category,omitempty"`
	Enabled  *bool  `yaml:"enabled" json:"enabled,omitempty"`
	Verifier string `yaml:"verifier" json:"verifier,omitempty"`

	// Rule matchers
	Keywords []string    `yaml:"keywords" json:"keywords,omitempty"`
	Patterns []Pattern   `yaml:"patterns" json:"patterns,omitempty"`
	Semantic *Semantic   `yaml:"semantic" json:"semantic,omitempty"`
	Fallback *Fallback   `yaml:"fallback" json:"fallback,omitempty"`

	// Rule behavior
	Logic    string        `yaml:"logic" json:"logic,omitempty"`     // any|all|custom
	Options  RuleOptions   `yaml:"options" json:"options,omitempty"`
	Response *RuleResponse `yaml:"response" json:"response,omitempty"`
	Cache    *RuleCache    `yaml:"cache" json:"cache,omitempty"`

	// Context gating (optional)
	When   *Condition `yaml:"when" json:"when,omitempty"`
	Unless *Condition `yaml:"unless" json:"unless,omitempty"`

	// Optional per-rule timeout (e.g., "50ms")
	Timeout string `yaml:"timeout" json:"timeout,omitempty"`
}

// RuleOptions configures rule matching behavior
type RuleOptions struct {
	CaseSensitive bool `yaml:"case_sensitive" json:"case_sensitive,omitempty"`
	WholeWord     bool `yaml:"whole_word" json:"whole_word,omitempty"`
}

// Pattern represents a regex pattern for Level 2 rules
type Pattern struct {
	Name     string   `yaml:"name" json:"name"`
	Regex    string   `yaml:"regex" json:"regex"`
	Flags    []string `yaml:"flags" json:"flags,omitempty"`
	Verifier string   `yaml:"verifier" json:"verifier,omitempty"`
}

// Condition gates rule execution using merged context
// A rule matches the condition if for every key in Match, the runtime context
// contains at least one of the listed values (OR within a key, AND across keys)
type Condition struct {
	Match map[string][]string `yaml:"match" json:"match"`
}

// Semantic configuration for Level 3 rules
type Semantic struct {
	Model               string  `yaml:"model" json:"model"`
	Temperature         float64 `yaml:"temperature" json:"temperature,omitempty"`
	MaxTokens           int     `yaml:"max_tokens" json:"max_tokens,omitempty"`
	AnalysisPrompt      string  `yaml:"analysis_prompt" json:"analysis_prompt"`
	ConfidenceThreshold float64 `yaml:"confidence_threshold" json:"confidence_threshold,omitempty"`
	FallbackOnError     bool    `yaml:"fallback_on_error" json:"fallback_on_error,omitempty"`
}

// Fallback defines fallback behavior when semantic analysis fails
type Fallback struct {
	Patterns []Pattern `yaml:"patterns" json:"patterns"`
	Action   string    `yaml:"action" json:"action"`
}

// RuleResponse defines how to respond when a rule matches
type RuleResponse struct {
	Action      string `yaml:"action" json:"action"`                           // allow, deny, quarantine, redact
	Message     string `yaml:"message" json:"message,omitempty"`
	Replacement string `yaml:"replacement" json:"replacement,omitempty"`       // For redaction
}

// RuleCache configures caching for rule results
type RuleCache struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	TTL        string `yaml:"ttl" json:"ttl,omitempty"`
	MaxEntries int    `yaml:"max_entries" json:"max_entries,omitempty"`
}

// Composition defines how rules are composed together
type Composition struct {
	Strategy string `yaml:"strategy" json:"strategy"`                        // all_matches, first_match, priority_order
	Priority int    `yaml:"priority,omitempty" json:"priority,omitempty"`    // Higher values = higher priority
}

// Performance configures performance limits and timeouts
type Performance struct {
	MaxLength        int    `yaml:"max_length" json:"max_length,omitempty"`
	Timeout          string `yaml:"timeout" json:"timeout,omitempty"`
	PerRuleTimeout   string `yaml:"per_rule_timeout" json:"per_rule_timeout,omitempty"`
	TotalScanTimeout string `yaml:"total_scan_timeout" json:"total_scan_timeout,omitempty"`
	Gate             struct {
		Enabled     bool `yaml:"enabled" json:"enabled"`
		MinTokenLen int  `yaml:"min_token_len" json:"min_token_len,omitempty"`
	} `yaml:"gate" json:"gate,omitempty"`
}

// Override allows overriding specific rule properties
type Override struct {
	RuleID   string `yaml:"rule_id" json:"rule_id"`
	Severity string `yaml:"severity" json:"severity,omitempty"`
	Enabled  *bool  `yaml:"enabled" json:"enabled,omitempty"`
}

// RulePackInfo represents metadata about a rulepack (from contracts/repository.go)
type RulePackInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	RuleCount   int    `json:"rule_count"`
	LoadedAt    string `json:"loaded_at,omitempty"`
}

// CompiledRule represents a rule that has been compiled and optimized
type CompiledRule struct {
	Original     Rule                   `json:"original"`
	ID           string                 `json:"id"`
	Level        int                    `json:"level"`
	Severity     Severity               `json:"severity"`
	CompiledAt   string                 `json:"compiled_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RuleEvaluationResult represents the result of evaluating a single rule
type RuleEvaluationResult struct {
	RuleID      string    `json:"rule_id"`
	Matched     bool      `json:"matched"`
	Confidence  float64   `json:"confidence,omitempty"`
	Evidence    string    `json:"evidence,omitempty"`
	Location    string    `json:"location,omitempty"`
	LatencyMs   int64     `json:"latency_ms"`
	CacheHit    bool      `json:"cache_hit,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// ScanContext provides context for rule evaluation
type ScanContext struct {
	TenantID    string                 `json:"tenant_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Provider    Provider               `json:"provider,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Endpoint    string                 `json:"endpoint,omitempty"`
	ContentType string                 `json:"content_type,omitempty"`
	UserContext map[string]interface{} `json:"user_context,omitempty"`
}