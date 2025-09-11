package scanner_test

import (
	"context"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
)

// repeatingReader generates a fixed number of bytes by emitting a given line repeatedly.
type repeatingReader struct {
	line   []byte
	remain int64
}

func newRepeatingReader(totalBytes int64, line string) *repeatingReader {
	return &repeatingReader{line: []byte(line), remain: totalBytes}
}

func (r *repeatingReader) Read(p []byte) (int, error) {
	if r.remain <= 0 {
		return 0, io.EOF
	}
	// Fill p with repeated copies of r.line up to remain
	written := 0
	for written < len(p) && r.remain > 0 {
		n := copy(p[written:], r.line)
		written += n
		r.remain -= int64(n)
	}
	// Per io.Reader contract, if we wrote some bytes, return nil error
	if written > 0 {
		return written, nil
	}
	return 0, io.EOF
}

// Benchmark scanning of ~1 GiB stream without allocations proportional to input size.
func BenchmarkScanOneGiB(b *testing.B) {
	const oneGiB = 1 << 30
	sc := scanner.ScanEngineCstor(0)
	// Built-in keyword rules removed
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "bench-1g"},
		// Use a rule unlikely to match to avoid result growth during the bench
		Rules: []rules.Rule{{ID: "kw", Level: 1, Severity: "INFO", Keywords: []string{"zzzz_unlikely_keyword"}}},
	}})

	b.SetBytes(oneGiB)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := newRepeatingReader(oneGiB, "normal line without keywords\n")
		if _, err := sc.ScanReader(context.Background(), rr, "1GiB-stream"); err != nil {
			b.Fatalf("scan error: %v", err)
		}
	}
}

// Optional large-file memory budget test. Skipped unless PS_RUN_LARGE=1.
func TestScanReader_OneGiB_MemoryBudget(t *testing.T) {
	if os.Getenv("PS_RUN_LARGE") != "1" {
		t.Skip("set PS_RUN_LARGE=1 to run 1GiB memory budget test")
	}
	const oneGiB = 1 << 30
	sc := scanner.ScanEngineCstor(0)
	// Built-in keyword rules removed
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "mem-1g"},
		Rules:    []rules.Rule{{ID: "kw", Level: 1, Severity: "INFO", Keywords: []string{"zzzz_unlikely_keyword"}}},
	}})

	// Measure memory before
	var m0, m1 runtime.MemStats
	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)

	rr := newRepeatingReader(oneGiB, "normal line without keywords\n")
	if _, err := sc.ScanReader(context.Background(), rr, "1GiB-stream"); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m1)

	const maxDelta = 500 * 1024 * 1024 // 500MB
	if delta := int64(m1.Alloc) - int64(m0.Alloc); delta > maxDelta {
		t.Fatalf("memory delta too high: %d bytes (limit %d)", delta, maxDelta)
	}
}
