//go:build toolsperf
// +build toolsperf

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"

	_ "net/http/pprof"
)

// extproc_local starts a gRPC ExternalProcessor on :9091 for Envoy integration tests.
// It loads rules from PS_ENFORCER_RULEPACK (file path) or runs with empty rules (fail-open).
func main() {
	addr := getenv("PERF_GRPC_ADDR", ":9091")
	pprofAddr := getenv("PPROF_ADDR", "localhost:6061")

	// Start pprof HTTP server (debug endpoints) on separate listener
	go func() {
		log.Printf("extproc pprof listening on http://%s/debug/pprof/", pprofAddr)
		_ = http.ListenAndServe(pprofAddr, nil)
	}()

	// Relax audit requirement for local runs
	_ = os.Setenv("PS_AUDIT_REQUIRED", "false")

	// Optional: enable fake L3 analyzer
	// PS_FAKE_L3=true
	// PS_FAKE_L3_DELAY_MS=30

	// Default stream window override for perf tuning
	if os.Getenv("PS_ENFORCER_STREAM_WINDOW") == "" {
		_ = os.Setenv("PS_ENFORCER_STREAM_WINDOW", "20480")
	}
	opts := grpcenforcer.Options{
		Timeout:         600 * time.Millisecond,
		RulepackPath:    os.Getenv("PS_ENFORCER_RULEPACK"),
		EnforcementMode: getenv("PS_ENFORCER_MODE", "observe"),
	}
	if _, err := grpcenforcer.Build(addr, opts); err != nil {
		log.Fatalf("ext_proc build failed: %v", err)
	}
	log.Printf("ext-proc gRPC listening on %s (rulepack=%s)", addr, opts.RulepackPath)
	select {} // block forever
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

