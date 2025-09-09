package scanner_test

import (
	"context"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
)

// TestScanThroughput measures scanning throughput under various conditions
func TestScanThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping throughput test in short mode")
	}

	sc := scanner.ScanEngineCstor(0)

	// Load comprehensive rule set using exact pattern from large_bench_test.go
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "throughput-test"},
		Rules: []rules.Rule{
			{ID: "kw1", Level: 1, Keywords: []string{"password", "secret", "api_key"}, Severity: "HIGH"},
			{ID: "kw2", Level: 1, Keywords: []string{"token", "credential", "private"}, Severity: "MEDIUM"},
			{ID: "regex1", Level: 2, Patterns: []rules.Pattern{{Regex: `\b\d{3}-\d{2}-\d{4}\b`}}, Severity: "CRITICAL"}, // SSN
		},
	}})

	tests := []struct {
		name        string
		contentSize int
		content     string
	}{
		{"Small_NoViolations", 1024, strings.Repeat("normal text line here\n", 50)},
		{"Medium_NoViolations", 100 * 1024, strings.Repeat("normal text line here\n", 5000)},
		{"Large_NoViolations", 1024 * 1024, strings.Repeat("normal text line here\n", 50000)},
		{"Small_WithViolations", 1024, strings.Repeat("password is secret123\n", 50)},
		{"Medium_WithViolations", 100 * 1024, strings.Repeat("password is secret123\n", 5000)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := tc.content
			if len(content) > tc.contentSize {
				content = content[:tc.contentSize]
			}

			// Measure throughput
			start := time.Now()
			result, err := sc.ScanReader(context.Background(), strings.NewReader(content), tc.name)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("scan error: %v", err)
			}

			throughputMBps := float64(len(content)) / (1024 * 1024) / duration.Seconds()
			t.Logf("Throughput: %.2f MB/s, Content: %d bytes, Duration: %v, Violations: %d",
				throughputMBps, len(content), duration, len(result.Violations))

			// Performance threshold (conservative for CI)
			minThroughputMBps := 0.8 // 0.8 MB/s minimum (CI-safe)
			if throughputMBps < minThroughputMBps {
				t.Errorf("Throughput too low: %.2f MB/s < %.2f MB/s", throughputMBps, minThroughputMBps)
			}
		})
	}
}

// TestMemoryGrowth checks for memory leaks during extended scanning
func TestMemoryGrowth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory growth test in short mode")
	}

	sc := scanner.ScanEngineCstor(0)

	// Load rules using working pattern
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "memory-test"},
		Rules: []rules.Rule{
			{ID: "test", Level: 1, Keywords: []string{"findme"}, Severity: "INFO"},
		},
	}})

	// Baseline memory measurement - using exact pattern from large_bench_test.go
	var m0, m1 runtime.MemStats
	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)

	content := strings.Repeat("some text without matches here\n", 1000)

	// Perform many scans
	iterations := 100 // Reduced for CI stability
	for i := 0; i < iterations; i++ {
		_, err := sc.ScanReader(context.Background(), strings.NewReader(content), "memory-test")
		if err != nil {
			t.Fatalf("scan error at iteration %d: %v", i, err)
		}
	}

	// Final memory check using exact pattern from large_bench_test.go
	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m1)

	// Use same threshold pattern as large_bench_test.go
	const maxDelta = 50 * 1024 * 1024 // 50MB
	if delta := int64(m1.Alloc) - int64(m0.Alloc); delta > maxDelta {
		t.Errorf("memory delta too high: %d bytes (limit %d) after %d iterations", delta, maxDelta, iterations)
	}

	finalGrowthMB := float64(m1.Alloc-m0.Alloc) / (1024 * 1024)
	t.Logf("Memory growth: %.2f MB after %d iterations", finalGrowthMB, iterations)
}

// TestConcurrentScanPerformance tests performance under concurrent load
func TestConcurrentScanPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent performance test in short mode")
	}

	sc := scanner.ScanEngineCstor(0)

	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "concurrent-test"},
		Rules: []rules.Rule{
			{ID: "concurrent", Level: 1, Keywords: []string{"secret"}, Severity: "HIGH"},
		},
	}})

	content := strings.Repeat("some content with secret in it\n", 100) // Small content for speed

	tests := []struct {
		name       string
		goroutines int
		scansEach  int
	}{
		{"Low_Concurrency", 2, 10},
		{"Medium_Concurrency", 5, 5},
		{"High_Concurrency", 10, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup
			start := time.Now()

			for i := 0; i < tc.goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < tc.scansEach; j++ {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						_, err := sc.ScanReader(ctx, strings.NewReader(content), "concurrent-test")
						cancel()
						if err != nil {
							t.Errorf("goroutine %d scan %d failed: %v", id, j, err)
						}
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(start)

			totalScans := tc.goroutines * tc.scansEach
			scansPerSec := float64(totalScans) / duration.Seconds()

			t.Logf("Concurrent performance: %.2f scans/sec with %d goroutines (%d total scans in %v)",
				scansPerSec, tc.goroutines, totalScans, duration)

			// Conservative performance threshold
			minScansPerSec := 5.0
			if scansPerSec < minScansPerSec {
				t.Errorf("Concurrent performance too low: %.2f scans/sec < %.2f scans/sec",
					scansPerSec, minScansPerSec)
			}
		})
	}
}

// TestSoakStability runs continuous scans for extended period
func TestSoakStability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

	// Skip unless explicitly enabled for CI safety
	if os.Getenv("PS_RUN_SOAK") != "1" {
		t.Skip("set PS_RUN_SOAK=1 to run soak stability test")
	}

	sc := scanner.ScanEngineCstor(0)

	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "soak-test"},
		Rules: []rules.Rule{
			{ID: "soak1", Level: 1, Keywords: []string{"password", "secret"}, Severity: "HIGH"},
			{ID: "soak2", Level: 2, Patterns: []rules.Pattern{{Regex: `\b\d{4}-\d{4}-\d{4}-\d{4}\b`}}, Severity: "MEDIUM"}, // Credit card
		},
	}})

	content := strings.Repeat("some test content here with password\n", 100)

	// Run for 10 seconds (reduced for CI)
	soakDuration := 10 * time.Second
	start := time.Now()
	end := start.Add(soakDuration)

	var (
		totalScans          int
		totalErrors         int
		totalBytesProcessed int64
	)

	for time.Now().Before(end) {
		_, err := sc.ScanReader(context.Background(), strings.NewReader(content), "soak-test")
		totalScans++
		totalBytesProcessed += int64(len(content))

		if err != nil {
			totalErrors++
			t.Logf("Soak test error at scan %d: %v", totalScans, err)
		}

		// Brief pause to avoid overwhelming the system
		time.Sleep(50 * time.Millisecond)
	}

	actualDuration := time.Since(start)
	scansPerSec := float64(totalScans) / actualDuration.Seconds()
	throughputMBps := float64(totalBytesProcessed) / (1024 * 1024) / actualDuration.Seconds()
	errorRate := float64(totalErrors) / float64(totalScans) * 100

	t.Logf("Soak test results: %d scans in %v", totalScans, actualDuration)
	t.Logf("Performance: %.2f scans/sec, %.2f MB/s throughput", scansPerSec, throughputMBps)
	t.Logf("Reliability: %.2f%% error rate (%d errors)", errorRate, totalErrors)

	// Reliability checks
	if errorRate > 5.0 { // Allow up to 5% error rate for CI stability
		t.Errorf("Too many errors during soak test: %.2f%% error rate", errorRate)
	}

	if totalScans < 10 {
		t.Errorf("Too few scans completed: %d (expected at least 10)", totalScans)
	}
}

// TestLeakDetection uses more aggressive memory tracking
func TestLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping leak detection test in short mode")
	}

	sc := scanner.ScanEngineCstor(0)

	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "leak-test"},
		Rules: []rules.Rule{
			{ID: "leak", Level: 1, Keywords: []string{"test"}, Severity: "INFO"},
		},
	}})

	// Force garbage collection and get baseline - exact pattern from large_bench_test.go
	var m0, m1 runtime.MemStats
	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)

	// Run scans with various content sizes
	for i := 0; i < 50; i++ { // Reduced iterations for CI
		// Vary content size to test different code paths
		size := (i%5 + 1) * 100 // 100 to 500 chars
		content := strings.Repeat("test content line\n", size/18)

		_, err := sc.ScanReader(context.Background(), strings.NewReader(content), "leak-test")
		if err != nil {
			t.Fatalf("scan error: %v", err)
		}

		// Check for goroutine leaks
		if runtime.NumGoroutine() > 50 { // Conservative threshold
			t.Errorf("Too many goroutines: %d (possible goroutine leak)", runtime.NumGoroutine())
		}
	}

	// Force cleanup and measure final memory - exact pattern from large_bench_test.go
	runtime.GC()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m1)

	// Use same delta pattern as large_bench_test.go
	const maxDelta = 10 * 1024 * 1024 // 10MB for smaller test
	if delta := int64(m1.Alloc) - int64(m0.Alloc); delta > maxDelta {
		t.Errorf("possible memory leak: %d bytes still allocated (limit %d)", delta, maxDelta)
	}

	allocDelta := int64(m1.Alloc) - int64(m0.Alloc)
	t.Logf("Memory delta: %d bytes after leak detection test", allocDelta)
}
