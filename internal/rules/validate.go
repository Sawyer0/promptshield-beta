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

	// JSON Schema validation
	if sch, err := getCompiledSchema(); err == nil {
		// Convert RulePack to JSON-compatible format for schema validation
		var generic any
		if p.SourcePath != "" {
			// Validate original source file if available
			if data, rerr := os.ReadFile(p.SourcePath); rerr == nil {
				if uerr := yaml.Unmarshal(data, &generic); uerr == nil {
					// YAML library handles JSON conversion automatically
				}
			}
		}
		if generic == nil {
			// Convert struct to JSON then back to generic interface
			if jsonData, jerr := json.Marshal(p); jerr == nil {
				_ = json.Unmarshal(jsonData, &generic)
			}
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
			// Note: analysis_prompt is not used by omni-moderation models and is optional for all models
		default:
			// schema already constrains 1..3, but keep friendly message if out-of-range sneaks in
			if r.Level < 1 || r.Level > 3 {
				errs = append(errs, fmt.Errorf("%s: rule %s has invalid level %d", p.Metadata.Name, r.ID, r.Level))
			}
		}
	}

	return errs
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
