package scanner

import (
	"strings"
	"testing"

	"github.com/promptshield/promptshield/internal/rules"
)

func TestCompileRules_ReportsUncompilablePatterns(t *testing.T) {
	compiled, errs := compileRules([]rules.Rule{
		{ID: "good", Level: 2, Patterns: []rules.Pattern{{Regex: `\d{3}`}}},
		{ID: "lookahead", Level: 2, Patterns: []rules.Pattern{{Regex: `\b(?!000)\d{3}\b`}, {Regex: `abc`}}},
	}, 0, false, false)

	if len(compiled) != 2 {
		t.Fatalf("compiled %d rules, want 2", len(compiled))
	}
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if msg := errs[0].Error(); !strings.Contains(msg, `rule "lookahead": pattern 0`) || !strings.Contains(msg, "(?!") {
		t.Errorf("error %q does not name the rule and pattern", msg)
	}
	if n := len(compiled[1].regexes); n != 1 {
		t.Errorf("lookahead rule kept %d regexes, want 1 (the valid one)", n)
	}
}
