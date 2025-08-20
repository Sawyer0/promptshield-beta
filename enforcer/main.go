package main

import (
	"context"
	"log/slog"
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
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
	"github.com/promptshield/promptshield/internal/version"
	"google.golang.org/grpc"
)

func main() {
	// Initialize structured logging
	logger := slog.With("component", "enforcer")
	
	// Optional GOMAXPROCS override for performance experiments
	if v := strings.TrimSpace(os.Getenv("PS_GOMAXPROCS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			runtime.GOMAXPROCS(n)
			logger.Info("GOMAXPROCS override", "value", n)
		}
	}
	license.Check()
	addr := os.Getenv("PS_ENFORCER_ADDR")
	if addr == "" {
		addr = ":9090"
	}
	// Telemetry init (opt-out, privacy-first)
	var telemetry contracts.TelemetryCollector
	var grpcTelemetry grpcenforcer.TelemetryCollector
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
		config := &types.TelemetryConfig{
			Enabled:  true,
			Endpoint: endpoint,
			File:     file,
			Sample:   sample,
			Service:  "ps-enforcer",
			Version:  version.Version,
		}
		telemetry = tel.NewCollector(config)
		if err := telemetry.Initialize(context.Background(), config); err != nil {
			logger.Warn("Failed to initialize telemetry", "error", err)
		}
		// Create gRPC telemetry adapter
		grpcTelemetry = &grpcTelemetryAdapter{telemetry: telemetry}
		// Record startup event
		startupEvent := &types.TelemetryEvent{
			Type:      "startup",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"version":    version.Version,
				"commit":     version.Commit,
				"build_date": version.BuildDate,
			},
		}
		_ = telemetry.RecordEvent(context.Background(), startupEvent)
	}

	srv := enforcerhttp.Serve(addr)
	logger.Info("HTTP server starting", "address", addr)

	// Start gRPC ext_proc server (optional)
	grpcAddr := os.Getenv("PS_ENFORCER_GRPC_ADDR")
	var gs *grpc.Server
	if grpcAddr == "" {
		grpcAddr = ":9091"
	}
	if s, err := grpcenforcer.Build(grpcAddr, grpcenforcer.Options{
		Timeout:         300 * time.Millisecond,
		Telemetry:       grpcTelemetry,
		EnforcementMode: os.Getenv("PS_ENFORCER_MODE"),
	}); err == nil {
		logger.Info("gRPC ext_proc server starting", "address", grpcAddr)
		gs = s
	} else {
		logger.Error("gRPC ext_proc startup failed", "error", err)
	}

	// Graceful shutdown on SIGINT/SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger.Info("Shutting down servers...")
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
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
		if err := telemetry.Close(); err != nil {
			logger.Error("Telemetry shutdown error", "error", err)
		}
	}
	logger.Info("Shutdown complete")
}

// grpcTelemetryAdapter adapts contracts.TelemetryCollector to grpcenforcer.TelemetryCollector
type grpcTelemetryAdapter struct {
	telemetry contracts.TelemetryCollector
}

func (a *grpcTelemetryAdapter) Collect(eventType string, payload map[string]any) {
	if a.telemetry == nil {
		return
	}
	event := &types.TelemetryEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	_ = a.telemetry.RecordEvent(context.Background(), event)
}
