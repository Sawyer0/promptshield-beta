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
	"github.com/promptshield/promptshield/internal/interfaces/http/controlplane"
)

func main() {
	ctx := context.Background()
	logger := slog.With("component", "controlplane")

	// Database connection
	dsn := os.Getenv("PS_PG_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost/promptshield?sslmode=disable"
		logger.Warn("PS_PG_DSN not set, using default", "dsn", dsn)
	}

	db, err := pg.NewPool(ctx, dsn)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize repositories
	tenantRepo := pg.TenantRepo(db)
	rulepackRepo := pg.RulepackRepo(db)
	assignmentRepo := pg.RulepackAssignmentRepo(db)
	auditRepo := pg.AuditRepo(db)

	// Initialize NATS publisher
	natsURL := os.Getenv("PS_NATS_URL") // Optional - will be no-op if empty
	publisher, err := nats.NewPublisher(natsURL)
	if err != nil {
		logger.Error("Failed to initialize NATS publisher", "error", err)
		os.Exit(1)
	}
	defer publisher.Close()

	// Initialize services
	rulepackSvc := services.RulepackServiceCstor(rulepackRepo, publisher)
	validationSvc := services.NewValidationService()

	// Initialize HTTP handler
	handler := controlplane.NewControlPlaneHandler(
		tenantRepo,
		rulepackRepo,
		assignmentRepo,
		auditRepo,
		rulepackSvc,
		validationSvc,
		publisher,
	)

	// Setup HTTP router
	mux := controlplane.NewMux(handler)

	// HTTP server setup
	addr := os.Getenv("PS_CONTROL_PLANE_ADDR")
	if addr == "" {
		addr = ":8085"
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("PromptShield Control Plane starting", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited")
}
