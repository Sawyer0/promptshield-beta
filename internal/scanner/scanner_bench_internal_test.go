package scanner

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/promptshield/promptshield/internal/rules"
)

func BenchmarkAhoKeywords_On(b *testing.B) {
	sc := ScanEngineCstor(0)
	// 200 L1 keywords, case-insensitive path (default)
	var rs []rules.Rule
	for i := 0; i < 200; i++ {
		rs = append(rs, rules.Rule{ID: fmt.Sprintf("kw%d", i), Level: 1, Keywords: []string{fmt.Sprintf("keyword%d", i)}})
	}
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: rs}})
	content := strings.Repeat("no hits here ", 2000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}

func BenchmarkAhoKeywords_Off(b *testing.B) {
	sc := ScanEngineCstor(0)
	var rs []rules.Rule
	for i := 0; i < 200; i++ {
		rs = append(rs, rules.Rule{ID: fmt.Sprintf("kw%d", i), Level: 1, Keywords: []string{fmt.Sprintf("keyword%d", i)}})
	}
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: rs}})
	// Disable Aho to measure fallback path
	sc.aho = nil
	content := strings.Repeat("no hits here ", 2000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}

func BenchmarkRegexGate_On(b *testing.B) {
	sc := ScanEngineCstor(0)
	rs := []rules.Rule{
		{ID: "rx1", Level: 2, Patterns: []rules.Pattern{{Regex: `(?i)token-[a-z0-9]{8}`}}},
	}
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: rs}})
	content := strings.Repeat("no tokens here ", 4000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}

func BenchmarkRegexGate_Off(b *testing.B) {
	sc := ScanEngineCstor(0)
	rs := []rules.Rule{
		{ID: "rx1", Level: 2, Patterns: []rules.Pattern{{Regex: `(?i)token-[a-z0-9]{8}`}}},
	}
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: rs}})
	// Disable gates to force regex work
	sc.ruleTokenAho = nil
	sc.bloomL2L3 = nil
	content := strings.Repeat("no tokens here ", 4000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}

func buildHeavyL2Rules(n int) []rules.Rule {
	rs := make([]rules.Rule, 0, n)
	for i := 0; i < n; i++ {
		// Mix of literal-containing regex patterns
		pat := rules.Pattern{Regex: fmt.Sprintf(`(?i)(tokenx%d|apikey%d|password%d)[a-z0-9]{6,}`, i, i, i)}
		rs = append(rs, rules.Rule{ID: fmt.Sprintf("rx%d", i), Level: 2, Patterns: []rules.Pattern{pat}})
	}
	return rs
}

func BenchmarkRegexGate_Heavy_NoMatch_On(b *testing.B) {
	sc := ScanEngineCstor(0)
	rs := buildHeavyL2Rules(500)
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: rs}})
	content := strings.Repeat("no tokens here ", 100000) // ~1.8MB
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}

func BenchmarkRegexGate_Heavy_NoMatch_Off(b *testing.B) {
	sc := ScanEngineCstor(0)
	rs := buildHeavyL2Rules(500)
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: rs}})
	sc.ruleTokenAho = nil
	sc.bloomL2L3 = nil
	content := strings.Repeat("no tokens here ", 100000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}

func BenchmarkRegexGate_Heavy_Maybe_On(b *testing.B) {
	sc := ScanEngineCstor(0)
	rs := buildHeavyL2Rules(500)
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: rs}})
	content := strings.Repeat("prefix ", 50000) + " apikey42ABCDXX " + strings.Repeat(" suffix", 50000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}

func BenchmarkRegexGate_Heavy_Maybe_Off(b *testing.B) {
	sc := ScanEngineCstor(0)
	rs := buildHeavyL2Rules(500)
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: rs}})
	sc.ruleTokenAho = nil
	sc.bloomL2L3 = nil
	content := strings.Repeat("prefix ", 50000) + " apikey42ABCDXX " + strings.Repeat(" suffix", 50000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}
