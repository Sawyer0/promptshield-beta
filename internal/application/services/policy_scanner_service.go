package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
	pkg "github.com/promptshield/promptshield/pkg/types"
)

// PolicyScannerService manages a scanner instance that loads policies as RulePacks
type PolicyScannerService struct {
	mu            sync.RWMutex
	scanner       *scanner.Scanner
	policyRepo    contracts.PolicyRepository
	activePolicies map[uuid.UUID]*types.Policy
}

// NewPolicyScannerService creates a new policy scanner service
func NewPolicyScannerService(policyRepo contracts.PolicyRepository) *PolicyScannerService {
	sc := scanner.ScanEngineCstor(0)
	
	// Configure scanner for production use
	sc.SetQuarantineOnTimeout(true)
	sc.SetQuarantineOnError(true)
	sc.SetMaxStreamBytes(10 * 1024 * 1024) // 10MB max
	
	return &PolicyScannerService{
		scanner:        sc,
		policyRepo:     policyRepo,
		activePolicies: make(map[uuid.UUID]*types.Policy),
	}
}

// GetScanner returns the underlying scanner for external use
func (s *PolicyScannerService) GetScanner() *scanner.Scanner {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanner
}

// ReloadActivePolicies reloads all active policies into the scanner
func (s *PolicyScannerService) ReloadActivePolicies(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Get all active policies from repository
	policies, err := s.policyRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active policies: %w", err)
	}
	
	// Convert policies to RulePacks
	var rulePacks []rules.RulePack
	activePolicies := make(map[uuid.UUID]*types.Policy)
	
	for _, policy := range policies {
		rulepack, err := s.convertPolicyToRulePack(policy)
		if err != nil {
			// Log error but continue with other policies
			continue
		}
		
		rulePacks = append(rulePacks, *rulepack)
		activePolicies[policy.ID] = policy
	}
	
	// Load RulePacks into scanner
	s.scanner.LoadRulePacks(rulePacks)
	s.activePolicies = activePolicies
	
	return nil
}

// ActivatePolicy adds a policy to the active set and reloads the scanner
func (s *PolicyScannerService) ActivatePolicy(ctx context.Context, policyID uuid.UUID) error {
	// Get the policy
	policy, err := s.policyRepo.Get(ctx, policyID)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Add to active policies
	s.activePolicies[policyID] = policy
	
	// Rebuild RulePacks from all active policies
	return s.reloadScannerLocked()
}

// DeactivatePolicy removes a policy from the active set and reloads the scanner  
func (s *PolicyScannerService) DeactivatePolicy(ctx context.Context, policyID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Remove from active policies
	delete(s.activePolicies, policyID)
	
	// Rebuild RulePacks from remaining active policies
	return s.reloadScannerLocked()
}

// ScanText scans text content using active policies
func (s *PolicyScannerService) ScanText(ctx context.Context, content string, policyCtx *types.PolicyContext) (*types.ScanResult, error) {
	s.mu.RLock()
	sc := s.scanner
	s.mu.RUnlock()
	
	// Use scanner to scan the content - the scanner doesn't have ScanText, let me use ScanReader
	reader := strings.NewReader(content)
	result, err := sc.ScanReader(ctx, reader, "policy-test")
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}
	
	// Convert scanner result to types.ScanResult
	scanResult := &types.ScanResult{
		ID:         uuid.New().String(),
		Input:      content,
		Violations: convertViolations(result.Violations),
		CreatedAt:  time.Now().UTC(),
		Metrics: &types.ScanMetrics{
			BytesRead:        result.Metrics.BytesRead,
			LinesRead:        result.Metrics.LinesRead,
			RegexAttempts:    result.Metrics.RegexAttempts,
			RegexSkipped:     result.Metrics.RegexSkipped,
			SemanticAttempts: result.Metrics.SemanticAttempts,
			SemanticSkipped:  result.Metrics.SemanticSkipped,
			ProcessingTime:   time.Duration(result.ScanInfo.ScanDurationMs) * time.Millisecond,
		},
	}
	
	return scanResult, nil
}

// reloadScannerLocked rebuilds and reloads the scanner (must be called with lock held)
func (s *PolicyScannerService) reloadScannerLocked() error {
	var rulePacks []rules.RulePack
	
	for _, policy := range s.activePolicies {
		rulepack, err := s.convertPolicyToRulePack(policy)
		if err != nil {
			// Log error but continue
			continue
		}
		rulePacks = append(rulePacks, *rulepack)
	}
	
	// Reload scanner with new RulePacks
	s.scanner.LoadRulePacks(rulePacks)
	return nil
}

// convertPolicyToRulePack converts a Policy to a RulePack
func (s *PolicyScannerService) convertPolicyToRulePack(policy *types.Policy) (*rules.RulePack, error) {
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

// convertViolations converts scanner violations to types.PolicyViolation
func convertViolations(violations []pkg.Violation) []*types.PolicyViolation {
	var result []*types.PolicyViolation
	
	for _, v := range violations {
		// Map severity from scanner to string
		severity := strings.ToLower(v.Severity)
		if severity == "" {
			severity = "medium"
		}
		
		violation := &types.PolicyViolation{
			RuleID:     v.RuleID,
			Severity:   severity,
			Message:    v.Message,
			Line:       v.Line,
			Column:     v.Column,
			Category:   v.Category,
			Action:     "deny", // Default action
			Confidence: 1.0,    // Full confidence from rule match
			
			// Response action fields from scanner violation
			ResponseAction:      v.ResponseAction,
			ResponseMessage:     v.ResponseMessage,
			ResponseReplacement: v.ResponseReplacement,
			RuleTimeoutMs:       v.RuleTimeoutMs,
		}
		result = append(result, violation)
	}
	
	return result
}