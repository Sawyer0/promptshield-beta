package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/license"
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
	srv := enforcerhttp.Serve(httpAddr)
	logger.Info("ps-gateway http server starting", "address", httpAddr)

	// Start gRPC ext_proc server
	var gs *grpc.Server
	if s, err := grpcenforcer.Build(grpcAddr, grpcenforcer.Options{
		Timeout:         300 * time.Millisecond,
		EnforcementMode: "observe",
	}); err == nil {
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
