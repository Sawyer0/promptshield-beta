package scanner_test

import (
	"context"
	"errors"
	"strings"
	"testing"

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
