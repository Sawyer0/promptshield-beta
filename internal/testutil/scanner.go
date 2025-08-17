package testutil

import (
	"context"
	"strings"
	"testing"

	"github.com/promptshield/promptshield/internal/rules"
	scpkg "github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/pkg/types"
)

// ScannerHelper provides common test utilities for scanner tests
type ScannerHelper struct {
	t  *testing.T
	sc *scpkg.Scanner
}

// NewScannerHelper creates a scanner with the given rules
func NewScannerHelper(t *testing.T, testRules []rules.Rule) *ScannerHelper {
	t.Helper()
	sc := scpkg.New(0)
	// Built-in keyword rules removed; nothing to disable
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "test"},
		Rules:    testRules,
	}})
	return &ScannerHelper{t: t, sc: sc}
}

// Scan runs a scan and returns violations
func (h *ScannerHelper) Scan(input string) []types.Violation {
	h.t.Helper()
	res, err := h.sc.ScanReader(context.Background(), strings.NewReader(input), "test")
	if err != nil {
		h.t.Fatalf("scan failed: %v", err)
	}
	return res.Violations
}

// AssertViolations checks violation count
func (h *ScannerHelper) AssertViolations(input string, want int) {
	h.t.Helper()
	got := h.Scan(input)
	if len(got) != want {
		h.t.Errorf("input %q: got %d violations, want %d", input, len(got), want)
	}
}

// WithContext sets runtime context
func (h *ScannerHelper) WithContext(ctx map[string]string) *ScannerHelper {
	h.sc.SetRuntimeContext(ctx)
	return h
}

// Rules provides common test rules
var Rules = struct {
	Keyword    rules.Rule
	Regex      rules.Rule
	EmailRegex rules.Rule
	WithWhen   rules.Rule
	WithUnless rules.Rule
}{
	Keyword: rules.Rule{
		ID:       "kw",
		Level:    1,
		Severity: "HIGH",
		Keywords: []string{"secret", "password"},
	},
	Regex: rules.Rule{
		ID:       "rx",
		Level:    2,
		Severity: "HIGH",
		Patterns: []rules.Pattern{{Regex: `sk-[a-z0-9]{24,}`, Flags: []string{"i"}}},
	},
	EmailRegex: rules.Rule{
		ID:       "email",
		Level:    2,
		Severity: "MEDIUM",
		Patterns: []rules.Pattern{{Regex: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`}},
	},
	WithWhen: rules.Rule{
		ID:       "prod-only",
		Level:    1,
		Keywords: []string{"secret"},
		When:     &rules.Condition{Match: map[string][]string{"env": {"production"}}},
	},
	WithUnless: rules.Rule{
		ID:       "not-dev",
		Level:    1,
		Keywords: []string{"password"},
		Unless:   &rules.Condition{Match: map[string][]string{"env": {"development"}}},
	},
}
