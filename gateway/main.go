package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/observability/telemetry"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
	"github.com/promptshield/promptshield/internal/version"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
)

// ps-gateway: minimal Envoy gateway without CLI or flags
// - HTTP: /healthz, /readyz, /metrics, /check
// - gRPC: ext_proc ExternalProcessor
func main() {
	// Initialize structured logging
	logger := slog.With("component", "ps-gateway")

	license.Check()

	// Fixed addresses (no flags/env)
	httpAddr := ":9090"
	grpcAddr := ":9091"

	// Start HTTP server (health/metrics/check)
	telemetryCollector := buildTelemetryCollector()
	srv := enforcerhttp.Serve(httpAddr)
	logger.Info("ps-gateway http server starting", "address", httpAddr)
	if telemetryCollector != nil {
		srv.Handler = otelhttp.NewHandler(srv.Handler, "ps_gateway_http", otelhttp.WithFilter(func(r *http.Request) bool {
			return r.URL.Path != "/healthz"
		}))
	}

	// Start gRPC ext_proc server
	var gs *grpc.Server
	grpcOpts := []grpc.ServerOption{}
	var grpcTelemetry grpcenforcer.TelemetryCollector
	if telemetryCollector != nil {
		grpcOpts = append(grpcOpts,
			grpc.StatsHandler(otelgrpc.NewServerHandler()),
		)
		grpcTelemetry = &grpcTelemetryAdapter{telemetry: telemetryCollector}
	}

	if s, err := grpcenforcer.Build(grpcAddr, grpcenforcer.Options{
		Timeout:         300 * time.Millisecond,
		EnforcementMode: "observe",
		Telemetry:       grpcTelemetry,
	}, grpcOpts...); err == nil {
		logger.Info("ps-gateway grpc ext_proc server starting", "address", grpcAddr)
		gs = s
	} else {
		logger.Error("grpc ext_proc startup failed", "error", err)
	}

	// Graceful shutdown on SIGINT/SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	logger.Info("Shutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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
	logger.Info("Shutdown complete")
}

func buildTelemetryCollector() *telemetry.Collector {
	if strings.EqualFold(os.Getenv("PS_TELEMETRY"), "false") || os.Getenv("PS_TELEMETRY") == "0" {
		return nil
	}

	endpoint := os.Getenv("PS_TELEMETRY_ENDPOINT")
	if strings.TrimSpace(endpoint) == "" {
		return nil
	}

	sample := 1.0
	if v := os.Getenv("PS_TELEMETRY_SAMPLE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			sample = f
		}
	}

	config := &types.TelemetryConfig{
		Enabled:  true,
		Endpoint: endpoint,
		Sample:   sample,
		Service:  "ps-gateway",
		Version:  version.Version,
	}

	collector := telemetry.NewCollector(config)
	if err := collector.Initialize(context.Background(), config); err != nil {
		slog.With("component", "telemetry").Warn("Failed to initialize gateway telemetry", "error", err)
		return nil
	}

	return collector
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
