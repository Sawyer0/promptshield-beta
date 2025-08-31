package rules

// RulePack models a user-authored YAML rule pack per v2 schema in plan/.
type RulePack struct {
	APIVersion  string            `yaml:"apiVersion" json:"apiVersion"`
	Kind        string            `yaml:"kind" json:"kind"`
	Metadata    Metadata          `yaml:"metadata" json:"metadata"`
	Rules       []Rule            `yaml:"rules" json:"rules"`
	Extends     []string          `yaml:"extends" json:"extends,omitempty"`
	Imports     []string          `yaml:"imports" json:"imports,omitempty"`
	Composition *Composition      `yaml:"composition" json:"composition,omitempty"`
	Performance *Performance      `yaml:"performance" json:"performance,omitempty"`
	Overrides   []Override        `yaml:"overrides" json:"overrides,omitempty"`
	Context     map[string]string `yaml:"context" json:"context,omitempty"`
	// SourcePath is the filesystem path the pack was loaded from (not serialized).
	SourcePath string `yaml:"-" json:"-"`
}

type Metadata struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version,omitempty"`
	Description string   `yaml:"description" json:"description,omitempty"`
	Author      string   `yaml:"author" json:"author,omitempty"`
	Authors     []string `yaml:"authors" json:"authors,omitempty"`
	License     string   `yaml:"license" json:"license,omitempty"`
	Homepage    string   `yaml:"homepage" json:"homepage,omitempty"`
	Repository  string   `yaml:"repository" json:"repository,omitempty"`
	Tags        []string `yaml:"tags" json:"tags,omitempty"`
}

type Rule struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name,omitempty"`
	Description string `yaml:"description" json:"description,omitempty"`
	Level       int    `yaml:"level" json:"level"`                 // 1=keyword, 2=regex, 3=semantic
	Severity    string `yaml:"severity" json:"severity,omitempty"` // INFO|WARNING|ERROR|CRITICAL
	Category    string `yaml:"category" json:"category,omitempty"`
	Enabled     *bool  `yaml:"enabled" json:"enabled,omitempty"`
	Verifier    string `yaml:"verifier" json:"verifier,omitempty"`

	Keywords []string  `yaml:"keywords" json:"keywords,omitempty"`
	Patterns []Pattern `yaml:"patterns" json:"patterns,omitempty"`
	Semantic *Semantic `yaml:"semantic" json:"semantic,omitempty"`
	Fallback *Fallback `yaml:"fallback" json:"fallback,omitempty"`

	Logic    string    `yaml:"logic" json:"logic,omitempty"` // any|all|custom
	Options  Options   `yaml:"options" json:"options,omitempty"`
	Response *Response `yaml:"response" json:"response,omitempty"`
	Cache    *Cache    `yaml:"cache" json:"cache,omitempty"`

	// Context gating (optional)
	When   *Condition `yaml:"when" json:"when,omitempty"`
	Unless *Condition `yaml:"unless" json:"unless,omitempty"`

	// Optional per-rule timeout (e.g., "50ms")
	Timeout string `yaml:"timeout" json:"timeout,omitempty"`
}

type Options struct {
	CaseSensitive bool `yaml:"case_sensitive" json:"case_sensitive,omitempty"`
	WholeWord     bool `yaml:"whole_word" json:"whole_word,omitempty"`
}

type Pattern struct {
	Name     string   `yaml:"name" json:"name,omitempty"`
	Regex    string   `yaml:"regex" json:"regex"`
	Flags    []string `yaml:"flags" json:"flags,omitempty"`
	Verifier string   `yaml:"verifier" json:"verifier,omitempty"`
}

// Condition gates rule execution using merged context (pack defaults overridden by CLI context).
// A rule matches the condition if for every key in Match, the runtime context
// contains at least one of the listed values (OR within a key, AND across keys).
type Condition struct {
	Match map[string][]string `yaml:"match" json:"match"`
}

type Semantic struct {
    // Engine selects the L3 engine: "omni" | "custom" | "custom+omni" (default "omni" if unset)
    Engine              string  `yaml:"engine" json:"engine,omitempty"`
    // Custom/Omni shared
    Model               string  `yaml:"model" json:"model,omitempty"`
    Temperature         float64 `yaml:"temperature" json:"temperature,omitempty"`
    MaxTokens           int     `yaml:"max_tokens" json:"max_tokens,omitempty"`
    AnalysisPrompt      string  `yaml:"analysis_prompt" json:"analysis_prompt,omitempty"`
    ConfidenceThreshold float64 `yaml:"confidence_threshold" json:"confidence_threshold,omitempty"`
    FallbackOnError     bool    `yaml:"fallback_on_error" json:"fallback_on_error,omitempty"`
    // AllowedCategories restricts evaluation to these moderation categories.
	// Examples: "violence", "violence/graphic", "self-harm/instructions", "illicit".
	// When empty, all categories returned by the provider are considered.
	AllowedCategories []string `yaml:"categories" json:"categories,omitempty"`
    // Inputs specifies which inputs are expected by the rule: "text", "image".
    // This is advisory for UIs; the backend relies on provider behavior.
    Inputs []string `yaml:"inputs" json:"inputs,omitempty"`
    // Provider profile id for BYOK custom engines
    ProviderProfile string `yaml:"provider_profile" json:"provider_profile,omitempty"`
    // Combine semantics for custom+omni
    CombineMode   string   `yaml:"combine_mode" json:"combine_mode,omitempty"` // or|and|weighted (default or)
    WeightOmni    float64  `yaml:"weight_omni" json:"weight_omni,omitempty"`
    WeightCustom  float64  `yaml:"weight_custom" json:"weight_custom,omitempty"`
}

type Fallback struct {
	Patterns []Pattern `yaml:"patterns" json:"patterns,omitempty"`
	Action   string    `yaml:"action" json:"action,omitempty"`
}

type Response struct {
	Action      string `yaml:"action" json:"action,omitempty"`
	Message     string `yaml:"message" json:"message,omitempty"`
	Replacement string `yaml:"replacement" json:"replacement,omitempty"`
}

type Cache struct {
	Enabled    bool   `yaml:"enabled" json:"enabled,omitempty"`
	TTL        string `yaml:"ttl" json:"ttl,omitempty"`
	MaxEntries int    `yaml:"max_entries" json:"max_entries,omitempty"`
}

type Composition struct {
	Strategy string `yaml:"strategy" json:"strategy,omitempty"`
	Priority int    `yaml:"priority,omitempty" json:"priority,omitempty"` // Higher values = higher priority (processed first)
}

type Performance struct {
	MaxLength        int    `yaml:"max_length" json:"max_length,omitempty"`
	Timeout          string `yaml:"timeout" json:"timeout,omitempty"`
	PerRuleTimeout   string `yaml:"per_rule_timeout" json:"per_rule_timeout,omitempty"`
	TotalScanTimeout string `yaml:"total_scan_timeout" json:"total_scan_timeout,omitempty"`
	Gate             struct {
		Enabled     bool `yaml:"enabled" json:"enabled,omitempty"`
		MinTokenLen int  `yaml:"min_token_len" json:"min_token_len,omitempty"`
	} `yaml:"gate" json:"gate,omitempty"`
}

type Override struct {
	RuleID   string `yaml:"rule_id" json:"rule_id"`
	Severity string `yaml:"severity" json:"severity,omitempty"`
	Enabled  *bool  `yaml:"enabled" json:"enabled,omitempty"`
}
