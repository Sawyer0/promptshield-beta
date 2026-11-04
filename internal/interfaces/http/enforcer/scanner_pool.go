package enforcerhttp

import (
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/promptshield/promptshield/internal/application/rulesets"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	semopenai "github.com/promptshield/promptshield/internal/semantic/openai"
)

// scannerPool hands out scanners preloaded with the active RulePacks. When
// the rule set reloads, the pool is replaced so new requests scan with the
// new packs while in-flight requests finish on the old ones.
type scannerPool struct {
	current        atomic.Pointer[sync.Pool]
	maxStreamBytes int64
}

func newScannerPool(set *rulesets.Set, maxStreamBytes int64) *scannerPool {
	p := &scannerPool{maxStreamBytes: maxStreamBytes}
	p.swap(set.Packs())
	set.OnChange(p.swap)
	return p
}

func (p *scannerPool) swap(packs []rules.RulePack) {
	pool := &sync.Pool{New: func() any { return p.build(packs) }}
	p.current.Store(pool)
}

// get returns a scanner and a release func that returns it to its pool.
func (p *scannerPool) get() (*scanner.Scanner, func()) {
	pool := p.current.Load()
	sc := pool.Get().(*scanner.Scanner)
	return sc, func() { pool.Put(sc) }
}

func (p *scannerPool) build(packs []rules.RulePack) *scanner.Scanner {
	sc := scanner.ScanEngineCstor(0)
	if p.maxStreamBytes > 0 {
		sc.SetMaxStreamBytes(p.maxStreamBytes)
	}
	sc.SetQuarantineOnTimeout(true)
	sc.SetQuarantineOnError(true)
	if len(packs) > 0 {
		sc.LoadRulePacks(packs)
	}
	if analyzer := semanticAnalyzerFromEnv(); analyzer != nil {
		sc.SetSemanticAnalyzer(analyzer)
	}
	return sc
}

// semanticAnalyzerFromEnv returns the OpenAI analyzer when
// PS_SEMANTIC_ENABLED=true, PS_SEMANTIC_PROVIDER=openai and a key is set.
func semanticAnalyzerFromEnv() *semopenai.Analyzer {
	if os.Getenv("PS_SEMANTIC_ENABLED") != "true" || os.Getenv("PS_SEMANTIC_PROVIDER") != "openai" {
		return nil
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil
	}
	analyzer := semopenai.New(semopenai.Options{
		APIKey:            apiKey,
		MaxConcurrency:    2,
		CacheSize:         1000,
		CacheTTL:          15 * time.Minute,
		RequestsPerSecond: 10,
		BurstSize:         20,
	})
	slog.With("component", "semantic").Info("OpenAI semantic analyzer initialized")
	return analyzer
}
