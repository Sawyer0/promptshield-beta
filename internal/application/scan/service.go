package scan

import (
	"context"
	"runtime"
	"strings"
	"time"

	"os"

	cfg "github.com/promptshield/promptshield/internal/config"
	"github.com/promptshield/promptshield/internal/discovery"
	"github.com/promptshield/promptshield/internal/rules"
	sharederrors "github.com/promptshield/promptshield/internal/shared/errors"
	"github.com/promptshield/promptshield/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

// Options represents inputs to a scan request.
type Options struct {
	RulepackPath  string
	ContextKVs    []string
	Workers       int
	PendingWindow int
}

// Service coordinates rulepack loading, discovery, and scanning.
type Service struct {
	scanner Engine
	// Optional config defaults
	config *cfg.Config
}

func NewService(sc Engine) *Service { return &Service{scanner: sc} }

// WithConfig injects config for rule defaults and returns the service for chaining.
func (s *Service) WithConfig(c cfg.Config) *Service {
	s.config = &c
	return s
}

func containsLevel3(packs []rules.RulePack) bool {
	for _, p := range packs {
		for _, r := range p.Rules {
			if r.Level == 3 {
				return true
			}
		}
		// If no rules parsed (e.g., pack uses spec.rules), do a lightweight parse.
		if len(p.Rules) == 0 && p.SourcePath != "" {
			if hasL3, _ := detectLevel3InSpec(p.SourcePath); hasL3 {
				return true
			}
		}
	}
	return false
}

// detectLevel3InSpec parses only spec.rules to find level 3 without full validation.
func detectLevel3InSpec(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var doc struct {
		Spec struct {
			Rules []struct {
				Level int `yaml:"level" json:"level"`
			} `yaml:"rules" json:"rules"`
		} `yaml:"spec" json:"spec"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false, err
	}
	for _, r := range doc.Spec.Rules {
		if r.Level == 3 {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) scannerTracer() trace.Tracer { return otel.Tracer("promptshield/app") }

// Scan performs path discovery, loads rulepacks, and executes scanning with
// deterministic ordered results.
func (s *Service) Scan(ctx context.Context, args []string, opts Options) ([]types.ScanResult, error) {
	// Apply total scan timeout from config if set
	if s.config != nil && s.config.Performance.TotalScanTimeout != "" {
		if d, err := time.ParseDuration(s.config.Performance.TotalScanTimeout); err == nil && d > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d)
			defer cancel()
		}
	}
	// Root run span
	ctx, runSpan := s.scannerTracer().Start(ctx, "scan_run")
	// Load rulepacks
	if opts.RulepackPath != "" {
		packs, err := rules.LoadPacks(opts.RulepackPath)
		if err != nil {
			runSpan.RecordError(err)
			runSpan.End()
			return nil, err
		}
		// Merge CLI context overrides (last-one-wins per pack)
		if len(opts.ContextKVs) > 0 {
			ctxMap := parseContextKVs(opts.ContextKVs)
			if len(packs) > 0 {
				for i := range packs {
					if packs[i].Context == nil {
						packs[i].Context = map[string]string{}
					}
					for k, v := range ctxMap {
						packs[i].Context[k] = v
					}
				}
			} else {
				// No packs loaded – still pass runtime context to scanner so rules with when/unless can work.
				s.scanner.SetRuntimeContext(ctxMap)
			}
		}
		// If L3 rules are present, require a configured semantic analyzer.
		if containsLevel3(packs) {
			if !s.scanner.HasSemanticAnalyzer() {
				runSpan.RecordError(context.DeadlineExceeded)
				runSpan.End()
				return nil, sharederrors.ErrSemanticAnalyzerNotConfigured
			}
		}

		// Apply global defaults from config before compiling rules
		if s.config != nil {
			var perRuleMs int64
			if d := s.config.Performance.PerRuleTimeout; d != "" {
				if dur, err := time.ParseDuration(d); err == nil {
					perRuleMs = dur.Milliseconds()
				}
			}
			s.scanner.SetRuleDefaults(perRuleMs, s.config.Performance.CaseSensitive, s.config.Performance.WholeWord)
			// Composition strategy override from config if provided
			if s.config.Composition.Strategy != "" {
				s.scanner.SetCompositionStrategy(strings.ToLower(s.config.Composition.Strategy))
			}
			// Enforce file size limit (default to 100MB if not set)
			if s.config.Performance.MaxFileSizeBytes > 0 {
				s.scanner.SetFileSizeLimit(s.config.Performance.MaxFileSizeBytes)
			} else {
				s.scanner.SetFileSizeLimit(100 * 1024 * 1024)
			}
			// Apply max pattern length when provided
			if s.config.Performance.MaxPatternLength > 0 {
				s.scanner.SetMaxPatternLength(s.config.Performance.MaxPatternLength)
			}
			// Apply total scan budget if no explicit context deadline
			if s.config.Performance.Timeout != "" {
				if d, err := time.ParseDuration(s.config.Performance.Timeout); err == nil && d > 0 {
					s.scanner.SetTotalScanBudget(d)
				}
			}
		}
		s.scanner.LoadRulePacks(packs)

		// Derive a runtime context map used for when/unless gating
		runtimeCtx := make(map[string]string)
		for _, p := range packs {
			for k, v := range p.Context {
				runtimeCtx[k] = v
			}
		}
		if len(runtimeCtx) > 0 {
			s.scanner.SetRuntimeContext(runtimeCtx)
		}
	}
	// If no rulepacks were loaded but context KVs provided, still set runtime context.
	if len(opts.ContextKVs) > 0 && s.scanner != nil {
		ctxMap := parseContextKVs(opts.ContextKVs)
		if len(ctxMap) > 0 {
			s.scanner.SetRuntimeContext(ctxMap)
		}
	}

	// Discover input paths
	paths, err := discovery.DiscoverPaths(args)
	if err != nil {
		runSpan.RecordError(err)
		runSpan.End()
		return nil, err
	}

	// Worker sizing
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(paths) {
		workers = len(paths)
	}
	if workers < 1 {
		workers = 1
	}
	window := opts.PendingWindow
	if window <= 0 {
		window = 256
	}

	res, err := s.scanner.ScanPathsOrdered(ctx, paths, workers, window, false)
	if err != nil {
		runSpan.RecordError(err)
	}
	runSpan.End()
	return res, err
}

// StreamOptions mirrors Options but provides an Emit callback for streaming results.
type StreamOptions struct {
	RulepackPath  string
	ContextKVs    []string
	Workers       int
	PendingWindow int
	Emit          func(types.ScanResult) error
}

// Stream performs discovery, rulepack loading, and streams results via Emit in deterministic order.
func (s *Service) Stream(ctx context.Context, args []string, opts StreamOptions) error {
	// Root run span (scanner handles per-file spans)
	_, runSpan := s.scannerTracer().Start(ctx, "scan_run_stream")
	// Load rulepacks (reuse logic from Scan)
	if opts.RulepackPath != "" {
		packs, err := rules.LoadPacks(opts.RulepackPath)
		if err != nil {
			runSpan.RecordError(err)
			runSpan.End()
			return err
		}
		if len(opts.ContextKVs) > 0 {
			ctxMap := parseContextKVs(opts.ContextKVs)
			if len(packs) > 0 {
				for i := range packs {
					if packs[i].Context == nil {
						packs[i].Context = map[string]string{}
					}
					for k, v := range ctxMap {
						packs[i].Context[k] = v
					}
				}
			} else {
				// No packs loaded – still pass runtime context to scanner so rules with when/unless can work.
				s.scanner.SetRuntimeContext(ctxMap)
			}
		}
		if containsLevel3(packs) {
			if !s.scanner.HasSemanticAnalyzer() {
				runSpan.RecordError(context.DeadlineExceeded)
				runSpan.End()
				return sharederrors.ErrSemanticAnalyzerNotConfigured
			}
		}
		if s.config != nil {
			var perRuleMs int64
			if d := s.config.Performance.PerRuleTimeout; d != "" {
				if dur, err := time.ParseDuration(d); err == nil {
					perRuleMs = dur.Milliseconds()
				}
			}
			s.scanner.SetRuleDefaults(perRuleMs, s.config.Performance.CaseSensitive, s.config.Performance.WholeWord)
			// Buffer and chunk overlap tuning
			if s.config.Performance.BufferBytes > 0 {
				s.scanner.SetBufferBytes(s.config.Performance.BufferBytes)
			}
			if s.config.Performance.ChunkOverlap > 0 {
				s.scanner.SetChunkOverlap(s.config.Performance.ChunkOverlap)
			}
		}
		s.scanner.LoadRulePacks(packs)
		runtimeCtx := make(map[string]string)
		for _, p := range packs {
			for k, v := range p.Context {
				runtimeCtx[k] = v
			}
		}
		if len(runtimeCtx) > 0 {
			s.scanner.SetRuntimeContext(runtimeCtx)
		}
	}
	paths, err := discovery.DiscoverPaths(args)
	if err != nil {
		runSpan.RecordError(err)
		runSpan.End()
		return err
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(paths) {
		workers = len(paths)
	}
	if workers < 1 {
		workers = 1
	}
	window := opts.PendingWindow
	if window <= 0 {
		window = 256
	}
	err = s.scanner.ScanPathsOrderedStream(ctx, paths, workers, window, false, opts.Emit)
	if err != nil {
		runSpan.RecordError(err)
	}
	runSpan.End()
	return err
}

func parseContextKVs(kvs []string) map[string]string {
	out := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				k := kv[:i]
				v := kv[i+1:]
				if k != "" {
					out[k] = v
				}
				break
			}
		}
	}
	return out
}
