package rules

// RulePack models a user-authored YAML rule pack per v2 schema in plan/.
type RulePack struct {
	APIVersion  string            `yaml:"apiVersion"`
	Kind        string            `yaml:"kind"`
	Metadata    Metadata          `yaml:"metadata"`
	Rules       []Rule            `yaml:"rules"`
	Extends     []string          `yaml:"extends"`
	Imports     []string          `yaml:"imports"`
	Composition *Composition      `yaml:"composition"`
	Performance *Performance      `yaml:"performance"`
	Overrides   []Override        `yaml:"overrides"`
	Context     map[string]string `yaml:"context"`
	// SourcePath is the filesystem path the pack was loaded from (not serialized).
	SourcePath string `yaml:"-"`
}

type Metadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	Author      string   `yaml:"author"`
	License     string   `yaml:"license"`
	Homepage    string   `yaml:"homepage"`
	Repository  string   `yaml:"repository"`
	Tags        []string `yaml:"tags"`
}

type Rule struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Level    int    `yaml:"level"`    // 1=keyword, 2=regex, 3=semantic
	Severity string `yaml:"severity"` // INFO|WARNING|ERROR|CRITICAL
	Category string `yaml:"category"`
	Enabled  *bool  `yaml:"enabled"`
	Verifier string `yaml:"verifier"`

	Keywords []string  `yaml:"keywords"`
	Patterns []Pattern `yaml:"patterns"`
	Semantic *Semantic `yaml:"semantic"`
	Fallback *Fallback `yaml:"fallback"`

	Logic    string    `yaml:"logic"` // any|all|custom
	Options  Options   `yaml:"options"`
	Response *Response `yaml:"response"`
	Cache    *Cache    `yaml:"cache"`

	// Context gating (optional)
	When   *Condition `yaml:"when"`
	Unless *Condition `yaml:"unless"`

	// Optional per-rule timeout (e.g., "50ms")
	Timeout string `yaml:"timeout"`
}

type Options struct {
	CaseSensitive bool `yaml:"case_sensitive"`
	WholeWord     bool `yaml:"whole_word"`
}

type Pattern struct {
	Name     string   `yaml:"name"`
	Regex    string   `yaml:"regex"`
	Flags    []string `yaml:"flags"`
	Verifier string   `yaml:"verifier"`
}

// Condition gates rule execution using merged context (pack defaults overridden by CLI context).
// A rule matches the condition if for every key in Match, the runtime context
// contains at least one of the listed values (OR within a key, AND across keys).
type Condition struct {
	Match map[string][]string `yaml:"match"`
}

type Semantic struct {
	Model               string  `yaml:"model"`
	Temperature         float64 `yaml:"temperature"`
	MaxTokens           int     `yaml:"max_tokens"`
	AnalysisPrompt      string  `yaml:"analysis_prompt"`
	ConfidenceThreshold float64 `yaml:"confidence_threshold"`
	FallbackOnError     bool    `yaml:"fallback_on_error"`
}

type Fallback struct {
	Patterns []Pattern `yaml:"patterns"`
	Action   string    `yaml:"action"`
}

type Response struct {
	Action      string `yaml:"action"`
	Message     string `yaml:"message"`
	Replacement string `yaml:"replacement"`
}

type Cache struct {
	Enabled    bool   `yaml:"enabled"`
	TTL        string `yaml:"ttl"`
	MaxEntries int    `yaml:"max_entries"`
}

type Composition struct {
	Strategy string `yaml:"strategy"`
	Priority int    `yaml:"priority,omitempty"` // Higher values = higher priority (processed first)
}

type Performance struct {
	MaxLength        int    `yaml:"max_length"`
	Timeout          string `yaml:"timeout"`
	PerRuleTimeout   string `yaml:"per_rule_timeout"`
	TotalScanTimeout string `yaml:"total_scan_timeout"`
	Gate             struct {
		Enabled     bool `yaml:"enabled"`
		MinTokenLen int  `yaml:"min_token_len"`
	} `yaml:"gate"`
}

type Override struct {
	RuleID   string `yaml:"rule_id"`
	Severity string `yaml:"severity"`
	Enabled  *bool  `yaml:"enabled"`
}
