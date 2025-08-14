package scanner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/pkg/types"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScanPathsOrderedStream_MemoryGuard(t *testing.T) {
	t.Parallel()
	sc := New(0)
	sc.SetBuiltinKeywordsEnabled(false)
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "t"},
		Rules:    []rules.Rule{{ID: "kw", Level: 1, Severity: "INFO", Keywords: []string{"x"}}},
	}})

	dir := t.TempDir()
	// Create many small files to exercise streaming
	var paths []string
	for i := 0; i < 1000; i++ {
		p := writeFile(t, dir, filepath.Join("d", "f"+strconv.Itoa(i)+".txt"), "x\n")
		paths = append(paths, p)
	}

	// Measure memory before
	var m0, m1 runtime.MemStats
	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)

	emitted := 0
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sc.ScanPathsOrderedStream(ctx, paths, 0, 32, false, func(_ types.ScanResult) error {
		emitted++
		return nil
	}); err != nil {
		t.Fatalf("streaming scan error: %v", err)
	}
	if emitted != len(paths) {
		t.Fatalf("expected %d results, got %d", len(paths), emitted)
	}

	// Measure memory after
	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m1)

	// Guardrail: peak allocation should not grow beyond ~50MB for 1000 small files
	// (heuristic to catch buffering/regressions). Adjust if CI machines differ.
	if delta := int64(m1.Alloc) - int64(m0.Alloc); delta > 50*1024*1024 {
		t.Fatalf("memory delta too high: %d bytes", delta)
	}
}
