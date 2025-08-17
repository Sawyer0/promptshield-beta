package scan

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/pkg/types"
)

// Engine abstracts the scanning engine so tests can inject fakes.
// It contains only the methods required by Service.
type Engine interface {
	LoadRulePacks([]rules.RulePack)
	HasSemanticAnalyzer() bool
	SetRuleDefaults(perRuleMs int64, caseSensitive, wholeWord bool)
	SetCompositionStrategy(string)
	SetFileSizeLimit(int64)
	SetMaxPatternLength(int)
	SetTotalScanBudget(time.Duration)
	SetBufferBytes(int)
	SetChunkOverlap(int)
	SetRuntimeContext(map[string]string)
	ScanPathsOrdered(ctx context.Context, paths []string, workers, window int, includeBinary bool) ([]types.ScanResult, error)
	ScanPathsOrderedStream(ctx context.Context, paths []string, workers, window int, includeBinary bool, emit func(types.ScanResult) error) error
}
