package fake

import (
	"context"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
)

// Analyzer is a deterministic test double for SemanticAnalyzer.
// It matches when the input contains the token "[FAKE_MATCH]" (case-insensitive).
// Optional delay can simulate latency/timeout scenarios.
type Analyzer struct {
	Delay time.Duration
}

func (a Analyzer) Analyze(ctx context.Context, input string, _ rules.Semantic) (bool, float64, error) {
	if a.Delay > 0 {
		select {
		case <-time.After(a.Delay):
		case <-ctx.Done():
			return false, 0, ctx.Err()
		}
	}
	if strings.Contains(strings.ToLower(input), "[fake_match]") {
		return true, 1.0, nil
	}
	return false, 1.0, nil
}
