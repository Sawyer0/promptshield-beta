package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// Ensure pgPolicyAssignmentRepo implements domain.PolicyAssignmentRepository
var _ domain.PolicyAssignmentRepository = (*pgPolicyAssignmentRepo)(nil)

type pgPolicyAssignmentRepo struct{ db *Pool }

func PolicyAssignmentRepo(db *Pool) domain.PolicyAssignmentRepository { 
	return &pgPolicyAssignmentRepo{db: db} 
}

func (r *pgPolicyAssignmentRepo) Create(ctx context.Context, assignment *domain.PolicyAssignment) error {
	if assignment.ID == uuid.Nil {
		assignment.ID = uuid.New()
	}
	if assignment.CreatedAt.IsZero() {
		assignment.CreatedAt = time.Now()
	}
	if assignment.UpdatedAt.IsZero() {
		assignment.UpdatedAt = time.Now()
	}

	q := `INSERT INTO policy_assignments (id, tenant_id, policy_id, target_scope, priority, enabled, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		ON CONFLICT (tenant_id, target_scope, policy_id) DO UPDATE SET 
			priority = $5, enabled = $6, updated_at = $8`
	
	_, err := r.db.Raw().Exec(ctx, q, 
		assignment.ID, assignment.TenantID, assignment.PolicyID, 
		assignment.TargetScope, assignment.Priority, assignment.Enabled,
		assignment.CreatedAt, assignment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create policy assignment: %w", err)
	}
	return nil
}

func (r *pgPolicyAssignmentRepo) Get(ctx context.Context, id uuid.UUID) (*domain.PolicyAssignment, error) {
	var a domain.PolicyAssignment
	q := `SELECT id, tenant_id, policy_id, target_scope, priority, enabled, created_at, updated_at 
		FROM policy_assignments WHERE id = $1`
	
	err := r.db.Raw().QueryRow(ctx, q, id).Scan(
		&a.ID, &a.TenantID, &a.PolicyID, &a.TargetScope, 
		&a.Priority, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get policy assignment: %w", err)
	}
	return &a, nil
}

func (r *pgPolicyAssignmentRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.PolicyAssignment, error) {
	q := `SELECT id, tenant_id, policy_id, target_scope, priority, enabled, created_at, updated_at 
		FROM policy_assignments 
		WHERE tenant_id = $1 
		ORDER BY priority DESC, created_at ASC`
	
	rows, err := r.db.Raw().Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list policy assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.PolicyAssignment
	for rows.Next() {
		var a domain.PolicyAssignment
		err := rows.Scan(&a.ID, &a.TenantID, &a.PolicyID, &a.TargetScope, 
			&a.Priority, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan policy assignment: %w", err)
		}
		assignments = append(assignments, &a)
	}
	return assignments, rows.Err()
}

func (r *pgPolicyAssignmentRepo) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.PolicyAssignment, error) {
	q := `SELECT id, tenant_id, policy_id, target_scope, priority, enabled, created_at, updated_at 
		FROM policy_assignments 
		WHERE policy_id = $1 
		ORDER BY priority DESC, created_at ASC`
	
	rows, err := r.db.Raw().Query(ctx, q, policyID)
	if err != nil {
		return nil, fmt.Errorf("list policy assignments by policy: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.PolicyAssignment
	for rows.Next() {
		var a domain.PolicyAssignment
		err := rows.Scan(&a.ID, &a.TenantID, &a.PolicyID, &a.TargetScope, 
			&a.Priority, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan policy assignment: %w", err)
		}
		assignments = append(assignments, &a)
	}
	return assignments, rows.Err()
}

func (r *pgPolicyAssignmentRepo) ListByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]*domain.PolicyAssignment, error) {
	q := `SELECT id, tenant_id, policy_id, target_scope, priority, enabled, created_at, updated_at 
		FROM policy_assignments 
		WHERE tenant_id = $1 AND target_scope = $2 AND enabled = true
		ORDER BY priority DESC, created_at ASC`
	
	rows, err := r.db.Raw().Query(ctx, q, tenantID, scope)
	if err != nil {
		return nil, fmt.Errorf("list policy assignments by scope: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.PolicyAssignment
	for rows.Next() {
		var a domain.PolicyAssignment
		err := rows.Scan(&a.ID, &a.TenantID, &a.PolicyID, &a.TargetScope, 
			&a.Priority, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan policy assignment: %w", err)
		}
		assignments = append(assignments, &a)
	}
	return assignments, rows.Err()
}

func (r *pgPolicyAssignmentRepo) Update(ctx context.Context, assignment *domain.PolicyAssignment) error {
	assignment.UpdatedAt = time.Now()
	
	q := `UPDATE policy_assignments SET 
		target_scope = $2, priority = $3, enabled = $4, updated_at = $5 
		WHERE id = $1`
	
	result, err := r.db.Raw().Exec(ctx, q, 
		assignment.ID, assignment.TargetScope, assignment.Priority, 
		assignment.Enabled, assignment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update policy assignment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("policy assignment not found")
	}
	return nil
}

func (r *pgPolicyAssignmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM policy_assignments WHERE id = $1`
	result, err := r.db.Raw().Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete policy assignment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("policy assignment not found")
	}
	return nil
}

func (r *pgPolicyAssignmentRepo) DeleteByTenantAndPolicy(ctx context.Context, tenantID, policyID uuid.UUID) error {
	q := `DELETE FROM policy_assignments WHERE tenant_id = $1 AND policy_id = $2`
	_, err := r.db.Raw().Exec(ctx, q, tenantID, policyID)
	if err != nil {
		return fmt.Errorf("delete policy assignments: %w", err)
	}
	return nil
}