package main

import (
	"context"
	"log"
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
	license.Check()

	// Fixed addresses (no flags/env)
	httpAddr := ":9090"
	grpcAddr := ":9091"

	// Start HTTP server (health/metrics/check)
	srv := enforcerhttp.Serve(httpAddr)
	log.Printf("ps-gateway http listening on %s", httpAddr)

	// Start gRPC ext_proc server
	var gs *grpc.Server
	if s, err := grpcenforcer.Build(grpcAddr, grpcenforcer.Options{
		Timeout:         300 * time.Millisecond,
		EnforcementMode: "observe",
	}); err == nil {
		log.Printf("ps-gateway grpc ext_proc listening on %s", grpcAddr)
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
}
