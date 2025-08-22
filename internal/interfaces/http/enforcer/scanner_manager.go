package enforcerhttp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"gopkg.in/yaml.v3"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/shared/events"
	"github.com/promptshield/promptshield/internal/shared/types"
	pkg "github.com/promptshield/promptshield/pkg/types"
)

// ScannerManager manages a scanner instance for real-time enforcement
// It subscribes to policy events and updates the scanner accordingly
type ScannerManager struct {
	mu             sync.RWMutex
	scanner        *scanner.Scanner
	activePolicies map[uuid.UUID]*types.Policy
	logger         *slog.Logger
}

// NewScannerManager creates a new scanner manager and subscribes to policy events
func NewScannerManager() *ScannerManager {
	sc := scanner.ScanEngineCstor(0)
	// Configure scanner for production use
	sc.SetQuarantineOnTimeout(true)
	sc.SetQuarantineOnError(true)
	sc.SetMaxStreamBytes(10 * 1024 * 1024) // 10MB max
	
	manager := &ScannerManager{
		scanner:        sc,
		activePolicies: make(map[uuid.UUID]*types.Policy),
		logger:         slog.With("component", "scanner-manager"),
	}
	
	// Subscribe to policy activation/deactivation events
	events.GlobalEventBus().Subscribe(events.EventTypePolicyActivated, manager.handlePolicyActivated)
	events.GlobalEventBus().Subscribe(events.EventTypePolicyDeactivated, manager.handlePolicyDeactivated)
	
	manager.logger.Info("Scanner manager initialized and subscribed to policy events")
	
	return manager
}

// GetScanner returns the scanner instance for use by enforcement handlers
func (sm *ScannerManager) GetScanner() *scanner.Scanner {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.scanner
}

// HasActivePolicies returns true if any policies are currently active
func (sm *ScannerManager) HasActivePolicies() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.activePolicies) > 0
}

// GetActivePolicyCount returns the number of active policies
func (sm *ScannerManager) GetActivePolicyCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.activePolicies)
}

// handlePolicyActivated processes policy activation events
func (sm *ScannerManager) handlePolicyActivated(ctx context.Context, event events.Event) error {
	activatedEvent, ok := event.(*events.PolicyActivated)
	if !ok {
		return fmt.Errorf("invalid event type for policy activation")
	}
	
	sm.logger.Info("Processing policy activation event", 
		"policy_id", activatedEvent.PolicyID, 
		"policy_name", activatedEvent.PolicyData.Name)
	
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Add policy to active set
	sm.activePolicies[activatedEvent.PolicyID] = &activatedEvent.PolicyData
	
	// Reload scanner with all active policies
	if err := sm.reloadScannerLocked(); err != nil {
		sm.logger.Error("Failed to reload scanner after policy activation", "error", err)
		return err
	}
	
	sm.logger.Info("Policy activated in enforcement scanner", 
		"policy_id", activatedEvent.PolicyID,
		"total_active", len(sm.activePolicies))
	
	return nil
}

// handlePolicyDeactivated processes policy deactivation events
func (sm *ScannerManager) handlePolicyDeactivated(ctx context.Context, event events.Event) error {
	deactivatedEvent, ok := event.(*events.PolicyDeactivated)
	if !ok {
		return fmt.Errorf("invalid event type for policy deactivation")
	}
	
	sm.logger.Info("Processing policy deactivation event", 
		"policy_id", deactivatedEvent.PolicyID, 
		"policy_name", deactivatedEvent.PolicyData.Name)
	
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Remove policy from active set
	delete(sm.activePolicies, deactivatedEvent.PolicyID)
	
	// Reload scanner with remaining active policies
	if err := sm.reloadScannerLocked(); err != nil {
		sm.logger.Error("Failed to reload scanner after policy deactivation", "error", err)
		return err
	}
	
	sm.logger.Info("Policy deactivated in enforcement scanner", 
		"policy_id", deactivatedEvent.PolicyID,
		"total_active", len(sm.activePolicies))
	
	return nil
}

// reloadScannerLocked rebuilds and reloads the scanner (must be called with lock held)
func (sm *ScannerManager) reloadScannerLocked() error {
	var rulePacks []rules.RulePack
	
	for policyID, policy := range sm.activePolicies {
		rulepack, err := sm.convertPolicyToRulePack(policy)
		if err != nil {
			sm.logger.Error("Failed to convert policy to rulepack", "policy_id", policyID, "error", err)
			continue
		}
		sm.logger.Debug("Converted policy to rulepack", "policy_id", policyID, "rulepack_name", rulepack.Metadata.Name, "rules_count", len(rulepack.Rules))
		for i, rule := range rulepack.Rules {
			sm.logger.Debug("Rule details", "rule_index", i, "rule_id", rule.ID, "rule_level", rule.Level, "keywords", rule.Keywords)
		}
		rulePacks = append(rulePacks, *rulepack)
	}
	
	sm.logger.Info("Reloading scanner", "rulepack_count", len(rulePacks))
	
	// Reload scanner with new RulePacks
	sm.scanner.LoadRulePacks(rulePacks)
	return nil
}

// convertPolicyToRulePack converts a Policy to a RulePack
// This is a copy of the logic from PolicyScannerService to avoid dependencies
func (sm *ScannerManager) convertPolicyToRulePack(policy *types.Policy) (*rules.RulePack, error) {
	// Try to parse the policy content as YAML first
	var rulepack rules.RulePack
	if err := yaml.Unmarshal([]byte(policy.Content), &rulepack); err == nil {
		// Successfully parsed as RulePack YAML
		if rulepack.Metadata.Name == "" {
			rulepack.Metadata.Name = policy.Name
		}
		if rulepack.APIVersion == "" {
			rulepack.APIVersion = "promptshield.io/v1"
		}
		if rulepack.Kind == "" {
			rulepack.Kind = "RulePack"
		}
		return &rulepack, nil
	}
	
	// If not valid RulePack YAML, try to parse as simple rules structure
	var simpleRules struct {
		Rules []rules.Rule `yaml:"rules" json:"rules"`
	}
	
	if err := yaml.Unmarshal([]byte(policy.Content), &simpleRules); err == nil && len(simpleRules.Rules) > 0 {
		// Create RulePack from simple rules
		rulepack = rules.RulePack{
			APIVersion: "promptshield.io/v1",
			Kind:       "RulePack",
			Metadata: rules.Metadata{
				Name:        policy.Name,
				Description: fmt.Sprintf("Policy %s", policy.Name),
				Version:     fmt.Sprintf("v%d", policy.Version),
			},
			Rules: simpleRules.Rules,
		}
		return &rulepack, nil
	}
	
	// If neither format works, create a simple keyword-based rule
	// This handles cases where policy content is just text or simple format
	rulepack = rules.RulePack{
		APIVersion: "promptshield.io/v1",
		Kind:       "RulePack",
		Metadata: rules.Metadata{
			Name:        policy.Name,
			Description: fmt.Sprintf("Policy %s", policy.Name),
			Version:     fmt.Sprintf("v%d", policy.Version),
		},
		Rules: []rules.Rule{
			{
				ID:       fmt.Sprintf("policy-%s", policy.ID.String()[:8]),
				Name:     policy.Name,
				Level:    1, // keyword level
				Severity: "MEDIUM",
				Keywords: []string{"inject", "ignore", "override"}, // default security keywords
				Response: &rules.Response{
					Action:  "warn",
					Message: "Policy violation detected",
				},
			},
		},
	}
	
	return &rulepack, nil
}

// ScanReader provides a wrapper around the scanner's ScanReader method
// This allows the enforcement handler to use the manager's scanner directly
func (sm *ScannerManager) ScanReader(ctx context.Context, reader interface{}, inputName string) (pkg.ScanResult, error) {
	sm.mu.RLock()
	scanner := sm.scanner
	policyCount := len(sm.activePolicies)
	sm.mu.RUnlock()
	
	sm.logger.Debug("Scanning with event-driven scanner", "active_policies", policyCount, "input_name", inputName)
	
	// Convert reader to the expected type and call scanner
	// The scanner expects an io.Reader
	if r, ok := reader.(interface{ Read([]byte) (int, error) }); ok {
		result, err := scanner.ScanReader(ctx, r, inputName)
		sm.logger.Debug("Scanner result", "violations", len(result.Violations), "error", err)
		return result, err
	}
	
	return pkg.ScanResult{}, fmt.Errorf("invalid reader type")
}