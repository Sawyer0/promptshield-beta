package main

import (
	"context"
	"log"
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

	// Database connection
	dsn := os.Getenv("PS_PG_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost/promptshield?sslmode=disable"
		log.Printf("PS_PG_DSN not set, using default: %s", dsn)
	}

	db, err := pg.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	tenantRepo := pg.TenantRepo(db)
	rulepackRepo := pg.RulepackRepo(db)
	assignmentRepo := pg.PolicyAssignmentRepo(db)
	auditRepo := pg.AuditRepo(db)

	// Initialize NATS publisher
	natsURL := os.Getenv("PS_NATS_URL") // Optional - will be no-op if empty
	publisher, err := nats.NewPublisher(natsURL)
	if err != nil {
		log.Fatalf("Failed to initialize NATS publisher: %v", err)
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
		log.Printf("PromptShield Control Plane starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
