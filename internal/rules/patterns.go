package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CompilePattern compiles a rule pattern the same way the scanner does:
// the i/ignorecase and m/multiline flags become inline (?i)/(?m) prefixes and
// the expression must be valid RE2 syntax (no lookaround or backreferences).
func CompilePattern(expr string, flags []string) (*regexp.Regexp, error) {
	if expr == "" {
		return nil, errors.New("empty regex")
	}
	prefix := ""
	for _, f := range flags {
		switch strings.ToLower(f) {
		case "ignorecase", "i":
			prefix = "(?i)" + prefix
		case "multiline", "m":
			prefix = "(?m)" + prefix
		}
	}
	return regexp.Compile(prefix + expr)
}

// CheckPatterns compiles every regex in the pack, including fallback
// patterns, and returns one error per pattern that does not compile. Each
// error names the pack and the rule id so a bad rule is never dropped
// silently.
func CheckPatterns(p RulePack) error {
	var errs []error
	check := func(ruleID, kind string, i int, pat Pattern) {
		if _, err := CompilePattern(pat.Regex, pat.Flags); err != nil {
			errs = append(errs, fmt.Errorf("rulepack %q: rule %q: %s %d: invalid regex %q: %w",
				p.Metadata.Name, ruleID, kind, i, pat.Regex, err))
		}
	}
	for _, r := range p.Rules {
		for i, pat := range r.Patterns {
			check(r.ID, "pattern", i, pat)
		}
		if r.Fallback != nil {
			for i, pat := range r.Fallback.Patterns {
				check(r.ID, "fallback pattern", i, pat)
			}
		}
	}
	return errors.Join(errs...)
}
