package scanner_test

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	ttime "time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
)

func BenchmarkScanLargeFile(b *testing.B) {
	sc := scanner.New(0)
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "bench"},
		Rules: []rules.Rule{
			{ID: "kw", Level: 1, Severity: "HIGH", Keywords: []string{"secret"}},
		},
	}})

	// 1MB file with scattered keywords
	content := strings.Repeat(strings.Repeat("x", 1000)+"secret ", 1000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sc.ScanReader(context.Background(), strings.NewReader(content), "large.txt")
	}
}

func BenchmarkParallelScan(b *testing.B) {
	sc := scanner.New(4) // 4 workers
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "bench"},
		Rules: []rules.Rule{
			{ID: "kw", Level: 1, Severity: "HIGH", Keywords: []string{"secret"}},
		},
	}})

	content := strings.Repeat("text with secret\n", 100)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sc.ScanReader(context.Background(), strings.NewReader(content), "file.txt")
		}
	})
}

func BenchmarkRuleMatching(b *testing.B) {
	// 100 rules: 50 keywords + 50 regex
	var benchRules []rules.Rule
	for i := 0; i < 50; i++ {
		benchRules = append(benchRules, rules.Rule{
			ID:       fmt.Sprintf("kw%d", i),
			Level:    1,
			Keywords: []string{fmt.Sprintf("keyword%d", i)},
		})
		benchRules = append(benchRules, rules.Rule{
			ID:       fmt.Sprintf("rx%d", i),
			Level:    2,
			Patterns: []rules.Pattern{{Regex: fmt.Sprintf(`pattern%d\w+`, i)}},
		})
	}

	sc := scanner.New(0)
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "bench"},
		Rules:    benchRules,
	}})

	content := "keyword25 pattern40xyz normal text"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sc.ScanReader(context.Background(), strings.NewReader(content), "test.txt")
	}
}

// BenchmarkP95L1L2 asserts that P95 per-invocation latency stays under a budget on a fixed corpus.
func BenchmarkP95L1L2(b *testing.B) {
	// Build balanced L1/L2 rules
	var benchRules []rules.Rule
	for i := 0; i < 50; i++ {
		benchRules = append(benchRules, rules.Rule{ID: fmt.Sprintf("kw%d", i), Level: 1, Keywords: []string{fmt.Sprintf("keyword%d", i)}})
		benchRules = append(benchRules, rules.Rule{ID: fmt.Sprintf("rx%d", i), Level: 2, Patterns: []rules.Pattern{{Regex: fmt.Sprintf(`token%d-[a-z0-9]{8}`, i)}}})
	}
	sc := scanner.New(8 * 1024 * 1024)
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: benchRules}})
	content := strings.Repeat("some text token10-abcdefgh and keyword25 here\n", 200)
	timings := make([]int64, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := ttime.Now()
		_, _ = sc.ScanReader(context.Background(), strings.NewReader(content), "x")
		timings = append(timings, ttime.Since(start).Milliseconds())
	}
	b.StopTimer()
	if len(timings) == 0 {
		return
	}
	// Compute P95
	p := percentile95(timings)
	// Assert under 25ms budget per-invocation on typical dev hardware
	if p > 25 {
		b.Fatalf("P95 latency %dms exceeds budget", p)
	}
}

// no helper functions needed; using time in this test file

func percentile95(xs []int64) int64 {
	if len(xs) == 1 {
		return xs[0]
	}
	// Simple selection by sorting copy
	ys := make([]int64, len(xs))
	copy(ys, xs)
	// insertion sort for simplicity given b.N sizes
	for i := 1; i < len(ys); i++ {
		v := ys[i]
		j := i - 1
		for j >= 0 && ys[j] > v {
			ys[j+1] = ys[j]
			j--
		}
		ys[j+1] = v
	}
	idx := int(math.Ceil(0.95*float64(len(ys)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ys) {
		idx = len(ys) - 1
	}
	return ys[idx]
}

func BenchmarkAhoKeywordScan(b *testing.B) {
	sc := scanner.New(0)
	// Build 200 keywords
	var benchRules []rules.Rule
	for i := 0; i < 200; i++ {
		benchRules = append(benchRules, rules.Rule{ID: fmt.Sprintf("kw%d", i), Level: 1, Keywords: []string{fmt.Sprintf("keyword%d", i)}})
	}
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "bench"}, Rules: benchRules}})
	content := strings.Repeat("no match text ", 1000) + " keyword123 "
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}

func BenchmarkBloomGateRegex(b *testing.B) {
	sc := scanner.New(0)
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "bench"},
		Rules: []rules.Rule{
			{ID: "rx", Level: 2, Patterns: []rules.Pattern{{Regex: `(?i)token-[a-z0-9]{8}`}}},
		},
	}})
	content := strings.Repeat("no tokens here ", 2000)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.ScanReader(context.Background(), strings.NewReader(content), "x")
	}
}
