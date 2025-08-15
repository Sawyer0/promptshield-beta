package scanner

import (
	"log/slog"
	"time"

	bbloom "github.com/bits-and-blooms/bloom/v3"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Scanner performs streaming scans over inputs and produces findings.
// This minimal implementation focuses on streaming and a simple keyword pass
// to validate the CLI and reporting pipeline. It is intentionally simple and
// will be extended to the full rule engine per the plan.
type Scanner struct {
	bufferSizeBytes     int
	compiled            []compiledRule
	runtimeContext      map[string]string
	tracer              trace.Tracer
	logger              *slog.Logger
	budgets             StageBudgets
	quarantineOnTimeout bool
	quarantineOnError   bool
	// Composition and performance controls
	firstMatch      bool
	maxLineForRegex int
	fileTimeout     time.Duration
	maxFileBytes    int64
	// MaxStreamBytes caps total bytes processed per reader scan (0 disables the cap)
	maxStreamBytes      int64
	compositionStrategy string

	// Semantic analyzer pluggable hook
	semantic SemanticAnalyzer

	// Global defaults influencing rule compilation
	defaultRuleTimeoutMs int64
	defaultCaseSensitive bool
	defaultWholeWord     bool

	// High-performance matchers
	aho          *Aho                // for L1 multi-keyword
	bloomL2L3    *bbloom.BloomFilter // quick reject for L2/L3 (kept for future tuning; currently unused)
	ruleTokenAho map[string]*Aho     // per-rule literal token matcher for precise gating
	// Overlap in bytes for chunked long-line evaluation
	chunkOverlapBytes int

	// Rule-level semantic result caches (optional, per rule id)
	semanticCaches map[string]*lru.Cache[string, semanticCacheEntry]

	// Complexity guards
	maxPatternLength int

	// Global resource ceilings
	maxResidentMemoryBytes uint64 // 0 disables check
	totalScanBudget        time.Duration
}

func New(maxTokenBytes int) *Scanner {
	if maxTokenBytes <= 0 {
		maxTokenBytes = 16 * 1024 * 1024 // 16 MiB default
	}
	return &Scanner{
		bufferSizeBytes:     maxTokenBytes,
		tracer:              otel.Tracer("promptshield/scanner"),
		logger:              nil,
		budgets:             defaultBudgets,
		quarantineOnTimeout: DefaultQuarantineOnTimeout,
		quarantineOnError:   DefaultQuarantineOnError,
		semanticCaches:      make(map[string]*lru.Cache[string, semanticCacheEntry]),
		maxPatternLength:    1000,
		totalScanBudget:     0,
	}
}

// semantic helpers moved to semantic.go

// HasSemanticAnalyzer reports whether a semantic analyzer has been configured.
func (s *Scanner) HasSemanticAnalyzer() bool { return s.semantic != nil }

// SetTracer configures an OpenTelemetry tracer for emitting spans. Passing nil resets to default.
func (s *Scanner) SetTracer(t trace.Tracer) {
	if t != nil {
		s.tracer = t
	} else {
		s.tracer = otel.Tracer("promptshield/scanner")
	}
}

// SetLogger configures a logger for debug output.
func (s *Scanner) SetLogger(l *slog.Logger) { s.logger = l }

// LoadRulePacks: see loader.go

// SetRuleDefaults configures global defaults used during rule compilation.
func (s *Scanner) SetRuleDefaults(timeoutMs int64, caseSensitive bool, wholeWord bool) {
	s.defaultRuleTimeoutMs = timeoutMs
	s.defaultCaseSensitive = caseSensitive
	s.defaultWholeWord = wholeWord
}

// SetFileSizeLimit sets a hard cap for input file sizes in bytes (0 disables the cap).
func (s *Scanner) SetFileSizeLimit(limitBytes int64) { s.maxFileBytes = limitBytes }

// SetMaxStreamBytes sets a cap on total bytes processed in ScanReader/scanChunked.
// When exceeded, behavior depends on quarantine flags: either emit a synthetic violation
// and return success, or return an error to be mapped by the runtime API layer.
func (s *Scanner) SetMaxStreamBytes(limitBytes int64) { s.maxStreamBytes = limitBytes }

// SetCompositionStrategy overrides pack-provided composition preference.
// Valid values: "", "first_match", "priority_order".
func (s *Scanner) SetCompositionStrategy(strategy string) { s.compositionStrategy = strategy }

// SetBufferBytes sets the maximum token buffer for line scanning.
func (s *Scanner) SetBufferBytes(n int) {
	if n > 0 {
		s.bufferSizeBytes = n
	}
}

// SetChunkOverlap sets the overlap size used when evaluating very long lines.
func (s *Scanner) SetChunkOverlap(n int) {
	if n >= 0 {
		s.chunkOverlapBytes = n
	}
}

// SetRuntimeContext configures a context map used for rule gating (when/unless).
func (s *Scanner) SetRuntimeContext(ctx map[string]string) {
	s.runtimeContext = ctx
}

// SetMaxPatternLength sets the maximum allowed regex pattern length for compilation.
// Patterns longer than this are skipped to avoid catastrophic complexity.
func (s *Scanner) SetMaxPatternLength(n int) {
	if n > 0 {
		s.maxPatternLength = n
		globalMaxPatternLength = n
	}
}

// SetMaxResidentMemoryBytes configures a hard ceiling; when exceeded during a file scan, the scan aborts.
func (s *Scanner) SetMaxResidentMemoryBytes(b uint64) { s.maxResidentMemoryBytes = b }

// SetTotalScanBudget sets a global budget for an entire multi-file scan. A context deadline takes precedence.
func (s *Scanner) SetTotalScanBudget(d time.Duration) { s.totalScanBudget = d }

// SetQuarantineOnTimeout controls whether timeouts produce a synthetic violation instead of an error.
func (s *Scanner) SetQuarantineOnTimeout(enable bool) { s.quarantineOnTimeout = enable }

// SetQuarantineOnError controls whether non-timeout errors produce a synthetic violation instead of an error.
func (s *Scanner) SetQuarantineOnError(enable bool) { s.quarantineOnError = enable }

// ScanFile, ScanReader, scanChunked: see io.go
// evaluateLine, evaluateLongLine: see evaluate.go
// bytesIndexByte, isWordChar: see util.go
