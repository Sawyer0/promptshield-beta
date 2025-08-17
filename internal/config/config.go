package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the typed effective configuration used by the CLI and services.
// Keys map 1:1 with Viper keys to keep precedence simple.
type Config struct {
	OutputFormat string `yaml:"output_format" json:"output_format"`
	Workers      int    `yaml:"workers" json:"workers"`
	Debug        bool   `yaml:"debug" json:"debug"`
	Color        *bool  `yaml:"color,omitempty" json:"color,omitempty"`
	AuditFile    string `yaml:"audit_file" json:"audit_file"`
	Redaction    struct {
		Enabled bool `yaml:"enabled" json:"enabled"`
	} `yaml:"redaction" json:"redaction"`

	RulepackPath string `yaml:"rulepack" json:"rulepack"`
	MetricsFile  string `yaml:"metrics_file" json:"metrics_file"`
	TraceFile    string `yaml:"trace_file" json:"trace_file"`
	FailOn       string `yaml:"fail_on" json:"fail_on"`

	// Enforcer-specific settings
	EnforcerMode             string `yaml:"enforcer_mode" json:"enforcer_mode"`
	RequireRulepackAtStartup bool   `yaml:"require_rulepack_at_startup" json:"require_rulepack_at_startup"`

	Composition struct {
		Strategy string `yaml:"strategy" json:"strategy"`
	} `yaml:"composition" json:"composition"`

	Performance struct {
		MaxLength        int    `yaml:"max_length" json:"max_length"`
		MaxFileSizeBytes int64  `yaml:"max_file_size" json:"max_file_size"`
		MaxPatternLength int    `yaml:"max_pattern_length" json:"max_pattern_length"`
		Timeout          string `yaml:"timeout" json:"timeout"`
		PerRuleTimeout   string `yaml:"per_rule_timeout" json:"per_rule_timeout"`
		TotalScanTimeout string `yaml:"total_scan_timeout" json:"total_scan_timeout"`
		CaseSensitive    bool   `yaml:"case_sensitive" json:"case_sensitive"`
		WholeWord        bool   `yaml:"whole_word" json:"whole_word"`
		BufferBytes      int    `yaml:"buffer_bytes" json:"buffer_bytes"`
		ChunkOverlap     int    `yaml:"chunk_overlap" json:"chunk_overlap"`
		Gate             struct {
			Enabled     bool `yaml:"enabled" json:"enabled"`
			MinTokenLen int  `yaml:"min_token_len" json:"min_token_len"`
		} `yaml:"gate" json:"gate"`
	} `yaml:"performance" json:"performance"`

	Telemetry struct {
		Enabled  bool    `yaml:"enabled" json:"enabled"`
		Endpoint string  `yaml:"endpoint" json:"endpoint"`
		File     string  `yaml:"file" json:"file"`
		Sample   float64 `yaml:"sample" json:"sample"`
	} `yaml:"telemetry" json:"telemetry"`
}

// Defaults provides baseline values for the CLI when nothing is provided.
func Defaults() Config {
	return Config{
		OutputFormat:             "stylish",
		Workers:                  0,
		Debug:                    false,
		EnforcerMode:             "observe",
		RequireRulepackAtStartup: false,
		Performance: struct {
			MaxLength        int    `yaml:"max_length" json:"max_length"`
			MaxFileSizeBytes int64  `yaml:"max_file_size" json:"max_file_size"`
			MaxPatternLength int    `yaml:"max_pattern_length" json:"max_pattern_length"`
			Timeout          string `yaml:"timeout" json:"timeout"`
			PerRuleTimeout   string `yaml:"per_rule_timeout" json:"per_rule_timeout"`
			TotalScanTimeout string `yaml:"total_scan_timeout" json:"total_scan_timeout"`
			CaseSensitive    bool   `yaml:"case_sensitive" json:"case_sensitive"`
			WholeWord        bool   `yaml:"whole_word" json:"whole_word"`
			BufferBytes      int    `yaml:"buffer_bytes" json:"buffer_bytes"`
			ChunkOverlap     int    `yaml:"chunk_overlap" json:"chunk_overlap"`
			Gate             struct {
				Enabled     bool `yaml:"enabled" json:"enabled"`
				MinTokenLen int  `yaml:"min_token_len" json:"min_token_len"`
			} `yaml:"gate" json:"gate"`
		}{
			Gate: struct {
				Enabled     bool `yaml:"enabled" json:"enabled"`
				MinTokenLen int  `yaml:"min_token_len" json:"min_token_len"`
			}{Enabled: true, MinTokenLen: 3},
		},
		Telemetry: struct {
			Enabled  bool    `yaml:"enabled" json:"enabled"`
			Endpoint string  `yaml:"endpoint" json:"endpoint"`
			File     string  `yaml:"file" json:"file"`
			Sample   float64 `yaml:"sample" json:"sample"`
		}{Enabled: false, Endpoint: "", File: "", Sample: 1.0},
	}
}

// Validate checks the configuration for logical and schema errors.
func Validate(cfg Config) []error {
	var errs []error
	switch strings.ToLower(cfg.OutputFormat) {
	case "", "stylish", "json", "github", "ndjson":
	default:
		errs = append(errs, fmt.Errorf("invalid output_format %q (valid: stylish,json,github,ndjson)", cfg.OutputFormat))
	}
	if cfg.Workers < 0 {
		errs = append(errs, fmt.Errorf("workers must be >= 0"))
	}
	if cfg.Performance.MaxFileSizeBytes < 0 {
		errs = append(errs, fmt.Errorf("performance.max_file_size must be >= 0"))
	}
	if cfg.Performance.MaxPatternLength < 0 {
		errs = append(errs, fmt.Errorf("performance.max_pattern_length must be >= 0"))
	}
	if cfg.FailOn != "" {
		sev := strings.ToUpper(cfg.FailOn)
		switch sev {
		case "INFO", "WARNING", "HIGH", "ERROR", "CRITICAL":
		default:
			errs = append(errs, fmt.Errorf("invalid fail_on %q (valid: INFO,WARNING,HIGH,ERROR,CRITICAL)", cfg.FailOn))
		}
	}
	if cfg.Composition.Strategy != "" {
		switch strings.ToLower(cfg.Composition.Strategy) {
		case "first_match", "priority_order":
		default:
			errs = append(errs, fmt.Errorf("invalid composition.strategy %q (valid: first_match, priority_order)", cfg.Composition.Strategy))
		}
	}
	if cfg.Telemetry.Sample < 0 || cfg.Telemetry.Sample > 1 {
		errs = append(errs, fmt.Errorf("telemetry.sample must be between 0 and 1"))
	}
	if err := ValidateEnforcementMode(strings.ToLower(cfg.EnforcerMode)); err != nil {
		errs = append(errs, err)
	}
	// Duration fields validated by consumers; keep as strings here
	return errs
}

// UnknownKeyError describes an unknown key found in a config file.
type UnknownKeyError struct {
	Path    string
	Suggest string
}

func (e UnknownKeyError) Error() string {
	// Normalize any YAML snippet style (e.g., "composition:\n  invalid") into dot notation
	p := e.Path
	// Replace common representations of nested keys with dot notation
	// 1) literal backslash-n sequences
	p = strings.ReplaceAll(p, ":\\n", ".")
	p = strings.ReplaceAll(p, ":\\r\\n", ".")
	// 2) real newlines (LF/CRLF)
	reNL := regexp.MustCompile(`:[\r\n]+\s*`)
	p = reNL.ReplaceAllString(p, ".")
	// 3) colon + spaces only
	reSpaces := regexp.MustCompile(`:\s+`)
	p = reSpaces.ReplaceAllString(p, ".")
	if e.Suggest != "" {
		return fmt.Sprintf("unknown config key: %s (did you mean %s?)", p, e.Suggest)
	}
	return fmt.Sprintf("unknown config key: %s", p)
}

// CheckUnknownKeys validates that a YAML config only contains allowed keys.
// It returns the first error encountered to keep messages actionable.
func CheckUnknownKeys(yamlBytes []byte) error {
	var raw map[string]any
	if err := yaml.Unmarshal(yamlBytes, &raw); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}
	allowed := map[string]any{
		"output_format": true,
		"workers":       true,
		"debug":         true,
		"color":         true,
		"audit_file":    true,
		"redaction": map[string]any{
			"enabled": true,
		},
		"rulepack":                    true,
		"metrics_file":                true,
		"trace_file":                  true,
		"fail_on":                     true,
		"enforcer_mode":               true,
		"require_rulepack_at_startup": true,
		"composition": map[string]any{
			"strategy": true,
		},
		"performance": map[string]any{
			"max_length":         true,
			"max_file_size":      true,
			"max_pattern_length": true,
			"timeout":            true,
			"per_rule_timeout":   true,
			"total_scan_timeout": true,
			"case_sensitive":     true,
			"whole_word":         true,
			"buffer_bytes":       true,
			"chunk_overlap":      true,
			"gate": map[string]any{
				"enabled":       true,
				"min_token_len": true,
			},
		},
		"telemetry": map[string]any{
			"enabled":  true,
			"endpoint": true,
			"file":     true,
			"sample":   true,
		},
	}
	return walkUnknown("", raw, allowed)
}

func walkUnknown(prefix string, cur any, allowed any) error {
	switch c := cur.(type) {
	case map[string]any:
		allowedMap, _ := allowed.(map[string]any)
		for k, v := range c {
			var p string
			if prefix == "" {
				p = k
			} else {
				p = prefix + "." + k
			}
			a, ok := allowedMap[k]
			if !ok {
				// Render nested context using dot-notation when parent key is known but child is not
				// Suggest nearest allowed key at this level
				suggest := nearestKey(k, allowedMap)
				return UnknownKeyError{Path: p, Suggest: suggest}
			}
			if err := walkUnknown(p, v, a); err != nil {
				return err
			}
		}
		return nil
	default:
		// Scalars/arrays are fine anywhere an allowed key is present
		return nil
	}
}

// nearestKey returns the closest key from allowedMap to the provided key using Levenshtein distance.
// Returns empty string when no reasonable suggestion is found.
func nearestKey(key string, allowedMap map[string]any) string {
	best := ""
	bestDist := 1 << 30
	for cand := range allowedMap {
		d := levenshtein(key, cand)
		if d < bestDist {
			bestDist = d
			best = cand
		}
	}
	// Heuristic: only suggest when reasonably close
	if bestDist <= 3 {
		return best
	}
	return ""
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	da := make([]int, len(b)+1)
	db := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		da[j] = j
	}
	for i := 1; i <= len(a); i++ {
		db[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := da[j] + 1
			ins := db[j-1] + 1
			sub := da[j-1] + cost
			db[j] = min3(del, ins, sub)
		}
		copy(da, db)
	}
	return da[len(b)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// ReadEffective loads the effective config by layering defaults with optional YAML file.
// The caller is responsible for applying flag/env overrides beforehand if they bypass Viper.
func ReadEffective(_ context.Context, configFileUsed string, get func(key string) any) Config {
	eff := Defaults()
	if v := get("output_format"); v != nil {
		eff.OutputFormat = toString(v, eff.OutputFormat)
	}
	if v := get("workers"); v != nil {
		eff.Workers = toInt(v, eff.Workers)
	}
	if v := get("debug"); v != nil {
		eff.Debug = toBool(v, eff.Debug)
	}
	if v := get("color"); v != nil {
		b := toBool(v, false)
		eff.Color = &b
	}
	if v := get("rulepack"); v != nil {
		eff.RulepackPath = toString(v, eff.RulepackPath)
	}
	if v := get("audit_file"); v != nil {
		eff.AuditFile = toString(v, eff.AuditFile)
	}
	if v := get("metrics_file"); v != nil {
		eff.MetricsFile = toString(v, eff.MetricsFile)
	}
	if v := get("trace_file"); v != nil {
		eff.TraceFile = toString(v, eff.TraceFile)
	}
	if v := get("fail_on"); v != nil {
		eff.FailOn = toString(v, eff.FailOn)
	}
	if v := get("enforcer_mode"); v != nil {
		eff.EnforcerMode = toString(v, eff.EnforcerMode)
	}
	if v := get("require_rulepack_at_startup"); v != nil {
		eff.RequireRulepackAtStartup = toBool(v, eff.RequireRulepackAtStartup)
	}
	if v := get("composition.strategy"); v != nil {
		eff.Composition.Strategy = toString(v, eff.Composition.Strategy)
	}
	if v := get("performance.max_length"); v != nil {
		eff.Performance.MaxLength = toInt(v, eff.Performance.MaxLength)
	}
	if v := get("performance.max_file_size"); v != nil {
		// store as int64; parse via int helper then widen
		iv := toInt(v, int(eff.Performance.MaxFileSizeBytes))
		eff.Performance.MaxFileSizeBytes = int64(iv)
	}
	if v := get("performance.timeout"); v != nil {
		eff.Performance.Timeout = toString(v, eff.Performance.Timeout)
	}
	if v := get("performance.per_rule_timeout"); v != nil {
		eff.Performance.PerRuleTimeout = toString(v, eff.Performance.PerRuleTimeout)
	}
	if v := get("performance.total_scan_timeout"); v != nil {
		eff.Performance.TotalScanTimeout = toString(v, eff.Performance.TotalScanTimeout)
	}
	if v := get("performance.case_sensitive"); v != nil {
		eff.Performance.CaseSensitive = toBool(v, eff.Performance.CaseSensitive)
	}
	if v := get("performance.whole_word"); v != nil {
		eff.Performance.WholeWord = toBool(v, eff.Performance.WholeWord)
	}
	if v := get("performance.buffer_bytes"); v != nil {
		eff.Performance.BufferBytes = toInt(v, eff.Performance.BufferBytes)
	}
	if v := get("performance.chunk_overlap"); v != nil {
		eff.Performance.ChunkOverlap = toInt(v, eff.Performance.ChunkOverlap)
	}
	if v := get("performance.gate.enabled"); v != nil {
		eff.Performance.Gate.Enabled = toBool(v, eff.Performance.Gate.Enabled)
	}
	if v := get("performance.gate.min_token_len"); v != nil {
		eff.Performance.Gate.MinTokenLen = toInt(v, eff.Performance.Gate.MinTokenLen)
	}
	if v := get("redaction.enabled"); v != nil {
		eff.Redaction.Enabled = toBool(v, eff.Redaction.Enabled)
	}
	if v := get("telemetry.enabled"); v != nil {
		eff.Telemetry.Enabled = toBool(v, eff.Telemetry.Enabled)
	}
	if v := get("telemetry.endpoint"); v != nil {
		eff.Telemetry.Endpoint = toString(v, eff.Telemetry.Endpoint)
	}
	if v := get("telemetry.file"); v != nil {
		eff.Telemetry.File = toString(v, eff.Telemetry.File)
	}
	if v := get("telemetry.sample"); v != nil {
		// parse float from various types
		switch x := v.(type) {
		case float64:
			eff.Telemetry.Sample = x
		case int:
			eff.Telemetry.Sample = float64(x)
		case int64:
			eff.Telemetry.Sample = float64(x)
		case string:
			// do not parse here; viper should provide numeric types already
		}
	}
	_ = configFileUsed
	return eff
}

// ValidateConfigFile checks for unknown keys when a config file is used.
func ValidateConfigFile(path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// If file disappeared between Viper open and now, tolerate
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	return CheckUnknownKeys(data)
}

func toString(v any, def string) string {
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
func toInt(v any, def int) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		// do not parse here; Viper should provide numeric types already
		return def
	default:
		return def
	}
}
func toBool(v any, def bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		switch strings.ToLower(s) {
		case "true", "1", "yes", "y":
			return true
		case "false", "0", "no", "n":
			return false
		}
	}
	return def
}
