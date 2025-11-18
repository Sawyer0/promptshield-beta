package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/events"
	"github.com/promptshield/promptshield/internal/shared/types"
	ctxutil "github.com/promptshield/promptshield/internal/util/context"
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
				_ = auditLogger.LogWithContext(ctx, *auditEvent)
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
	var activatedBy *uuid.UUID
	if userIDStr, ok := ctxutil.GetUserID(ctx); ok {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			activatedBy = &userID
		}
	}

	activationEvent := &events.PolicyActivated{
		BaseEvent:   events.NewBaseEvent(events.EventTypePolicyActivated, nil),
		PolicyID:    policy.ID,
		PolicyData:  *policy,
		ActivatedBy: activatedBy,
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
	var deactivatedBy *uuid.UUID
	if userIDStr, ok := ctxutil.GetUserID(ctx); ok {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			deactivatedBy = &userID
		}
	}

	deactivationEvent := &events.PolicyDeactivated{
		BaseEvent:     events.NewBaseEvent(events.EventTypePolicyDeactivated, nil),
		PolicyID:      policy.ID,
		PolicyData:    *policy,
		DeactivatedBy: deactivatedBy,
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
// This is a simplified version - in production you'd parse YAML properly
func parseRulesFromContent(content string) []interface{} {
	// Mock implementation - would parse YAML rules
	return []interface{}{}
}
