package scanner_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	semfake "github.com/promptshield/promptshield/internal/semantic/fake"
)

// analyzerErr always returns an error (SAFE=false, err!=nil).
type analyzerErr struct{}

func (analyzerErr) Analyze(_ context.Context, _ string, _ rules.Semantic) (bool, float64, error) {
	return false, 0, errors.New("semantic provider error")
}

func TestSemanticLevel3_SafeStillEvaluatesFallback(t *testing.T) {
	sc := scanner.ScanEngineCstor(0)
	// Built-in keyword rules removed
	// Fake analyzer returns SAFE=false (no error) unless [FAKE_MATCH] present.
	sc.SetSemanticAnalyzer(semfake.Analyzer{})
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "sem"},
		Rules: []rules.Rule{{
			ID:    "sem-safe-fallback",
			Level: 3, Severity: "ERROR",
			Semantic: &rules.Semantic{Model: "fake", AnalysisPrompt: "Analyze: {input}"},
			Fallback: &rules.Fallback{Patterns: []rules.Pattern{{Regex: "fallback-token"}}},
		}},
	}})

	// Semantic returns SAFE; fallback pattern exists → should match via fallback
	res, err := sc.ScanReader(context.Background(), strings.NewReader("contains fallback-token"), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation via fallback, got %d", len(res.Violations))
	}

	// No fallback token → no match
	res, err = sc.ScanReader(context.Background(), strings.NewReader("no token here"), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("expected 0 violations without fallback token, got %d", len(res.Violations))
	}
}

func TestSemanticLevel3_Error_NoFallbackOnError(t *testing.T) {
	sc := scanner.ScanEngineCstor(0)
	// Built-in keyword rules removed
	sc.SetSemanticAnalyzer(analyzerErr{})
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "sem"},
		Rules: []rules.Rule{{
			ID: "err-no-fallback", Level: 3, Severity: "ERROR",
			Semantic: &rules.Semantic{Model: "fake", AnalysisPrompt: "Analyze: {input}", FallbackOnError: false},
			Fallback: &rules.Fallback{Patterns: []rules.Pattern{{Regex: "\\bfallback\\b"}}},
		}},
	}})

	res, err := sc.ScanReader(context.Background(), strings.NewReader("use fallback"), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(res.Violations))
	}
}

func TestSemanticLevel3_Error_WithFallbackOnError(t *testing.T) {
	sc := scanner.ScanEngineCstor(0)
	// Built-in keyword rules removed
	sc.SetSemanticAnalyzer(analyzerErr{})
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "sem"},
		Rules: []rules.Rule{{
			ID: "err-with-fallback", Level: 3, Severity: "ERROR",
			Semantic: &rules.Semantic{Model: "fake", AnalysisPrompt: "Analyze: {input}", FallbackOnError: true},
			Fallback: &rules.Fallback{Patterns: []rules.Pattern{{Regex: "\\bfallback\\b"}}},
		}},
	}})

	res, err := sc.ScanReader(context.Background(), strings.NewReader("use fallback"), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(res.Violations))
	}
	if res.Violations[0].RuleID != "err-with-fallback" {
		t.Fatalf("unexpected rule id: %s", res.Violations[0].RuleID)
	}
}

// L3 -> L2 -> L1 path: if semantic fails (timeout) and fallback patterns miss, keyword should still match
func TestEscalation_L3TimeoutThenL2MissThenL1Match(t *testing.T) {
	sc := scanner.ScanEngineCstor(0)
	// Configure semantic to timeout beyond rule timeout
	sc.SetSemanticAnalyzer(semfake.Analyzer{Delay: 200 * time.Millisecond})
	// One pack with all three: L3 semantic with fallback regex not present, plus a keyword rule
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "combo"},
		Rules: []rules.Rule{
			{ID: "l3", Level: 3, Severity: "ERROR", Timeout: "50ms", Semantic: &rules.Semantic{Model: "fake", AnalysisPrompt: "Detect: {input}", FallbackOnError: true}, Fallback: &rules.Fallback{Patterns: []rules.Pattern{{Regex: "present-only-in-alt"}}}},
			{ID: "l2", Level: 2, Severity: "WARNING", Patterns: []rules.Pattern{{Regex: "nope-not-here"}}},
			{ID: "l1", Level: 1, Severity: "INFO", Keywords: []string{"keyword"}},
		},
	}})

	res, err := sc.ScanReader(context.Background(), strings.NewReader("this has a keyword"), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Violations) != 1 || res.Violations[0].RuleID != "l1" {
		t.Fatalf("expected fallback to L1 keyword only, got %v", res.Violations)
	}
}

// L3 success in first_match mode with no L1 trigger; should include an L3 violation
func TestEscalation_L3SuccessWithFirstMatchIncludesL3(t *testing.T) {
	sc := scanner.ScanEngineCstor(0)
	sc.SetSemanticAnalyzer(semfake.Analyzer{}) // immediate success
	sc.SetCompositionStrategy("first_match")
	// Include an L3 gating token in a regex so L3 is evaluated
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "combo"},
		Rules: []rules.Rule{
			{ID: "l3", Level: 3, Severity: "ERROR", Patterns: []rules.Pattern{{Regex: "FAKE_MATCH"}}, Semantic: &rules.Semantic{Model: "fake", AnalysisPrompt: "Detect: {input}"}},
			{ID: "l2", Level: 2, Severity: "WARNING", Patterns: []rules.Pattern{{Regex: "keyword"}}},
			{ID: "l1", Level: 1, Severity: "INFO", Keywords: []string{"keyword"}},
		},
	}})
	// Do NOT include the 'keyword' token; only FAKE_MATCH
	res, err := sc.ScanReader(context.Background(), strings.NewReader("contains [FAKE_MATCH] token"), "x")
	if err != nil {
		t.Fatal(err)
	}
	foundL3 := false
	for _, v := range res.Violations {
		if v.RuleID == "l3" {
			foundL3 = true
			break
		}
	}
	if !foundL3 {
		t.Fatalf("expected L3 violation present, got %v", res.Violations)
	}
}
