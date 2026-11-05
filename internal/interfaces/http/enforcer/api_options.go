package enforcerhttp

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	"github.com/promptshield/promptshield/internal/observability/telemetry"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
	"github.com/promptshield/promptshield/internal/version"
)

// getAPIOptions constructs API options from environment variables
func getAPIOptions() api.Options {
	return getAPIOptionsWithDB(nil)
}

// getAPIOptionsWithDB constructs API options with optional database pool
func getAPIOptionsWithDB(dbPool *pg.Pool) api.Options {
	adminToken := os.Getenv("PS_ENFORCER_ADMIN_TOKEN")

	// Optional insecure mode toggle for local dev / demos
	allowInsecure := false
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_ALLOW_INSECURE_ADMIN"))); v != "" {
		switch v {
		case "1", "true", "yes", "on":
			allowInsecure = true
		}
	}

	// Initialize services with database if available
	var rulepackService *services.RulepackService
	var policyService contracts.PolicyService

	if dbPool != nil {
		// Use PostgreSQL repositories
		rulepackRepo := pg.RulepackRepo(dbPool)
		rulepackService = services.RulepackServiceCstor(rulepackRepo, nil)

		// Use in-memory policy service for fast enforcement
		// Policies are persisted in frontend and synced via API
		policyService = initializePolicyService()
	} else {
		// Use in-memory implementations for high-performance enforcement
		// Policies are persisted in frontend database and synced via API

		// Create in-memory rulepack repository
		rulepackRepo := memory.NewRulepackRepository()
		rulepackService = services.RulepackServiceCstor(rulepackRepo, nil)

		// Use in-memory policy repository
		policyService = initializePolicyService()
	}

	// Create scanner manager for event-driven real-time enforcement
	scannerManager := NewScannerManager()

	telemetryCollector := buildTelemetryCollector()

	return api.Options{
		AdminToken:         adminToken,
		AllowInsecureAdmin: allowInsecure,
		PolicyService:      policyService,
		RulepackService:    rulepackService,
		ScannerManager:     scannerManager,
		Telemetry:          telemetryCollector,
	}
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
		Service:  "ps-enforcer-http",
		Version:  version.Version,
	}

	collector := telemetry.NewCollector(config)
	if err := collector.Initialize(context.Background(), config); err != nil {
		slog.With("component", "telemetry").Warn("Failed to initialize HTTP telemetry", "error", err)
		return nil
	}

	return collector
}

// initializePolicyService creates and configures the policy management service
func initializePolicyService() contracts.PolicyService {
	// Initialize policy dependencies with bootstrap
	policyDeps := bootstrap.InitializePolicyDependencies(
		nil, // ruleCompiler - will be nil for now, add later when integrating with scanner
		nil, // scanEngine - will be nil for now, add later when integrating with scanner
		nil, // auditLogger - will be nil for now, add later when integrating with audit
	)

	return policyDeps.Service
}
