package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/repository"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
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

	// Initialize OpenTelemetry tracing if enabled
	if telemetryEnabled() {
		if tp, shutdown, err := initTracing(ctx); err != nil {
			logger.Warn("OpenTelemetry init failed", "error", err)
		} else {
			defer func() { _ = shutdown(context.Background()) }()
			logger.Info("OpenTelemetry tracing enabled", "endpoint", getEnv("PS_TELEMETRY_ENDPOINT", ""))
			_ = tp // keep for potential future use
		}
	}

	// Configuration from environment
	httpAddr := getEnv("PS_ENFORCER_ADDR", "localhost:8098")
	grpcAddr := getEnv("PS_ENFORCER_GRPC_ADDR", "localhost:9091")
	dsn := getEnv("PS_PG_DSN", "")

	if dsn == "" {
		if iamUser := getEnv("PS_DB_IAM_USER", ""); iamUser != "" {
			writer := getEnv("AURORA_WRITER", "")
			if writer == "" { writer = getEnv("AURORA_PROXY_ENDPOINT", "") }
			dbName := getEnv("AURORA_DB_NAME", "promptshield")
			if writer == "" {
				logger.Error("AURORA_WRITER or AURORA_PROXY_ENDPOINT is required when using PS_DB_IAM_USER for IAM auth")
				os.Exit(1)
			}
			// Build a minimal DSN without credentials; pgx BeforeConnect will inject an IAM token.
			dsn = "postgres://@" + writer + ":5432/" + dbName + "?sslmode=require"
			os.Setenv("PS_PG_DSN", dsn)
			logger.Info("Configured IAM auth for Aurora (token generated on connect)", "endpoint", writer, "db", dbName, "user", iamUser)
		} else {
			logger.Error("PS_PG_DSN is required (or set PS_DB_IAM_USER with AURORA_[WRITER|PROXY_ENDPOINT]/DB)")
			os.Exit(1)
		}
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
		// DB and AuditLogger are omitted; repositories handle DB access via factory
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
		AssignmentRepo:  repoFactory.RulepackAssignment(), // Endpoint-scoped assignment resolution
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

// telemetryEnabled checks PS_TELEMETRY for on/off. Defaults to on if set to 1/true/yes.
func telemetryEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_TELEMETRY")))
	if v == "" {
		// default is enabled if env is present in compose/env.example; treat empty as enabled
		return true
	}
	return v == "1" || v == "true" || v == "yes"
}

// initTracing initializes a global OpenTelemetry TracerProvider with OTLP gRPC exporter.
func initTracing(ctx context.Context) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	endpoint := strings.TrimSpace(getEnv("PS_TELEMETRY_ENDPOINT", ""))
	if endpoint == "" {
		return nil, func(context.Context) error { return nil }, fmt.Errorf("PS_TELEMETRY_ENDPOINT is empty")
	}

exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
if err != nil {
		return nil, nil, err
	}

	// Sampling ratio (0.0 - 1.0)
	sample := 1.0
	if v := strings.TrimSpace(os.Getenv("PS_TRACE_SAMPLE")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			sample = f
		}
	}

	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			semconv.ServiceNameKey.String("promptshield-gateway"),
			semconv.ServiceVersionKey.String(version.Version),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sample))),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))

	shutdown := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := tp.ForceFlush(ctx); err != nil {
			return err
		}
		return tp.Shutdown(ctx)
	}
	return tp, shutdown, nil
}
