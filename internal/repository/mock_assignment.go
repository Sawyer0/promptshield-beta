package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// MockRulepackAssignmentRepository provides a simple in-memory implementation
type MockRulepackAssignmentRepository struct {
	mu          sync.RWMutex
	assignments map[uuid.UUID]*domain.RulepackAssignment
}

// NewMockRulepackAssignmentRepository creates a new mock assignment repository
func NewMockRulepackAssignmentRepository() *MockRulepackAssignmentRepository {
	return &MockRulepackAssignmentRepository{
		assignments: make(map[uuid.UUID]*domain.RulepackAssignment),
	}
}

// Create creates a new rulepack assignment
func (r *MockRulepackAssignmentRepository) Create(ctx context.Context, assignment *domain.RulepackAssignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assignments[assignment.ID] = assignment
	return nil
}

// Get retrieves an assignment by ID
func (r *MockRulepackAssignmentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.RulepackAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if assignment, exists := r.assignments[id]; exists {
		return assignment, nil
	}
	return nil, fmt.Errorf("assignment not found")
}

// ListByTenant retrieves assignments by tenant ID
func (r *MockRulepackAssignmentRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.RulepackAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	assignments := make([]*domain.RulepackAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID {
			assignments = append(assignments, assignment)
		}
	}
	return assignments, nil
}

// ListByPolicy retrieves assignments by policy ID (using RulepackID)
func (r *MockRulepackAssignmentRepository) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.RulepackAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	assignments := make([]*domain.RulepackAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.RulepackID == policyID {
			assignments = append(assignments, assignment)
		}
	}
	return assignments, nil
}

// ListByScope retrieves assignments by tenant and scope
func (r *MockRulepackAssignmentRepository) ListByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]*domain.RulepackAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	assignments := make([]*domain.RulepackAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.TargetScope == scope {
			assignments = append(assignments, assignment)
		}
	}
	return assignments, nil
}

// Update updates an existing assignment
func (r *MockRulepackAssignmentRepository) Update(ctx context.Context, assignment *domain.RulepackAssignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assignments[assignment.ID] = assignment
	return nil
}

// Delete deletes an assignment
func (r *MockRulepackAssignmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.assignments, id)
	return nil
}

// DeleteByTenantAndPolicy deletes assignments by tenant and policy (using RulepackID)
func (r *MockRulepackAssignmentRepository) DeleteByTenantAndPolicy(ctx context.Context, tenantID, policyID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for id, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.RulepackID == policyID {
			delete(r.assignments, id)
		}
	}
	return nil
}

// Reset clears all data (for testing)
func (r *MockRulepackAssignmentRepository) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assignments = make(map[uuid.UUID]*domain.RulepackAssignment)
	return nil
}