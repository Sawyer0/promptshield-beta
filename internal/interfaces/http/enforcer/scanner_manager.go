package enforcerhttp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	enc "github.com/promptshield/promptshield/internal/security/crypto"
	semcomp "github.com/promptshield/promptshield/internal/semantic/composite"
	semcustom "github.com/promptshield/promptshield/internal/semantic/custom"
	semopenai "github.com/promptshield/promptshield/internal/semantic/openai"
	pkg "github.com/promptshield/promptshield/pkg/types"
	"gopkg.in/yaml.v3"
)

// ScannerManager manages a scanner instance for real-time enforcement
// It loads rulepacks from the database and manages scanner state
type ScannerManager struct {
	mu              sync.RWMutex
	scanner         *scanner.Scanner
	loadedRulepacks int // Track number of loaded rulepacks
	rulepackService *services.RulepackService
	logger          *slog.Logger
	db              *postgres.Pool
}

// NewScannerManager creates a new scanner manager with database-backed rulepacks
func NewScannerManager() *ScannerManager {
	return NewScannerManagerWithRulepackService(nil, nil)
}

// NewScannerManagerWithRulepackService creates a new scanner manager with rulepack service
func NewScannerManagerWithRulepackService(rulepackService *services.RulepackService, db *postgres.Pool) *ScannerManager {
	sc := scanner.ScanEngineCstor(0)
	// Configure scanner for production use
	sc.SetQuarantineOnTimeout(true)
	sc.SetQuarantineOnError(true)
	sc.SetMaxStreamBytes(10 * 1024 * 1024) // 10MB max

	manager := &ScannerManager{
		scanner:         sc,
		rulepackService: rulepackService,
		logger:          slog.With("component", "scanner-manager"),
		db:              db,
	}

	// Load any active rulepacks from database at startup
	manager.loadActiveRulepacksFromDatabase()

	manager.logger.Info("Scanner manager initialized with database-backed rulepacks")

	// Initialize semantic analyzers (omni + optional custom composite)
	var omni scanner.SemanticAnalyzer
	if apiKey := getenv("OPENAI_API_KEY"); apiKey != "" {
		omni = semopenai.New(semopenai.Options{APIKey: apiKey})
	}

	var custom scanner.SemanticAnalyzer
	if db != nil {
		resolver := func(ctx context.Context, cfg rules.Semantic) (string, string, error) {
			// tenant id from context (set by middleware)
			var tenant string
			if v := ctx.Value("tenant.id"); v != nil {
				if s, ok := v.(string); ok {
					tenant = s
				}
			}
			if tenant == "" {
				if v := ctx.Value("tenant_id"); v != nil {
					if s, ok := v.(string); ok {
						tenant = s
					}
				}
			}
			if tenant == "" {
				return "", "", fmt.Errorf("missing tenant in context")
			}
			if strings.TrimSpace(cfg.ProviderProfile) == "" {
				return "", "", fmt.Errorf("missing provider_profile")
			}
			// Query provider_profiles
			row := db.Raw().QueryRow(ctx, `SELECT api_key_encrypted, COALESCE(base_url,'') FROM provider_profiles WHERE tenant_id=$1 AND id=$2`, tenant, cfg.ProviderProfile)
			var encKey, baseURL string
			if err := row.Scan(&encKey, &baseURL); err != nil {
				return "", "", err
			}
			key, err := enc.DecryptString(encKey)
			if err != nil {
				return "", "", err
			}
			return key, baseURL, nil
		}
		custom = semcustom.New(resolver)
	}

	if omni != nil || custom != nil {
		var analyzer scanner.SemanticAnalyzer
		if omni != nil && custom != nil {
			analyzer = semcomp.New(omni, custom)
		} else if custom != nil {
			analyzer = custom
		} else {
			analyzer = omni
		}
		manager.scanner.SetSemanticAnalyzer(analyzer)
	}

	return manager
}

func getenv(k string) string { return strings.TrimSpace(sysgetenv(k)) }

// sysgetenv isolated for testing/mocking
func sysgetenv(k string) string { return os.Getenv(k) }

// GetScanner returns the scanner instance for use by enforcement handlers
func (sm *ScannerManager) GetScanner() *scanner.Scanner {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.scanner
}

// HasActivePolicies returns true if any rulepacks are currently loaded
func (sm *ScannerManager) HasActivePolicies() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.loadedRulepacks > 0
}

// GetActiveRulepackCount returns the number of loaded rulepacks
func (sm *ScannerManager) GetActiveRulepackCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.loadedRulepacks
}

// ReloadRulepacks forces a reload of rulepacks from the database
// This can be called after rulepacks are uploaded via API
func (sm *ScannerManager) ReloadRulepacks() error {
	sm.loadActiveRulepacksFromDatabase()
	return nil
}

// ScanReader provides a wrapper around the scanner's ScanReader method
// This allows the enforcement handler to use the manager's scanner directly
func (sm *ScannerManager) ScanReader(ctx context.Context, reader interface{}, inputName string) (pkg.ScanResult, error) {
	sm.mu.RLock()
	scanner := sm.scanner
	rulepackCount := sm.loadedRulepacks
	sm.mu.RUnlock()

	sm.logger.Debug("Scanning with database-loaded rulepacks", "loaded_rulepacks", rulepackCount, "input_name", inputName)

	// Ensure base context (tenant/tracing) is available to semantic analyzers
	scanner.SetBaseContext(ctx)

	// Convert reader to the expected type and call scanner
	// The scanner expects an io.Reader
	if r, ok := reader.(interface{ Read([]byte) (int, error) }); ok {
		result, err := scanner.ScanReader(ctx, r, inputName)
		sm.logger.Debug("Scanner result", "violations", len(result.Violations), "error", err)
		return result, err
	}

	return pkg.ScanResult{}, fmt.Errorf("invalid reader type")
}

// loadActiveRulepacksFromDatabase loads any active rulepacks from the database
// This is the only source of rulepacks - no file fallback
func (sm *ScannerManager) loadActiveRulepacksFromDatabase() {
	if sm.rulepackService == nil {
		sm.logger.Info("No rulepack service available - scanner will be empty until rulepacks are loaded via API")
		return
	}

	ctx := context.Background()
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001") // Default tenant

	rulepacks, err := sm.rulepackService.List(ctx, tenantID)
	if err != nil {
		sm.logger.Warn("Failed to load rulepacks from database", "error", err)
		return
	}

	var activePacks []rules.RulePack
	for _, rp := range rulepacks {
		if rp.Active {
			// Get the active version DSL
			if dsl, _, err := sm.rulepackService.GetActive(ctx, rp.ID); err == nil {
				var pack rules.RulePack
				if err := yaml.Unmarshal(dsl, &pack); err == nil {
					activePacks = append(activePacks, pack)
					sm.logger.Info("Loaded active rulepack from database",
						"rulepack_id", rp.ID, "name", rp.Name, "version", rp.Version, "rules_count", len(pack.Rules))
				} else {
					sm.logger.Warn("Failed to parse rulepack DSL", "rulepack_id", rp.ID, "error", err)
				}
			} else {
				sm.logger.Warn("Failed to get active rulepack DSL", "rulepack_id", rp.ID, "error", err)
			}
		}
	}

	sm.mu.Lock()
	if len(activePacks) > 0 {
		sm.scanner.LoadRulePacks(activePacks)
		sm.loadedRulepacks = len(activePacks)
		sm.logger.Info("Successfully loaded rulepacks from database", "count", len(activePacks))
	} else {
		sm.loadedRulepacks = 0
		sm.logger.Info("No active rulepacks found in database")
	}
	sm.mu.Unlock()
}
