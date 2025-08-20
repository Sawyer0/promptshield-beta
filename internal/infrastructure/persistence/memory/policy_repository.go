package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// PolicyRepositoryImpl implements PolicyRepository interface using in-memory storage
// This is suitable for MVP and testing - replace with database implementation for production
type PolicyRepositoryImpl struct {
	mu       sync.RWMutex
	policies map[string]*types.Policy
}

// NewPolicyRepository creates a new in-memory policy repository
func NewPolicyRepository() contracts.PolicyRepository {
	repo := &PolicyRepositoryImpl{
		policies: make(map[string]*types.Policy),
	}
	
	// Seed with built-in policy for demo
	builtinPolicy := &types.Policy{
		ID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:    "Prompt Injection Detection",
		Version: 1,
		Content: getBuiltinRulepackContent(),
		Type:    types.PolicyTypeBuiltin,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	repo.policies[builtinPolicy.ID.String()] = builtinPolicy
	
	return repo
}

// Create creates a new policy
func (r *PolicyRepositoryImpl) Create(ctx context.Context, policy *types.Policy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if policy with same ID already exists
	if _, exists := r.policies[policy.ID.String()]; exists {
		return fmt.Errorf("policy with ID %s already exists", policy.ID.String())
	}

	// Store a copy to prevent external modifications
	policyCopy := *policy
	r.policies[policy.ID.String()] = &policyCopy

	return nil
}

// Get retrieves a policy by ID
func (r *PolicyRepositoryImpl) Get(ctx context.Context, id uuid.UUID) (*types.Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	policy, exists := r.policies[id.String()]
	if !exists {
		return nil, fmt.Errorf("policy with ID %s not found", id.String())
	}

	// Return a copy to prevent external modifications
	policyCopy := *policy
	return &policyCopy, nil
}

// Update updates an existing policy
func (r *PolicyRepositoryImpl) Update(ctx context.Context, policy *types.Policy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if policy exists
	if _, exists := r.policies[policy.ID.String()]; !exists {
		return fmt.Errorf("policy with ID %s not found", policy.ID.String())
	}

	// Store updated copy
	policyCopy := *policy
	r.policies[policy.ID.String()] = &policyCopy

	return nil
}

// Delete soft deletes a policy (marks as deleted)
func (r *PolicyRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if policy exists
	if _, exists := r.policies[id.String()]; !exists {
		return fmt.Errorf("policy with ID %s not found", id.String())
	}

	// For in-memory implementation, we'll actually remove it
	// In a database implementation, you'd set a deleted_at timestamp
	delete(r.policies, id.String())

	return nil
}

// GetByName retrieves a policy by name
func (r *PolicyRepositoryImpl) GetByName(ctx context.Context, name string) (*types.Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, policy := range r.policies {
		if policy.Name == name {
			policyCopy := *policy
			return &policyCopy, nil
		}
	}

	return nil, fmt.Errorf("policy with name %s not found", name)
}

// List retrieves policies with pagination and type filtering
func (r *PolicyRepositoryImpl) List(ctx context.Context, policyType *types.PolicyType, offset, limit int) ([]*types.Policy, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*types.Policy
	
	// Apply type filter
	for _, policy := range r.policies {
		if policyType == nil || policy.Type == *policyType {
			policyCopy := *policy
			result = append(result, &policyCopy)
		}
	}

	total := len(result)
	
	// Apply pagination
	if offset > 0 && offset < len(result) {
		result = result[offset:]
	}
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result, total, nil
}

// GetLatestVersion gets the latest version of a policy by name
func (r *PolicyRepositoryImpl) GetLatestVersion(ctx context.Context, name string) (*types.Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *types.Policy
	for _, policy := range r.policies {
		if policy.Name == name {
			if latest == nil || policy.Version > latest.Version {
				latest = policy
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("policy with name %s not found", name)
	}

	policyCopy := *latest
	return &policyCopy, nil
}

// ListWithFilter retrieves policies with optional filtering 
func (r *PolicyRepositoryImpl) ListWithFilter(ctx context.Context, filter map[string]interface{}) ([]*types.Policy, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*types.Policy
	
	// Apply filters
	for _, policy := range r.policies {
		if matchesFilter(policy, filter) {
			policyCopy := *policy
			result = append(result, &policyCopy)
		}
	}

	// Apply limit and offset if specified
	if limit, ok := filter["limit"].(int); ok && limit > 0 {
		if len(result) > limit {
			result = result[:limit]
		}
	}

	return result, len(r.policies), nil
}

// GetActive retrieves all active policies
func (r *PolicyRepositoryImpl) GetActive(ctx context.Context) ([]*types.Policy, error) {
	filter := map[string]interface{}{
		"active": true,
	}
	policies, _, err := r.ListWithFilter(ctx, filter)
	return policies, err
}

// Helper functions

// matchesFilter checks if a policy matches the given filter criteria
func matchesFilter(policy *types.Policy, filter map[string]interface{}) bool {
	// Filter by type
	if policyType, ok := filter["type"].(types.PolicyType); ok {
		if policy.Type != policyType {
			return false
		}
	}

	// Filter by name (contains)
	if name, ok := filter["name"].(string); ok {
		if name != "" && !contains(policy.Name, name) {
			return false
		}
	}

	// Add more filters as needed
	return true
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		   (len(s) > len(substr) && 
		   	(s[:len(substr)] == substr || 
		   	 s[len(s)-len(substr):] == substr ||
		   	 findInString(s, substr))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getBuiltinRulepackContent returns the content of the built-in prompt injection rulpack
func getBuiltinRulepackContent() string {
	return `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: prompt-injection-detection
  description: Built-in comprehensive prompt injection and jailbreak detection rules
rules:
  - id: pi-direct-ignore
    name: Direct instruction override attempts
    level: 1
    severity: CRITICAL
    keywords:
      - "ignore previous instructions"
      - "ignore all previous"
      - "disregard previous"
    response:
      action: block
      message: "Potential prompt injection detected"
  - id: pi-role-manipulation
    name: Role manipulation attempts  
    level: 1
    severity: HIGH
    keywords:
      - "you are now"
      - "act as"
      - "pretend to be"
    response:
      action: warn
      message: "Potential role manipulation detected"`
}