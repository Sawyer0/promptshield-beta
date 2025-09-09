package api

import (
	"context"
	"net/http"
	"time"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/observability/telemetry"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/usage"
	pkg "github.com/promptshield/promptshield/pkg/types"
)

// Options configures the API mux.
type Options struct {
	AdminToken         string
	AllowInsecureAdmin bool
	ConfigStore        *RuntimeConfigStore
	RulepackService    *services.RulepackService
	PolicyService      contracts.PolicyService
	UsageStore         usage.UsageStore

	// Multi-tenant support
	DB postgres.DB // Database interface for tenant validation
	// AuditLogger, when set, receives durable audit trail events
	AuditLogger contracts.AuditLogger
	// Events provides a simple in-memory broadcaster for SSE and hooks.
	// When nil, a default hub is created by the router.
	Events *EventHub
	// OnDrain is called when /v1/admin/drain is invoked. Optional.
	OnDrain func(ctx context.Context) error
	// OnShutdown is called when /v1/admin/shutdown is invoked. Optional.
	// Implementations should gracefully stop the server after the given delay.
	OnShutdown func(ctx context.Context, delay time.Duration) error
	// QuotaStore enables per-tenant rate limiting when set.
	QuotaStore usage.QuotaStore
	// Security Gateway uses simple token auth only
	// Telemetry provides OpenTelemetry tracing and metrics collection
	Telemetry *telemetry.Collector

	// Enterprise repositories for tenant management
	TenantRepository     domain.TenantRepository
	AssignmentRepository domain.RulepackAssignmentRepository // rulepack assignments
	AuditRepository      domain.AuditRepository
	SettingsRepository   domain.SettingsRepository
	// Security Gateway - no provider key management needed
	// Security Gateway - no complex quota management needed

	// Security Gateway - no routing needed

	// Provider API keys management (deprecated - use ProviderKeyStore)
	ProviderKeys map[string]string // provider -> encrypted key

	// Tool runner (BYOK and internal tools)
	ToolRunner contracts.ToolRunner

	// Security scanning components
	Scanner   *scanner.Scanner  // Core scanning engine for policy enforcement
	RulePacks []*rules.RulePack // Active rule packs for scanning
	// ScannerManager handles event-driven real-time enforcement scanning
	ScannerManager interface {
		HasActivePolicies() bool
		ScanReader(ctx context.Context, reader interface{}, inputName string) (pkg.ScanResult, error)
		ReloadRulepacks() error
	}

    // PlanState for Plan-Then-Execute and Dual-LLM lane tokens
    PlanState contracts.PlanState
}

// Deprecated: Use writeErrorJSON instead
func writeError(w http.ResponseWriter, status int, code, msg string, details map[string]any) {
	writeErrorJSON(w, status, code, msg, details, nil)
}

func versionHeader(v string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-PS-API-Version", v)
			next.ServeHTTP(w, r)
		})
	}
}
