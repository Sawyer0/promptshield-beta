package scanner

import "time"

type StageBudgets struct {
	Canonicalization time.Duration
	Level1Keywords   time.Duration
	Level2Regex      time.Duration
	Level3Semantic   time.Duration
    // PerFile bounds total processing time for a single file
    PerFile          time.Duration
}

const (
	DefaultQuarantineOnTimeout = true
	DefaultQuarantineOnError   = true
)

var defaultBudgets = StageBudgets{
	Canonicalization: 5 * time.Millisecond,
	Level1Keywords:   10 * time.Millisecond,
	Level2Regex:      50 * time.Millisecond,
    Level3Semantic:   300 * time.Millisecond,
    PerFile:          10 * time.Second,
}
