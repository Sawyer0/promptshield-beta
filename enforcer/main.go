package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/license"
	tel "github.com/promptshield/promptshield/internal/observability/telemetry"
	"github.com/promptshield/promptshield/internal/version"
	"google.golang.org/grpc"
)

func main() {
	// Optional GOMAXPROCS override for performance experiments
	if v := strings.TrimSpace(os.Getenv("PS_GOMAXPROCS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			runtime.GOMAXPROCS(n)
		}
	}
	license.Check()
	addr := os.Getenv("PS_ENFORCER_ADDR")
	if addr == "" {
		addr = ":9090"
	}
	// Telemetry init (opt-out, privacy-first)
	var telemetry *tel.Collector
	enabled := true
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_TELEMETRY"))); v == "0" || v == "false" || v == "off" {
		enabled = false
	}
	endpoint := os.Getenv("PS_TELEMETRY_ENDPOINT")
	file := os.Getenv("PS_TELEMETRY_FILE")
	sample := 1.0
	if v := os.Getenv("PS_TELEMETRY_SAMPLE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			sample = f
		}
	}
	if enabled {
		// Default to local NDJSON sink when no remote endpoint/file is configured
		if endpoint == "" && file == "" {
			file = "spans.ndjson"
		}
		telemetry = tel.New(tel.Options{Enabled: true, Endpoint: endpoint, File: file, Sample: sample, Service: "ps-enforcer", Version: version.Version})
		telemetry.Collect("startup", map[string]any{"version": version.Version, "commit": version.Commit, "build_date": version.BuildDate})
	}

	srv := enforcerhttp.Serve(addr)
	log.Printf("http listening on %s", addr)

	// Start gRPC ext_proc server (optional)
	grpcAddr := os.Getenv("PS_ENFORCER_GRPC_ADDR")
	var gs *grpc.Server
	if grpcAddr == "" {
		grpcAddr = ":9091"
	}
	if s, err := grpcenforcer.Build(grpcAddr, grpcenforcer.Options{
		Timeout:         300 * time.Millisecond,
		Telemetry:       telemetry,
		EnforcementMode: os.Getenv("PS_ENFORCER_MODE"),
	}); err == nil {
		log.Printf("grpc ext_proc listening on %s", grpcAddr)
		gs = s
	} else {
		log.Printf("grpc ext_proc startup failed: %v", err)
	}

	// Graceful shutdown on SIGINT/SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	if gs != nil {
		done := make(chan struct{})
		go func() { gs.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			gs.Stop()
		}
	}
	if telemetry != nil {
		if err := telemetry.Shutdown(ctx); err != nil {
			log.Printf("Telemetry shutdown error: %v", err)
		}
	}
}
