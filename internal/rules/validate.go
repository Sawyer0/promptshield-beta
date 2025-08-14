package rules

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var embeddedSchema []byte

var (
	schemaOnce     sync.Once
	compiledSchema *jsonschema.Schema
	compileErr     error
)

func getCompiledSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		c := jsonschema.NewCompiler()
		// Use an in-memory loader for the embedded schema
		if err := c.AddResource("inmemory://rulepack.schema.json", bytes.NewReader(embeddedSchema)); err != nil {
			compileErr = fmt.Errorf("adding schema resource: %w", err)
			return
		}
		compiledSchema, compileErr = c.Compile("inmemory://rulepack.schema.json")
	})
	return compiledSchema, compileErr
}

// ValidatePack performs strict validation using JSON Schema plus additional semantic checks.
func ValidatePack(p RulePack) []error {
	var errs []error

	// JSON Schema validation (YAML -> JSON)
	if sch, err := getCompiledSchema(); err == nil {
		// Build a generic JSON-compatible value and validate
		var generic any
		if p.SourcePath != "" {
			if data, rerr := os.ReadFile(p.SourcePath); rerr == nil {
				var y any
				if uerr := yaml.Unmarshal(data, &y); uerr == nil {
					generic = convertYAMLToJSON(y)
				}
			}
		}
		if generic == nil {
			generic = packToJSONMap(p)
		}
		if vErr := sch.Validate(generic); vErr != nil {
			errs = append(errs, flattenValidationError(vErr)...)
		}
	} else {
		// If schema fails to compile, surface the error as a single validation error
		errs = append(errs, fmt.Errorf("schema compile error: %v", err))
	}

	// Duplicate rule IDs and rule-level semantic checks not expressible in JSON Schema
	seen := make(map[string]struct{}, len(p.Rules))
	for _, r := range p.Rules {
		if r.ID == "" {
			errs = append(errs, fmt.Errorf("%s: rule missing id", p.Metadata.Name))
			continue
		}
		if _, ok := seen[r.ID]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate rule id %q", p.Metadata.Name, r.ID))
		}
		seen[r.ID] = struct{}{}
	}

	for _, r := range p.Rules {
		switch r.Level {
		case 1:
			if len(r.Keywords) == 0 {
				errs = append(errs, fmt.Errorf("%s: rule %s (level 1) requires non-empty keywords", p.Metadata.Name, r.ID))
			}
		case 2:
			if len(r.Patterns) == 0 {
				errs = append(errs, fmt.Errorf("%s: rule %s (level 2) requires at least one regex pattern", p.Metadata.Name, r.ID))
			}
			for _, pat := range r.Patterns {
				if len(pat.Regex) > 1000 {
					errs = append(errs, fmt.Errorf("%s: rule %s regex too long (max 1000 chars)", p.Metadata.Name, r.ID))
					continue
				}
				// Compile first to surface syntax errors as 'regex error' per UX/tests
				if _, e := compileRegexStrict(pat.Regex, pat.Flags); e != nil {
					errs = append(errs, fmt.Errorf("%s: rule %s regex error: %v", p.Metadata.Name, r.ID, e))
					continue
				}
				// Then enforce structural complexity limits
				if err := CheckRegexComplexity(pat.Regex, pat.Flags); err != nil {
					errs = append(errs, fmt.Errorf("%s: rule %s regex complexity: %v", p.Metadata.Name, r.ID, err))
					continue
				}
			}
			if r.Timeout != "" {
				if _, e := time.ParseDuration(r.Timeout); e != nil {
					errs = append(errs, fmt.Errorf("%s: rule %s invalid timeout %q", p.Metadata.Name, r.ID, r.Timeout))
				}
			}
		case 3:
			if r.Semantic == nil {
				errs = append(errs, fmt.Errorf("%s: rule %s (level 3) requires semantic configuration", p.Metadata.Name, r.ID))
				break
			}
			if strings.TrimSpace(r.Semantic.Model) == "" {
				errs = append(errs, fmt.Errorf("%s: rule %s (level 3) requires semantic.model", p.Metadata.Name, r.ID))
			}
			if strings.TrimSpace(r.Semantic.AnalysisPrompt) == "" {
				errs = append(errs, fmt.Errorf("%s: rule %s (level 3) requires semantic.analysis_prompt", p.Metadata.Name, r.ID))
			}
		default:
			// schema already constrains 1..3, but keep friendly message if out-of-range sneaks in
			if r.Level < 1 || r.Level > 3 {
				errs = append(errs, fmt.Errorf("%s: rule %s has invalid level %d", p.Metadata.Name, r.ID, r.Level))
			}
		}
	}

	return errs
}

// packRawJSON returns a JSON encoding of the original YAML if SourcePath is set,
// otherwise constructs a JSON-compatible map from the typed struct.
func packRawJSON(p RulePack) ([]byte, error) {
	if p.SourcePath != "" {
		// Best-effort: read original YAML to detect unknown keys via schema
		data, err := os.ReadFile(p.SourcePath)
		if err == nil {
			var generic any
			if e := yaml.Unmarshal(data, &generic); e == nil {
				// Normalize YAML maps to JSON by re-encoding
				return json.Marshal(convertYAMLToJSON(generic))
			}
		}
		// Fall through to struct mapping on error
	}
	m := packToJSONMap(p)
	return json.Marshal(m)
}

// convertYAMLToJSON normalizes YAML-decoded values into JSON-compatible forms.
func convertYAMLToJSON(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = convertYAMLToJSON(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[fmt.Sprint(k)] = convertYAMLToJSON(val)
		}
		return out
	case []any:
		for i := range x {
			x[i] = convertYAMLToJSON(x[i])
		}
		return x
	default:
		return x
	}
}

// packToJSONMap builds a JSON-compatible representation aligned to schema property names.
func packToJSONMap(p RulePack) map[string]any {
	m := map[string]any{
		"apiVersion": p.APIVersion,
		"kind":       p.Kind,
		"metadata": map[string]any{
			"name":        p.Metadata.Name,
			"version":     p.Metadata.Version,
			"description": p.Metadata.Description,
			"author":      p.Metadata.Author,
			"license":     p.Metadata.License,
			"homepage":    p.Metadata.Homepage,
			"repository":  p.Metadata.Repository,
			"tags":        toAnySliceString(p.Metadata.Tags),
		},
	}
	if len(p.Extends) > 0 {
		m["extends"] = toAnySliceString(p.Extends)
	}
	if len(p.Imports) > 0 {
		m["imports"] = toAnySliceString(p.Imports)
	}
	if p.Composition != nil {
		m["composition"] = map[string]any{"strategy": p.Composition.Strategy}
	}
	if p.Performance != nil {
		perf := map[string]any{
			"max_length":         p.Performance.MaxLength,
			"timeout":            p.Performance.Timeout,
			"per_rule_timeout":   p.Performance.PerRuleTimeout,
			"total_scan_timeout": p.Performance.TotalScanTimeout,
			"gate": map[string]any{
				"enabled":       p.Performance.Gate.Enabled,
				"min_token_len": p.Performance.Gate.MinTokenLen,
			},
		}
		m["performance"] = perf
	}
	if len(p.Overrides) > 0 {
		ovs := make([]any, 0, len(p.Overrides))
		for _, o := range p.Overrides {
			ov := map[string]any{
				"rule_id":  o.RuleID,
				"severity": o.Severity,
			}
			if o.Enabled != nil {
				ov["enabled"] = *o.Enabled
			}
			ovs = append(ovs, ov)
		}
		m["overrides"] = ovs
	}
	if len(p.Context) > 0 {
		m["context"] = toAnyMapStringString(p.Context)
	}

	rulesArr := make([]any, 0, len(p.Rules))
	for _, r := range p.Rules {
		rr := map[string]any{
			"id":       r.ID,
			"name":     r.Name,
			"level":    r.Level,
			"severity": r.Severity,
			"category": r.Category,
			"logic":    r.Logic,
			"options": map[string]any{
				"case_sensitive": r.Options.CaseSensitive,
				"whole_word":     r.Options.WholeWord,
			},
		}
		if r.Enabled != nil {
			rr["enabled"] = *r.Enabled
		}
		if len(r.Keywords) > 0 {
			rr["keywords"] = toAnySliceString(r.Keywords)
		}
		if len(r.Patterns) > 0 {
			ps := make([]any, 0, len(r.Patterns))
			for _, p := range r.Patterns {
				ps = append(ps, map[string]any{
					"name":  p.Name,
					"regex": p.Regex,
					"flags": toAnySliceString(p.Flags),
				})
			}
			rr["patterns"] = ps
		}
		if r.Semantic != nil {
			rr["semantic"] = map[string]any{
				"model":                r.Semantic.Model,
				"temperature":          r.Semantic.Temperature,
				"max_tokens":           r.Semantic.MaxTokens,
				"analysis_prompt":      r.Semantic.AnalysisPrompt,
				"confidence_threshold": r.Semantic.ConfidenceThreshold,
				"fallback_on_error":    r.Semantic.FallbackOnError,
			}
		}
		if r.Fallback != nil {
			fb := map[string]any{"action": r.Fallback.Action}
			if len(r.Fallback.Patterns) > 0 {
				fps := make([]any, 0, len(r.Fallback.Patterns))
				for _, p := range r.Fallback.Patterns {
					fps = append(fps, map[string]any{
						"name":  p.Name,
						"regex": p.Regex,
						"flags": toAnySliceString(p.Flags),
					})
				}
				fb["patterns"] = fps
			}
			rr["fallback"] = fb
		}
		if r.Response != nil {
			rr["response"] = map[string]any{
				"action":      r.Response.Action,
				"message":     r.Response.Message,
				"replacement": r.Response.Replacement,
			}
		}
		if r.Cache != nil {
			rr["cache"] = map[string]any{
				"enabled":     r.Cache.Enabled,
				"ttl":         r.Cache.TTL,
				"max_entries": r.Cache.MaxEntries,
			}
		}
		if r.When != nil {
			rr["when"] = map[string]any{"match": r.When.Match}
		}
		if r.Unless != nil {
			rr["unless"] = map[string]any{"match": r.Unless.Match}
		}
		if r.Timeout != "" {
			rr["timeout"] = r.Timeout
		}
		rulesArr = append(rulesArr, rr)
	}
	m["rules"] = rulesArr

	return m
}

func toAnySliceString(in []string) []any {
	if len(in) == 0 {
		return nil
	}
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

func toAnyMapStringString(in map[string]string) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func flattenValidationError(err error) []error {
	// Try to unwrap the jsonschema.ValidationError tree; if not available, return the error as-is.
	var out []error
	var ve *jsonschema.ValidationError
	if errors.As(err, &ve) {
		// Depth-first traversal collecting leaf errors with instance location
		var walk func(e *jsonschema.ValidationError)
		walk = func(e *jsonschema.ValidationError) {
			if len(e.Causes) == 0 {
				loc := normalizeInstanceLocation(e.InstanceLocation)
				if loc == "" {
					loc = "#"
				}
				msg := e.Message
				if msg == "" {
					msg = e.Error()
				}
				out = append(out, fmt.Errorf("%s: %s", loc, msg))
				return
			}
			for _, c := range e.Causes {
				walk(c)
			}
		}
		walk(ve)
		return out
	}
	return []error{err}
}

func normalizeInstanceLocation(ptr string) string {
	// jsonschema uses JSON Pointer like "" (root) or "/metadata/name".
	if ptr == "" || ptr == "#" {
		return "#"
	}
	// Trim leading '#'
	ptr = strings.TrimPrefix(ptr, "#")
	ptr = strings.TrimPrefix(ptr, "/")
	if ptr == "" {
		return "#"
	}
	parts := strings.Split(ptr, "/")
	for i, p := range parts {
		// Unescape JSON Pointer tokens
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		parts[i] = p
	}
	return strings.Join(parts, ".")
}

func compileRegexStrict(expr string, flags []string) (*regexp.Regexp, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty regex")
	}
	allowed := map[string]bool{"ignorecase": true, "i": true, "multiline": true, "m": true}
	prefix := ""
	for _, f := range flags {
		f = strings.ToLower(f)
		if !allowed[f] {
			return nil, fmt.Errorf("invalid flag %q", f)
		}
		switch f {
		case "ignorecase", "i":
			prefix = "(?i)" + prefix
		case "multiline", "m":
			prefix = "(?m)" + prefix
		}
	}
	return regexp.Compile(prefix + expr)
}
