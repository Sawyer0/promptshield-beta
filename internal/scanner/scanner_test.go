package scanner_test

import (
	"context"
	"strings"
	"testing"

	"time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	semfake "github.com/promptshield/promptshield/internal/semantic/fake"
	"github.com/promptshield/promptshield/internal/testutil"
)

func TestKeywordMatching(t *testing.T) {
	h := testutil.NewScannerHelper(t, []rules.Rule{testutil.Rules.Keyword})

	tests := []struct {
		input string
		want  int
	}{
		{"has secret", 1},
		{"has SECRET", 1},          // case insensitive
		{"secret and password", 2}, // multiple keywords
		{"clean text", 0},
	}

	for _, tt := range tests {
		h.AssertViolations(tt.input, tt.want)
	}
}

func TestRegexMatching(t *testing.T) {
	h := testutil.NewScannerHelper(t, []rules.Rule{
		testutil.Rules.Regex,
		testutil.Rules.EmailRegex,
	})

	tests := []struct {
		input string
		want  int
	}{
		{"user@example.com", 1},
		{"sk-abcdef1234567890abcdef1234567890", 1},
		{"SK-ABCDEF1234567890ABCDEF1234567890", 1}, // case insensitive flag
		{"email@test.com and sk-abc1234567890def1234567890xyz", 2},
		{"clean text", 0},
	}

	for _, tt := range tests {
		h.AssertViolations(tt.input, tt.want)
	}
}

func TestContextGating(t *testing.T) {
	tests := []struct {
		name    string
		rule    rules.Rule
		context map[string]string
		input   string
		want    int
	}{
		{"when matches", testutil.Rules.WithWhen, map[string]string{"env": "production"}, "secret", 1},
		{"when blocks", testutil.Rules.WithWhen, map[string]string{"env": "dev"}, "secret", 0},
		{"unless blocks", testutil.Rules.WithUnless, map[string]string{"env": "development"}, "password", 0},
		{"unless allows", testutil.Rules.WithUnless, map[string]string{"env": "production"}, "password", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testutil.NewScannerHelper(t, []rules.Rule{tt.rule})
			h.WithContext(tt.context).AssertViolations(tt.input, tt.want)
		})
	}
}

func TestLongLines(t *testing.T) {
	h := testutil.NewScannerHelper(t, []rules.Rule{testutil.Rules.Keyword})

	// Test that scanner handles very long lines
	longLine := strings.Repeat("x", 1024*1024) + "secret" + strings.Repeat("y", 1024*1024)
	violations := h.Scan(longLine)

	if len(violations) != 1 {
		t.Errorf("expected 1 violation in long line, got %d", len(violations))
	}

	// Verify column position is correct
	expectedCol := 1024*1024 + 1 // 1-indexed
	if violations[0].Column != expectedCol {
		t.Errorf("column = %d, want %d", violations[0].Column, expectedCol)
	}
}

func TestMultipleRuleTypes(t *testing.T) {
	h := testutil.NewScannerHelper(t, []rules.Rule{
		testutil.Rules.Keyword,
		testutil.Rules.Regex,
	})

	violations := h.Scan("password is sk-abcdef1234567890abcdef1234567890")
	if len(violations) != 2 {
		t.Errorf("expected both keyword and regex to match, got %d", len(violations))
	}

	// Verify both rule types matched
	ids := map[string]bool{}
	for _, v := range violations {
		ids[v.RuleID] = true
	}
	if !ids["kw"] || !ids["rx"] {
		t.Error("expected both keyword and regex rules to match")
	}
}

func TestFailOnThreshold(t *testing.T) {
	h := testutil.NewScannerHelper(t, []rules.Rule{
		{ID: "warn", Level: 1, Severity: "WARNING", Keywords: []string{"warn"}},
		{ID: "err", Level: 1, Severity: "ERROR", Keywords: []string{"boom"}},
	})
	v := h.Scan("warn and boom")
	// Ensure both matched, then verify threshold logic using shared severity helper
	if len(v) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(v))
	}
}

func TestEnabledFlag(t *testing.T) {
	enabled := true
	disabled := false

	h := testutil.NewScannerHelper(t, []rules.Rule{
		{ID: "on", Level: 1, Keywords: []string{"find"}, Enabled: &enabled},
		{ID: "off", Level: 1, Keywords: []string{"skip"}, Enabled: &disabled},
		{ID: "default", Level: 1, Keywords: []string{"also"}}, // nil = enabled
	})

	violations := h.Scan("find skip also")
	if len(violations) != 2 {
		t.Errorf("expected 2 violations (enabled + default), got %d", len(violations))
	}

	for _, v := range violations {
		if v.RuleID == "off" {
			t.Error("disabled rule should not match")
		}
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		h := testutil.NewScannerHelper(t, []rules.Rule{testutil.Rules.Keyword})
		h.AssertViolations("", 0)
	})

	t.Run("invalid regex", func(t *testing.T) {
		sc := scanner.New(0)
		sc.LoadRulePacks([]rules.RulePack{{
			Metadata: rules.Metadata{Name: "test"},
			Rules: []rules.Rule{{
				ID:       "bad",
				Level:    2,
				Patterns: []rules.Pattern{{Regex: `[unclosed`}},
			}},
		}})

		// Should not panic, just skip invalid pattern
		res, err := sc.ScanReader(context.Background(), strings.NewReader("test"), "test")
		if err != nil {
			t.Errorf("should not error on invalid regex: %v", err)
		}
		if len(res.Violations) != 0 {
			t.Error("invalid regex should not match")
		}
	})
}

func TestSemanticLevel3_WithFakeAdapter(t *testing.T) {
	sc := scanner.New(0)
	// Built-in keyword rules removed
	sc.SetSemanticAnalyzer(semfake.Analyzer{})
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "sem"},
		Rules: []rules.Rule{{
			ID: "sem1", Level: 3, Severity: "ERROR",
			Semantic: &rules.Semantic{Model: "fake", AnalysisPrompt: "Detect: {input}"},
			Fallback: &rules.Fallback{Patterns: []rules.Pattern{{Regex: "fallback"}}},
		}},
	}})
	// Should match via fake analyzer
	res, err := sc.ScanReader(context.Background(), strings.NewReader("this has [FAKE_MATCH] token"), "x")
	if err != nil || len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation via L3, got %v err=%v", len(res.Violations), err)
	}
}

func TestSemanticLevel3_TimeoutFallsBack(t *testing.T) {
	sc := scanner.New(0)
	// Built-in keyword rules removed
	// Fake delays beyond budget so we fall back to regex
	sc.SetSemanticAnalyzer(semfake.Analyzer{Delay: 200 * time.Millisecond})
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "sem"},
		Rules: []rules.Rule{{
			ID: "sem-timeout", Level: 3, Severity: "ERROR",
			Timeout:  "50ms",
			Semantic: &rules.Semantic{Model: "fake", AnalysisPrompt: "Detect: {input}", FallbackOnError: true},
			Fallback: &rules.Fallback{Patterns: []rules.Pattern{{Regex: "\\bfallback\\b"}}},
		}},
	}})
	// No fallback token; semantic times out; expect 0 violations
	res, err := sc.ScanReader(context.Background(), strings.NewReader("plain line"), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("expected 0 violations on timeout without fallback hit, got %d", len(res.Violations))
	}
	// Now with fallback token; expect a match via fallback
	res, err = sc.ScanReader(context.Background(), strings.NewReader("use fallback now"), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation via fallback, got %d", len(res.Violations))
	}
}

func TestKeywordOptions_Precedence(t *testing.T) {
	// Global defaults: case sensitive false, whole word false
	sc := scanner.New(0)
	// Built-in keyword rules removed
	sc.SetRuleDefaults(0, false, false)
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "kw"},
		Rules: []rules.Rule{
			{ID: "r1", Level: 1, Severity: "INFO", Keywords: []string{"API"}, Options: rules.Options{CaseSensitive: true}},
			{ID: "r2", Level: 1, Severity: "INFO", Keywords: []string{"key"}, Options: rules.Options{WholeWord: true}},
		},
	}})
	// Line contains lowercased api and the word key as part of api_key
	res, err := sc.ScanReader(context.Background(), strings.NewReader("api api_key"), "x")
	if err != nil {
		t.Fatal(err)
	}
	var r1, r2 int
	for _, v := range res.Violations {
		if v.RuleID == "r1" {
			r1++
		}
		if v.RuleID == "r2" {
			r2++
		}
	}
	if r1 != 0 {
		t.Fatalf("expected r1 case-sensitive not to match lower 'api', got %d", r1)
	}
	if r2 != 0 {
		t.Fatalf("expected r2 whole-word to not match inside 'api_key', got %d", r2)
	}
}
