package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/license"
	tel "github.com/promptshield/promptshield/internal/observability/telemetry"
	"google.golang.org/grpc"
)

func main() {
	license.Check()
	addr := os.Getenv("PS_ENFORCER_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	// Optional telemetry init (opt-in)
	var telemetry *tel.Collector
	enabled := os.Getenv("PS_TELEMETRY") == "1"
	endpoint := os.Getenv("PS_TELEMETRY_ENDPOINT")
	file := os.Getenv("PS_TELEMETRY_FILE")
	sample := 1.0
	if v := os.Getenv("PS_TELEMETRY_SAMPLE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			sample = f
		}
	}
	if enabled && (endpoint != "" || file != "") {
		telemetry = tel.New(tel.Options{Enabled: true, Endpoint: endpoint, File: file, Sample: sample, Service: "ps-enforcer", Version: "0.2.0"})
		telemetry.Collect("startup", map[string]any{"version": "0.2.0", "commit": "dev", "build_date": "unknown"})
	}

	srv := enforcerhttp.Serve(addr)
	log.Printf("http listening on %s", addr)

	// Start gRPC ext_proc server (optional)
	grpcAddr := os.Getenv("PS_ENFORCER_GRPC_ADDR")
	var gs *grpc.Server
	if grpcAddr == "" {
		grpcAddr = ":8081"
	}
	if s, err := grpcenforcer.Run(grpcAddr, grpcenforcer.NewWithOptions(grpcenforcer.Options{
		Timeout:         300 * time.Millisecond,
		Telemetry:       telemetry,
		EnforcementMode: os.Getenv("PS_ENFORCER_MODE"),
	})); err == nil {
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
	_ = srv.Shutdown(ctx)
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
		_ = telemetry.Shutdown(ctx)
	}
}
