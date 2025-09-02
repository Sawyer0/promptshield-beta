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
	Patterns    *Patterns         `yaml:"patterns" json:"patterns,omitempty"`
	Preset      *Preset           `yaml:"preset" json:"preset,omitempty"`
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
	Engine string `yaml:"engine" json:"engine,omitempty"`
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
	CombineMode  string  `yaml:"combine_mode" json:"combine_mode,omitempty"` // or|and|weighted (default or)
	WeightOmni   float64 `yaml:"weight_omni" json:"weight_omni,omitempty"`
	WeightCustom float64 `yaml:"weight_custom" json:"weight_custom,omitempty"`
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

// Patterns and Preset types for agent hardening (capability-based)
type Patterns struct {
	ActionSelector      *ActionSelector      `yaml:"action_selector" json:"action_selector,omitempty"`
	ContextMinimization *ContextMinimization `yaml:"context_minimization" json:"context_minimization,omitempty"`
	PlanThenExecute     *PlanThenExecute     `yaml:"plan_then_execute" json:"plan_then_execute,omitempty"`
	MapReduce           *MapReduce           `yaml:"map_reduce" json:"map_reduce,omitempty"`
	DualLLM             *DualLLM             `yaml:"dual_llm" json:"dual_llm,omitempty"`
}

type ActionSelector struct {
	Enabled            bool   `yaml:"enabled" json:"enabled"`
	Mode               string `yaml:"mode" json:"mode,omitempty"`
	AllowedToolQuery   string `yaml:"allowed_tool_query" json:"allowed_tool_query,omitempty"`
	PerActionTimeoutMs int    `yaml:"per_action_timeout_ms" json:"per_action_timeout_ms,omitempty"`
	PerActionRateLimit int    `yaml:"per_action_rate_limit" json:"per_action_rate_limit,omitempty"`
}

type ContextMinimization struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`
	StripPoint string   `yaml:"strip_point" json:"strip_point,omitempty"`
	Step       int      `yaml:"step" json:"step,omitempty"`
	MaskToken  string   `yaml:"mask_token" json:"mask_token,omitempty"`
	Retain     []string `yaml:"retain" json:"retain,omitempty"`
}

type PlanThenExecute struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	MaxSteps       int    `yaml:"max_steps" json:"max_steps,omitempty"`
	DriftPolicy    string `yaml:"drift_policy" json:"drift_policy,omitempty"`
	PlanHashHeader string `yaml:"plan_hash_header" json:"plan_hash_header,omitempty"`
	Signature      struct {
		Type   string `yaml:"type" json:"type,omitempty"`
		KeyRef string `yaml:"key_ref" json:"key_ref,omitempty"`
	} `yaml:"signature" json:"signature,omitempty"`
}

type MapReduce struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	MapUnit       string `yaml:"map_unit" json:"map_unit,omitempty"`
	MapOutput     string `yaml:"map_output" json:"map_output,omitempty"`
	TextMaxTokens int    `yaml:"text_max_tokens" json:"text_max_tokens,omitempty"`
	ReduceType    string `yaml:"reduce_type" json:"reduce_type,omitempty"`
}

type DualLLM struct {
	Enabled                  bool `yaml:"enabled" json:"enabled"`
	QuarantinedToolsDisabled bool `yaml:"quarantined_tools_disabled" json:"quarantined_tools_disabled,omitempty"`
	BridgeHandlesOnly        bool `yaml:"bridge_handles_only" json:"bridge_handles_only,omitempty"`
}

type Preset struct {
	PresetID     string         `yaml:"preset_id" json:"preset_id"`
	Mode         string         `yaml:"mode" json:"mode,omitempty"`
	ArgContracts []string       `yaml:"arg_contracts" json:"arg_contracts,omitempty"`
	RiskRules    map[string]any `yaml:"risk_rules" json:"risk_rules,omitempty"`
	Overrides    map[string]any `yaml:"overrides" json:"overrides,omitempty"`
}
