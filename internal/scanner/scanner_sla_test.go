package scanner_test

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
)

// TestScannerThroughput_SLA enforces minimum scanner throughput on a 1MB synthetic stream.
// Skipped unless PS_ENFORCE_SLA=1 to avoid CI/machine variability.
func TestScannerThroughput_SLA(t *testing.T) {
	if os.Getenv("PS_ENFORCE_SLA") != "1" {
		t.Skip("set PS_ENFORCE_SLA=1 to enforce scanner throughput SLA")
	}
	minMBps := 200.0 // can override via PS_SLA_SCANNER_MBPS_MIN
	if v := os.Getenv("PS_SLA_SCANNER_MBPS_MIN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			minMBps = f
		}
	}
	sc := scanner.ScanEngineCstor(0)
	sc.LoadRulePacks([]rules.RulePack{{Metadata: rules.Metadata{Name: "sla"}, Rules: []rules.Rule{{ID: "kw", Level: 1, Severity: "INFO", Keywords: []string{"secret"}}}}})
	// 1MB buffer with scattered tokens
	buf := make([]byte, 1<<20)
	for i := 0; i < len(buf); i += 1000 {
		buf[i] = 's'
	}
	start := time.Now()
	if _, err := sc.ScanReader(context.Background(), bytes.NewReader(buf), "sla"); err != nil {
		t.Fatalf("scan error: %v", err)
	}
	dur := time.Since(start).Seconds()
	mbps := (float64(len(buf)) / (1024 * 1024)) / dur
	if mbps < minMBps {
		t.Fatalf("scanner throughput %.2f MB/s below SLA (min %.2f MB/s)", mbps, minMBps)
	}
}
