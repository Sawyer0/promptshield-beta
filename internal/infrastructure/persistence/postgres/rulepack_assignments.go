package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// Ensure pgPolicyAssignmentRepo implements domain.RulepackAssignmentRepository
var _ domain.RulepackAssignmentRepository = (*pgRulepackAssignmentRepo)(nil)

type pgRulepackAssignmentRepo struct{ db *Pool }

// normalizeMethod coerces empty values to "*" and uppercases standard methods.
func normalizeMethod(m string) string {
	m = strings.TrimSpace(strings.ToUpper(m))
	if m == "" { return "*" }
	switch m {
	case "GET","POST","PUT","PATCH","DELETE","HEAD","OPTIONS","*":
		return m
	default:
		return m // allow custom verbs if needed
	}
}

func RulepackAssignmentRepo(db *Pool) domain.RulepackAssignmentRepository {
	return &pgRulepackAssignmentRepo{db: db}
}

func (r *pgRulepackAssignmentRepo) Create(ctx context.Context, assignment *domain.RulepackAssignment) error {
	// Validation
	if strings.TrimSpace(assignment.TargetScope) == "" {
		return fmt.Errorf("target scope cannot be empty")
	}
	if assignment.Priority <= 0 {
		return fmt.Errorf("priority must be positive")
	}
	if assignment.ID == uuid.Nil {
		assignment.ID = uuid.New()
	}
	if assignment.CreatedAt.IsZero() {
		assignment.CreatedAt = time.Now()
	}
	if assignment.UpdatedAt.IsZero() {
		assignment.UpdatedAt = time.Now()
	}

q := `INSERT INTO rulepack_assignments (id, tenant_id, rulepack_id, target_scope, method, priority, enabled, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		ON CONFLICT (tenant_id, target_scope, method, rulepack_id) DO UPDATE SET 
			priority = EXCLUDED.priority, enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at`

_, err := r.db.Raw().Exec(ctx, q,
		assignment.ID, assignment.TenantID, assignment.RulepackID,
		assignment.TargetScope, normalizeMethod(assignment.Method), assignment.Priority, assignment.Enabled,
		assignment.CreatedAt, assignment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create rulepack assignment: %w", err)
	}
	return nil
}

func (r *pgRulepackAssignmentRepo) Get(ctx context.Context, id uuid.UUID) (*domain.RulepackAssignment, error) {
	var a domain.RulepackAssignment
q := `SELECT id, tenant_id, rulepack_id, target_scope, method, priority, enabled, created_at, updated_at 
		FROM rulepack_assignments WHERE id = $1`

err := r.db.Raw().QueryRow(ctx, q, id).Scan(
		&a.ID, &a.TenantID, &a.RulepackID, &a.TargetScope, &a.Method,
		&a.Priority, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get rulepack assignment: %w", err)
	}
	return &a, nil
}

func (r *pgRulepackAssignmentRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.RulepackAssignment, error) {
q := `SELECT id, tenant_id, rulepack_id, target_scope, method, priority, enabled, created_at, updated_at 
		FROM rulepack_assignments 
		WHERE tenant_id = $1 
		ORDER BY priority DESC, created_at ASC`

	rows, err := r.db.Raw().Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list rulepack assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.RulepackAssignment
	for rows.Next() {
		var a domain.RulepackAssignment
err := rows.Scan(&a.ID, &a.TenantID, &a.RulepackID, &a.TargetScope, &a.Method,
			&a.Priority, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan rulepack assignment: %w", err)
		}
		assignments = append(assignments, &a)
	}
	return assignments, rows.Err()
}

func (r *pgRulepackAssignmentRepo) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.RulepackAssignment, error) {
q := `SELECT id, tenant_id, rulepack_id, target_scope, method, priority, enabled, created_at, updated_at 
		FROM rulepack_assignments 
		WHERE rulepack_id = $1 
		ORDER BY priority DESC, created_at ASC`

	rows, err := r.db.Raw().Query(ctx, q, policyID)
	if err != nil {
		return nil, fmt.Errorf("list rulepack assignments by rulepack: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.RulepackAssignment
	for rows.Next() {
		var a domain.RulepackAssignment
err := rows.Scan(&a.ID, &a.TenantID, &a.RulepackID, &a.TargetScope, &a.Method,
			&a.Priority, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan rulepack assignment: %w", err)
		}
		assignments = append(assignments, &a)
	}
	return assignments, rows.Err()
}

func (r *pgRulepackAssignmentRepo) ListByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]*domain.RulepackAssignment, error) {
q := `SELECT id, tenant_id, rulepack_id, target_scope, method, priority, enabled, created_at, updated_at 
		FROM rulepack_assignments 
		WHERE tenant_id = $1 AND target_scope = $2 AND enabled = true
		ORDER BY priority DESC, created_at ASC`

	rows, err := r.db.Raw().Query(ctx, q, tenantID, scope)
	if err != nil {
		return nil, fmt.Errorf("list rulepack assignments by scope: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.RulepackAssignment
	for rows.Next() {
		var a domain.RulepackAssignment
err := rows.Scan(&a.ID, &a.TenantID, &a.RulepackID, &a.TargetScope, &a.Method,
			&a.Priority, &a.Enabled, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan rulepack assignment: %w", err)
		}
		assignments = append(assignments, &a)
	}
	return assignments, rows.Err()
}

func (r *pgRulepackAssignmentRepo) Update(ctx context.Context, assignment *domain.RulepackAssignment) error {
	// Validation
	if strings.TrimSpace(assignment.TargetScope) == "" {
		return fmt.Errorf("target scope cannot be empty")
	}
	if assignment.Priority <= 0 {
		return fmt.Errorf("priority must be positive")
	}
	assignment.UpdatedAt = time.Now()

q := `UPDATE rulepack_assignments SET 
		target_scope = $2, method = $3, priority = $4, enabled = $5, updated_at = $6 
		WHERE id = $1`

result, err := r.db.Raw().Exec(ctx, q,
		assignment.ID, assignment.TargetScope, normalizeMethod(assignment.Method), assignment.Priority,
		assignment.Enabled, assignment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update rulepack assignment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("rulepack assignment not found")
	}
	return nil
}

func (r *pgRulepackAssignmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM rulepack_assignments WHERE id = $1`
	result, err := r.db.Raw().Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete rulepack assignment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("rulepack assignment not found")
	}
	return nil
}

func (r *pgRulepackAssignmentRepo) DeleteByTenantAndPolicy(ctx context.Context, tenantID, policyID uuid.UUID) error {
	q := `DELETE FROM rulepack_assignments WHERE tenant_id = $1 AND rulepack_id = $2`
	_, err := r.db.Raw().Exec(ctx, q, tenantID, policyID)
	if err != nil {
		return fmt.Errorf("delete rulepack assignments: %w", err)
	}
	return nil
}
