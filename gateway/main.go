package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/repository"
	"github.com/promptshield/promptshield/internal/scanner"
	"google.golang.org/grpc"
)

// ps-gateway: Unified PromptShield server with full API + enforcement capabilities
// Features:
// - Security Enforcement: /check, /scan (HTTP + gRPC ext_proc)
// - Management API: /rulepacks, /tenants, /admin/settings, etc.
// - Multi-tenant with database integration
// - Real-time rulepack sync between UI and enforcer
func main() {
	ctx := context.Background()
	logger := slog.With("component", "ps-gateway")

	license.Check()

	// Configuration from environment
	httpAddr := getEnv("PS_ENFORCER_ADDR", "localhost:8098")
	grpcAddr := getEnv("PS_ENFORCER_GRPC_ADDR", "localhost:9091")
	dsn := getEnv("PS_PG_DSN", "")

	if dsn == "" {
		logger.Error("PS_PG_DSN environment variable is required")
		os.Exit(1)
	}

	// Initialize repository factory with automatic environment detection
	logger.Info("Initializing repository factory")
	repoFactory, err := repository.BuildWithFallback(ctx)
	if err != nil {
		logger.Error("Failed to initialize repository factory", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := repoFactory.Close(); closeErr != nil {
			logger.Error("Failed to close repository factory", "error", closeErr)
		}
	}()
	
	// Verify repository factory health
	if err := repoFactory.HealthCheck(ctx); err != nil {
		logger.Warn("Repository factory health check failed, continuing with degraded functionality", "error", err)
	} else {
		logger.Info("Repository factory initialized successfully")
	}

	// Get repositories from factory
	tenantRepo := repoFactory.Tenant()
	assignmentRepo := repoFactory.RulepackAssignment()
	auditRepo := repoFactory.Audit()
	settingsRepo := repoFactory.Settings()

	// Legacy database connection for middleware compatibility
	// TODO: Remove this once all middleware is updated to use repository factory
	db, err := pg.NewPool(ctx, dsn)
	if err != nil {
		logger.Error("Failed to connect to legacy database pool", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize audit event store and hash chain (MANDATORY for security)
	auditEventStore := pg.NewAuditEventStore(db)
	auditHashChain := pg.NewAuditHashChain(auditEventStore)
	auditLogger := pg.NewAuditLogger(auditHashChain)
	logger.Info("Audit hash chain initialized - all events will be cryptographically chained")

	// Initialize NATS publisher (optional)
	natsURL := getEnv("PS_NATS_URL", "")
	publisher, err := nats.NewPublisher(natsURL)
	if err != nil {
		logger.Warn("NATS publisher initialization failed, using no-op publisher", "error", err)
		// Continue without NATS - create no-op publisher
		publisher = &nats.Publisher{} // Use empty struct as no-op
	}
	defer publisher.Close()

	// Initialize services using repository factory
	rulepackSvc := services.RulepackServiceFromFactory(repoFactory, publisher)

	// Initialize scanner for enforcement
	scanEngine := scanner.ScanEngineCstor(0)

	// Configure unified API server options
	apiOptions := api.Options{
		AdminToken:           getEnv("PS_ENFORCER_ADMIN_TOKEN", "admin"),
		AllowInsecureAdmin:   true,
		DB:                   db,          // Unified database connection for tenant validation
		AuditLogger:          auditLogger, // MANDATORY hash-chained audit logging
		RulepackService:      rulepackSvc,
		TenantRepository:     tenantRepo,
		AssignmentRepository: assignmentRepo,
		AuditRepository:      auditRepo,
		SettingsRepository:   settingsRepo,
		Scanner:              scanEngine,
	}

	// Create unified API mux (includes both enforcement and management endpoints)
	mux := api.NewMux(apiOptions)

	// HTTP Server
	httpServer := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	go func() {
		logger.Info("HTTP server starting", "address", httpAddr)
		logger.Info("Available endpoints:")
		logger.Info("  Security: /check, /scan, /healthz, /readyz")
		logger.Info("  API: /rulepacks, /admin/tenants, /admin/settings")
		logger.Info("  Metrics: /metrics, /stats")
		logger.Info("  Business: /api/usage, /api/violations, /api/compliance")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Start gRPC ext_proc server for Envoy integration
	var grpcServer *grpc.Server
	if s, err := grpcenforcer.Build(grpcAddr, grpcenforcer.Options{
		Timeout:         300 * time.Millisecond,
		EnforcementMode: getEnv("PS_ENFORCER_MODE", "observe"),
		RulepackRepo:    repoFactory.Rulepack(), // Enable tenant-aware enforcement using factory
	}); err == nil {
		logger.Info("gRPC ext_proc server starting", "address", grpcAddr)
		grpcServer = s
	} else {
		logger.Error("gRPC ext_proc startup failed", "error", err)
	}

	// Graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	logger.Info("Shutting down servers...")

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	// Shutdown gRPC server
	if grpcServer != nil {
		done := make(chan struct{})
		go func() { grpcServer.GracefulStop(); close(done) }()
		select {
		case <-done:
			logger.Info("gRPC server shutdown complete")
		case <-time.After(5 * time.Second):
			logger.Warn("gRPC server force shutdown")
			grpcServer.Stop()
		}
	}

	logger.Info("Unified ps-gateway shutdown complete")
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
