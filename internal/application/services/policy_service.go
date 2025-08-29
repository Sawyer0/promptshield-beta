package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/events"
	"github.com/promptshield/promptshield/internal/shared/types"
	"gopkg.in/yaml.v3"
)

// PolicyServiceImpl implements the PolicyService interface
type PolicyServiceImpl struct {
	repository     contracts.PolicyRepository
	validator      contracts.RuleCompiler
	scanEngine     contracts.ScanEngine
	auditLogger    contracts.AuditLogger
	scannerService *PolicyScannerService
}

// NewPolicyService creates a new policy service instance
func NewPolicyService(
	repository contracts.PolicyRepository,
	validator contracts.RuleCompiler,
	scanEngine contracts.ScanEngine,
	auditLogger contracts.AuditLogger,
) contracts.PolicyService {
	scannerService := NewPolicyScannerService(repository)

	service := &PolicyServiceImpl{
		repository:     repository,
		validator:      validator,
		scanEngine:     scanEngine,
		auditLogger:    auditLogger,
		scannerService: scannerService,
	}

	// Initialize scanner with active policies on startup
	go func() {
		ctx := context.Background()
		if err := scannerService.ReloadActivePolicies(ctx); err != nil {
			// Log error but don't fail startup
			if auditLogger != nil {
				auditEvent := &types.AuditEvent{
					Action:     "policy.scanner.init_failed",
					ObjectType: "system",
					Metadata: map[string]interface{}{
						"error": err.Error(),
					},
					Timestamp: time.Now().UTC(),
				}
				if err := auditLogger.LogWithContext(ctx, *auditEvent); err != nil {
					// Log audit failure but don't fail the operation
					slog.Error("Failed to audit policy service creation", "error", err)
				}
			}
		}
	}()

	return service
}

// CreatePolicy creates a new policy with validation
func (s *PolicyServiceImpl) CreatePolicy(ctx context.Context, policy *types.Policy) (*types.Policy, error) {
	// Set creation metadata
	policy.ID = uuid.New()
	policy.CreatedAt = time.Now().UTC()
	policy.UpdatedAt = policy.CreatedAt
	policy.Version = 1

	// Validate policy structure and rules
	if err := s.ValidatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("policy validation failed: %w", err)
	}

	// Create policy in repository
	if err := s.repository.Create(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	// Audit the policy creation
	if s.auditLogger != nil {
		auditEvent := &types.AuditEvent{
			Action:     "policy.create",
			ObjectType: "policy",
			ObjectID:   policy.ID,
			Metadata: map[string]interface{}{
				"policy_name": policy.Name,
				"policy_type": policy.Type,
				"rules_count": len(parseRulesFromContent(policy.Content)),
			},
			Timestamp: time.Now().UTC(),
		}
		_ = s.auditLogger.LogWithContext(ctx, *auditEvent)
	}

	return policy, nil
}

// UpdatePolicy updates an existing policy
func (s *PolicyServiceImpl) UpdatePolicy(ctx context.Context, policy *types.Policy) (*types.Policy, error) {
	// Validate the policy exists
	existing, err := s.repository.Get(ctx, policy.ID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	// Preserve creation metadata, update modification metadata
	policy.CreatedAt = existing.CreatedAt
	policy.CreatedBy = existing.CreatedBy
	policy.UpdatedAt = time.Now().UTC()
	policy.Version = existing.Version + 1

	// Validate updated policy
	if err := s.ValidatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("policy validation failed: %w", err)
	}

	// Update in repository
	if err := s.repository.Update(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	// Audit the policy update
	if s.auditLogger != nil {
		auditEvent := &types.AuditEvent{
			Action:     "policy.update",
			ObjectType: "policy",
			ObjectID:   policy.ID,
			Before: map[string]interface{}{
				"version": existing.Version,
				"name":    existing.Name,
			},
			After: map[string]interface{}{
				"version": policy.Version,
				"name":    policy.Name,
			},
			Metadata: map[string]interface{}{
				"policy_name": policy.Name,
				"old_version": existing.Version,
				"new_version": policy.Version,
			},
			Timestamp: time.Now().UTC(),
		}
		_ = s.auditLogger.LogWithContext(ctx, *auditEvent)
	}

	return policy, nil
}

// DeletePolicy deletes a policy
func (s *PolicyServiceImpl) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	// Validate the policy exists
	policy, err := s.repository.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	// Audit the policy deletion
	if s.auditLogger != nil {
		auditEvent := &types.AuditEvent{
			Action:     "policy.delete",
			ObjectType: "policy",
			ObjectID:   policy.ID,
			Metadata: map[string]interface{}{
				"policy_name": policy.Name,
				"policy_type": policy.Type,
			},
			Timestamp: time.Now().UTC(),
		}
		_ = s.auditLogger.LogWithContext(ctx, *auditEvent)
	}

	return nil
}

// GetPolicy retrieves a policy by ID
func (s *PolicyServiceImpl) GetPolicy(ctx context.Context, id uuid.UUID) (*types.Policy, error) {
	policy, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}
	return policy, nil
}

// ListPolicies lists policies with filtering
func (s *PolicyServiceImpl) ListPolicies(ctx context.Context, filter map[string]interface{}) ([]*types.Policy, int, error) {
	policies, total, err := s.repository.ListWithFilter(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list policies: %w", err)
	}
	return policies, total, nil
}

// ActivatePolicy activates a policy and triggers scanner reload
func (s *PolicyServiceImpl) ActivatePolicy(ctx context.Context, id uuid.UUID) error {
	// Get the policy
	policy, err := s.repository.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	// Update policy timestamp in repository
	policy.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	// Activate policy in scanner service
	if s.scannerService != nil {
		if err := s.scannerService.ActivatePolicy(ctx, id); err != nil {
			return fmt.Errorf("failed to activate policy in scanner: %w", err)
		}
	}

	// Publish policy activation event for real-time enforcement
	activationEvent := &events.PolicyActivated{
		BaseEvent:   events.NewBaseEvent(events.EventTypePolicyActivated, nil),
		PolicyID:    policy.ID,
		PolicyData:  *policy,
		ActivatedBy: getUserIDFromContext(ctx),
	}

	// Publish asynchronously so API response isn't delayed
	if err := events.GlobalEventBus().Publish(ctx, activationEvent); err != nil {
		// Log error but don't fail the activation
		if s.auditLogger != nil {
			auditEvent := &types.AuditEvent{
				Action:     "policy.activation.event_failed",
				ObjectType: "policy",
				ObjectID:   policy.ID,
				Metadata: map[string]interface{}{
					"policy_name": policy.Name,
					"error":       err.Error(),
				},
				Timestamp: time.Now().UTC(),
			}
			_ = s.auditLogger.LogWithContext(ctx, *auditEvent)
		}
	}

	// Audit the activation
	if s.auditLogger != nil {
		auditEvent := &types.AuditEvent{
			Action:     "policy.activate",
			ObjectType: "policy",
			ObjectID:   policy.ID,
			Metadata: map[string]interface{}{
				"policy_name": policy.Name,
			},
			Timestamp: time.Now().UTC(),
		}
		_ = s.auditLogger.LogWithContext(ctx, *auditEvent)
	}

	return nil
}

// DeactivatePolicy deactivates a policy
func (s *PolicyServiceImpl) DeactivatePolicy(ctx context.Context, id uuid.UUID) error {
	// Get the policy
	policy, err := s.repository.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	// Update policy timestamp in repository
	policy.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	// Deactivate policy in scanner service
	if s.scannerService != nil {
		if err := s.scannerService.DeactivatePolicy(ctx, id); err != nil {
			return fmt.Errorf("failed to deactivate policy in scanner: %w", err)
		}
	}

	// Publish policy deactivation event for real-time enforcement
	deactivationEvent := &events.PolicyDeactivated{
		BaseEvent:     events.NewBaseEvent(events.EventTypePolicyDeactivated, nil),
		PolicyID:      policy.ID,
		PolicyData:    *policy,
		DeactivatedBy: getUserIDFromContext(ctx),
	}

	// Publish asynchronously
	if err := events.GlobalEventBus().Publish(ctx, deactivationEvent); err != nil {
		// Log error but don't fail the deactivation
		if s.auditLogger != nil {
			auditEvent := &types.AuditEvent{
				Action:     "policy.deactivation.event_failed",
				ObjectType: "policy",
				ObjectID:   policy.ID,
				Metadata: map[string]interface{}{
					"policy_name": policy.Name,
					"error":       err.Error(),
				},
				Timestamp: time.Now().UTC(),
			}
			_ = s.auditLogger.LogWithContext(ctx, *auditEvent)
		}
	}

	// Audit the deactivation
	if s.auditLogger != nil {
		auditEvent := &types.AuditEvent{
			Action:     "policy.deactivate",
			ObjectType: "policy",
			ObjectID:   policy.ID,
			Metadata: map[string]interface{}{
				"policy_name": policy.Name,
			},
			Timestamp: time.Now().UTC(),
		}
		_ = s.auditLogger.LogWithContext(ctx, *auditEvent)
	}

	return nil
}

// ValidatePolicy validates policy structure and rules
func (s *PolicyServiceImpl) ValidatePolicy(ctx context.Context, policy *types.Policy) error {
	// Basic validation
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if policy.Content == "" {
		return fmt.Errorf("policy content is required")
	}

	// Use rule compiler for advanced validation
	if s.validator != nil {
		if err := s.validator.ValidatePolicy(ctx, policy); err != nil {
			return fmt.Errorf("rule validation failed: %w", err)
		}
	}

	return nil
}

// TestPolicy tests content against a specific policy
func (s *PolicyServiceImpl) TestPolicy(ctx context.Context, policyID uuid.UUID, content string) (*types.ScanResult, error) {
	// Validate the policy exists
	_, err := s.repository.Get(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	// Create a temporary policy context for testing
	policyCtx := &types.PolicyContext{
		TenantID:  uuid.Nil, // Test context
		RequestID: uuid.New().String(),
		Timestamp: time.Now().UTC(),
	}

	// Use scanner service to test content against this specific policy
	if s.scannerService != nil {
		// Create a temporary scanner with just this policy for testing
		tempScannerService := NewPolicyScannerService(s.repository)

		// Temporarily activate this policy in the test scanner
		if err := tempScannerService.ActivatePolicy(ctx, policyID); err != nil {
			return nil, fmt.Errorf("failed to activate policy for testing: %w", err)
		}

		// Scan the content
		return tempScannerService.ScanText(ctx, content, policyCtx)
	}

	// Fallback to scan engine if available
	if s.scanEngine != nil {
		return s.scanEngine.ScanText(ctx, content, policyCtx)
	}

	// If no scanner available, return empty result
	return &types.ScanResult{
		ID:         uuid.New().String(),
		Input:      content,
		Violations: []*types.PolicyViolation{},
		CreatedAt:  time.Now().UTC(),
	}, nil
}

// GetActiveScanner returns the scanner instance for real-time enforcement
func (s *PolicyServiceImpl) GetActiveScanner() interface{} {
	if s.scannerService != nil {
		return s.scannerService.GetScanner()
	}
	return nil
}

// HasActivePolicies checks if any policies are currently active
func (s *PolicyServiceImpl) HasActivePolicies() bool {
	if s.scannerService == nil {
		return false
	}

	s.scannerService.mu.RLock()
	defer s.scannerService.mu.RUnlock()

	return len(s.scannerService.activePolicies) > 0
}

// Helper functions

// parseRulesFromContent extracts rules from policy content
func parseRulesFromContent(content string) []interface{} {
	if content == "" {
		return []interface{}{}
	}

	// Try to parse as a full policy structure first
	var policy struct {
		Metadata struct {
			Name        string            `yaml:"name"`
			Description string            `yaml:"description"`
			Version     string            `yaml:"version"`
			Tags        []string          `yaml:"tags"`
			Author      string            `yaml:"author"`
			CreatedAt   string            `yaml:"created_at"`
			UpdatedAt   string            `yaml:"updated_at"`
			Custom      map[string]string `yaml:"custom"`
		} `yaml:"metadata"`
		Rules []struct {
			ID          string   `yaml:"id"`
			Name        string   `yaml:"name"`
			Description string   `yaml:"description"`
			Level       int      `yaml:"level"`
			Severity    string   `yaml:"severity"`
			Keywords    []string `yaml:"keywords"`
			Patterns    []struct {
				Regex         string `yaml:"regex"`
				CaseSensitive bool   `yaml:"case_sensitive"`
			} `yaml:"patterns"`
			Actions    []string `yaml:"actions"`
			Tags       []string `yaml:"tags"`
			Enabled    bool     `yaml:"enabled"`
			Priority   int      `yaml:"priority"`
			Categories []string `yaml:"categories"`
			References []string `yaml:"references"`
		} `yaml:"rules"`
		Settings struct {
			MaxViolations int    `yaml:"max_violations"`
			Timeout       string `yaml:"timeout"`
			CacheEnabled  bool   `yaml:"cache_enabled"`
		} `yaml:"settings"`
	}

	if err := yaml.Unmarshal([]byte(content), &policy); err != nil {
		// If full policy parsing fails, try parsing as just a rules array
		var rulesOnly []struct {
			ID          string   `yaml:"id"`
			Name        string   `yaml:"name"`
			Description string   `yaml:"description"`
			Level       int      `yaml:"level"`
			Severity    string   `yaml:"severity"`
			Keywords    []string `yaml:"keywords"`
			Patterns    []struct {
				Regex         string `yaml:"regex"`
				CaseSensitive bool   `yaml:"case_sensitive"`
			} `yaml:"patterns"`
			Actions    []string `yaml:"actions"`
			Tags       []string `yaml:"tags"`
			Enabled    bool     `yaml:"enabled"`
			Priority   int      `yaml:"priority"`
			Categories []string `yaml:"categories"`
			References []string `yaml:"references"`
		}

		if err := yaml.Unmarshal([]byte(content), &rulesOnly); err != nil {
			// If both fail, try parsing as generic interface{} for backward compatibility
			var genericPolicy struct {
				Rules []interface{} `yaml:"rules"`
			}
			if err := yaml.Unmarshal([]byte(content), &genericPolicy); err != nil {
				// Log the parsing error for debugging
				slog.Warn("Failed to parse policy content for rule counting",
					"error", err,
					"content_length", len(content))
				return []interface{}{}
			}
			return genericPolicy.Rules
		}

		// Convert typed rules to interface{} for consistency
		rules := make([]interface{}, len(rulesOnly))
		for i, rule := range rulesOnly {
			rules[i] = rule
		}
		return rules
	}

	// Convert typed rules to interface{} for consistency
	rules := make([]interface{}, len(policy.Rules))
	for i, rule := range policy.Rules {
		rules[i] = rule
	}
	return rules
}

// getUserFromContext extracts user information from request context
func getUserFromContext(ctx context.Context) *UserInfo {
	if ctx == nil {
		return &UserInfo{
			ID:   "system",
			Name: "System",
		}
	}
	
	// Try to get user ID from context
	if userID := ctx.Value("user_id"); userID != nil {
		if strUserID, ok := userID.(string); ok {
			user := &UserInfo{
				ID: strUserID,
			}
			
			// Try to get additional user details
			if userName := ctx.Value("user_name"); userName != nil {
				if strUserName, ok := userName.(string); ok {
					user.Name = strUserName
				}
			}
			
			if userEmail := ctx.Value("user_email"); userEmail != nil {
				if strUserEmail, ok := userEmail.(string); ok {
					user.Email = strUserEmail
				}
			}
			
			return user
		}
	}
	
	// Default to system user if no context available
	return &UserInfo{
		ID:   "system",
		Name: "System",
	}
}

// getUserIDFromContext extracts user ID as UUID from request context
func getUserIDFromContext(ctx context.Context) *uuid.UUID {
	if ctx == nil {
		return nil
	}
	
	// Try to get user ID from context
	if userID := ctx.Value("user_id"); userID != nil {
		if strUserID, ok := userID.(string); ok {
			if parsedUUID, err := uuid.Parse(strUserID); err == nil {
				return &parsedUUID
			}
		}
	}
	
	// Return nil for system/anonymous operations
	return nil
}

// UserInfo represents user information for audit events
type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}
