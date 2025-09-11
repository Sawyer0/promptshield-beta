package enforcerhttp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	semdeberta "github.com/promptshield/promptshield/internal/semantic/deberta"
	semfake "github.com/promptshield/promptshield/internal/semantic/fake"
	ckeys "github.com/promptshield/promptshield/internal/shared/contextkeys"
	pkg "github.com/promptshield/promptshield/pkg/types"
	"gopkg.in/yaml.v3"
)

// ScannerManager manages scanners for real-time enforcement per tenant.
// It loads rulepacks from the database and manages scanner state
// for each tenant independently.
type ScannerManager struct {
	mu              sync.RWMutex
	logger          *slog.Logger
	rulepackService *services.RulepackService
	db              *postgres.Pool

	// Global semantic analyzer (omni/custom/composite)
	analyzer scanner.SemanticAnalyzer

	// Per-tenant scanners cache
	tenantScanners map[uuid.UUID]*scanner.Scanner
	// Optional simple TTL per tenant to refresh packs periodically
	lastLoaded map[uuid.UUID]time.Time
	cacheTTL   time.Duration
}

// NewScannerManager creates a new scanner manager with database-backed rulepacks
func NewScannerManager() *ScannerManager {
	return NewScannerManagerWithRulepackService(nil, nil)
}

// NewScannerManagerWithRulepackService creates a new scanner manager with rulepack service
func NewScannerManagerWithRulepackService(rulepackService *services.RulepackService, db *postgres.Pool) *ScannerManager {
	manager := &ScannerManager{
		rulepackService: rulepackService,
		logger:          slog.With("component", "scanner-manager"),
		db:              db,
		tenantScanners:  make(map[uuid.UUID]*scanner.Scanner),
		lastLoaded:      make(map[uuid.UUID]time.Time),
		cacheTTL:        60 * time.Second,
	}

	// Initialize semantic analyzer: DeBERTa (ProtectAI) by default.
	// Local fake L3 analyzer for deterministic testing when PS_FAKE_L3 is enabled
	if v := strings.ToLower(strings.TrimSpace(getenv("PS_FAKE_L3"))); v == "1" || v == "true" || v == "yes" {
		var d time.Duration
		if ms := strings.TrimSpace(getenv("PS_FAKE_L3_DELAY_MS")); ms != "" {
			if n, err := strconv.Atoi(ms); err == nil && n > 0 {
				d = time.Duration(n) * time.Millisecond
			}
		}
		manager.analyzer = semfake.Analyzer{Delay: d}
	} else {
		endpoint := getenv("PS_DEBERTA_ENDPOINT")
		apiKey := getenv("HF_TOKEN") // optional; not required for local servers
		if endpoint != "" {
			manager.analyzer = semdeberta.New(semdeberta.Options{Endpoint: endpoint, APIKey: apiKey})
		} else {
			// As a safe default, no analyzer configured; L3 will be inert until endpoint is provided
			manager.logger.Warn("No PS_DEBERTA_ENDPOINT configured; Level-3 semantic analysis disabled")
		}
	}

	manager.logger.Info("Scanner manager initialized (tenant-aware)")
	return manager
}

func getenv(k string) string { return strings.TrimSpace(sysgetenv(k)) }

// sysgetenv isolated for testing/mocking
func sysgetenv(k string) string { return os.Getenv(k) }

// GetScanner returns a default scanner (not tenant-scoped).
// Prefer ScanReader which is tenant-aware.
func (sm *ScannerManager) GetScanner() *scanner.Scanner {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	// Provide a fresh minimal scanner for non-tenant paths
	sc := scanner.ScanEngineCstor(0)
	if sm.analyzer != nil {
		sc.SetSemanticAnalyzer(sm.analyzer)
	}
	return sc
}

// HasActivePolicies returns true when manager can serve scans (service present).
func (sm *ScannerManager) HasActivePolicies() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.rulepackService != nil
}

// GetActiveRulepackCount is not meaningful in tenant mode; return 0 (unused).
func (sm *ScannerManager) GetActiveRulepackCount() int { return 0 }

// ReloadRulepacks clears tenant scanner cache so next scan reloads fresh packs.
func (sm *ScannerManager) ReloadRulepacks() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.tenantScanners = make(map[uuid.UUID]*scanner.Scanner)
	sm.lastLoaded = make(map[uuid.UUID]time.Time)
	return nil
}

// ScanReader uses a tenant-specific scanner based on tenant id in context.
func (sm *ScannerManager) ScanReader(ctx context.Context, reader interface{}, inputName string) (pkg.ScanResult, error) {
	// Determine tenant id from context
	var tenantID uuid.UUID
	if v := ctx.Value(ckeys.TenantID); v != nil {
		if s, ok := v.(string); ok {
			if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
				tenantID = id
			}
		}
	}
	if tenantID == uuid.Nil {
		if v := ctx.Value("tenant_id"); v != nil {
			if s, ok := v.(string); ok {
				if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
					tenantID = id
				}
			}
		}
	}

	sm.mu.RLock()
	an := sm.analyzer
	sm.mu.RUnlock()

	// Choose scanner: tenant-specific when tenantID present; else default
	var sc *scanner.Scanner
	if tenantID != uuid.Nil {
		var err error
		sc, err = sm.getOrLoadTenantScanner(ctx, tenantID)
		if err != nil {
			return pkg.ScanResult{}, err
		}
	} else {
		sc = scanner.ScanEngineCstor(0)
		if an != nil {
			sc.SetSemanticAnalyzer(an)
		}
	}

	// Ensure base context (tenant/tracing) is available to semantic analyzers
	sc.SetBaseContext(ctx)

	// io.Reader check
	if r, ok := reader.(interface{ Read([]byte) (int, error) }); ok {
		result, err := sc.ScanReader(ctx, r, inputName)
		return result, err
	}
	return pkg.ScanResult{}, fmt.Errorf("invalid reader type")
}

// getOrLoadTenantScanner returns a cached scanner for tenant or loads rulepacks.
func (sm *ScannerManager) getOrLoadTenantScanner(ctx context.Context, tenantID uuid.UUID) (*scanner.Scanner, error) {
	sm.mu.RLock()
	if sc, ok := sm.tenantScanners[tenantID]; ok {
		// Simple TTL refresh
		if time.Since(sm.lastLoaded[tenantID]) < sm.cacheTTL {
			sm.mu.RUnlock()
			return sc, nil
		}
	}
	sm.mu.RUnlock()

	if sm.rulepackService == nil {
		// No service; return empty scanner
		sc := scanner.ScanEngineCstor(0)
		if sm.analyzer != nil {
			sc.SetSemanticAnalyzer(sm.analyzer)
		}
		return sc, nil
	}

	// Load active rulepacks for tenant
	packs, err := sm.loadTenantPacks(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	sc := scanner.ScanEngineCstor(0)
	if sm.analyzer != nil {
		sc.SetSemanticAnalyzer(sm.analyzer)
	}
	if len(packs) > 0 {
		sc.LoadRulePacks(packs)
	}

	sm.mu.Lock()
	sm.tenantScanners[tenantID] = sc
	sm.lastLoaded[tenantID] = time.Now()
	sm.mu.Unlock()
	return sc, nil
}

func (sm *ScannerManager) loadTenantPacks(ctx context.Context, tenantID uuid.UUID) ([]rules.RulePack, error) {
	infos, err := sm.rulepackService.List(ctx, tenantID)
	if err != nil {
		sm.logger.Warn("Failed to list rulepacks", "tenant_id", tenantID, "error", err)
		return nil, err
	}
	var activePacks []rules.RulePack
	for _, rp := range infos {
		if !rp.Active {
			continue
		}
		dsl, _, err := sm.rulepackService.GetActive(ctx, rp.ID)
		if err != nil {
			sm.logger.Warn("Failed to get active rulepack DSL", "rulepack_id", rp.ID, "error", err)
			continue
		}
		var pack rules.RulePack
		if err := yaml.Unmarshal(dsl, &pack); err != nil {
			sm.logger.Warn("Failed to parse rulepack DSL", "rulepack_id", rp.ID, "error", err)
			continue
		}
		activePacks = append(activePacks, pack)
	}
	return activePacks, nil
}
