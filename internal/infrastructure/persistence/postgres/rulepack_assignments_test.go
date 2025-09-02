package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRulepackAssignmentRepository_Create(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()
	db.EnsureTestSchema(t)

	pool := &Pool{inner: db.pool}
	repo := RulepackAssignmentRepo(pool)
	tenantID, _, rulepackID, _ := db.SeedTestData(t)

	assignment := &domain.RulepackAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RulepackID:  rulepackID,
		TargetScope: "/api/v1/users",
		Priority:    1,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, repo.Create(context.Background(), assignment))
}

func TestRulepackAssignmentRepository_Create_Validation(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()
	db.EnsureTestSchema(t)

	pool := &Pool{inner: db.pool}
	repo := RulepackAssignmentRepo(pool)
	tenantID, _, rulepackID, _ := db.SeedTestData(t)

	// Empty target scope
	a1 := &domain.RulepackAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RulepackID:  rulepackID,
		TargetScope: "",
		Priority:    1,
		Enabled:     true,
	}
	err := repo.Create(context.Background(), a1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target scope")

	// Non-positive priority
	a2 := &domain.RulepackAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RulepackID:  rulepackID,
		TargetScope: "/api/v1/users",
		Priority:    0,
		Enabled:     true,
	}
	err = repo.Create(context.Background(), a2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "priority")
}

func TestRulepackAssignmentRepository_GetByTenantID(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()
	db.EnsureTestSchema(t)

	db.RunWithTx(t, func(t *testing.T) {
		// Seed tenant/rulepack and insert assignment
		tenantID, _, rulepackID, _ := db.SeedTestData(t)
		db.Exec(t, `
			INSERT INTO rulepack_assignments (id, tenant_id, rulepack_id, target_scope, priority, enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			uuid.New(),
			tenantID,
			rulepackID,
			"/api/v1/users",
			1,
			true,
			time.Now(),
			time.Now(),
		)

		var count int
		err := db.QueryRow(t, "SELECT COUNT(*) FROM rulepack_assignments WHERE tenant_id = $1", tenantID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestRulepackAssignmentRepository_Update(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()
	db.EnsureTestSchema(t)

	pool := &Pool{inner: db.pool}
	repo := RulepackAssignmentRepo(pool)
	tenantID, _, rulepackID, _ := db.SeedTestData(t)
	assignment := &domain.RulepackAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RulepackID:  rulepackID,
		TargetScope: "/api/v1/admin",
		Priority:    2,
		Enabled:     false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	// Insert outside of transaction to ensure visibility
	db.Exec(t, `
		INSERT INTO rulepack_assignments (id, tenant_id, rulepack_id, target_scope, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, assignment.ID, assignment.TenantID, assignment.RulepackID, assignment.TargetScope, assignment.Priority, assignment.Enabled, assignment.CreatedAt, assignment.UpdatedAt)

	assignment.Priority = 5
	require.NoError(t, repo.Update(context.Background(), assignment))
}

func TestRulepackAssignmentRepository_Update_Validation(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()
	db.EnsureTestSchema(t)

	pool := &Pool{inner: db.pool}
	repo := RulepackAssignmentRepo(pool)

	// Empty scope
	bad1 := &domain.RulepackAssignment{ID: uuid.New(), TargetScope: "", Priority: 1, Enabled: true}
	err := repo.Update(context.Background(), bad1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target scope")

	// Non-positive priority
	bad2 := &domain.RulepackAssignment{ID: uuid.New(), TargetScope: "/api/x", Priority: 0, Enabled: true}
	err = repo.Update(context.Background(), bad2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "priority")
}

func TestRulepackAssignmentRepository_Delete(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()
	db.EnsureTestSchema(t)

	db.RunWithTx(t, func(t *testing.T) {
		tenantID, _, rulepackID, _ := db.SeedTestData(t)
		assignmentID := uuid.New()
		db.Exec(t, `
			INSERT INTO rulepack_assignments (id, tenant_id, rulepack_id, target_scope, priority, enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, assignmentID, tenantID, rulepackID, "/api/v1/users", 1, true, time.Now(), time.Now())

		db.Exec(t, "DELETE FROM rulepack_assignments WHERE id = $1", assignmentID)

		var count int
		err := db.QueryRow(t, "SELECT COUNT(*) FROM rulepack_assignments WHERE id = $1", assignmentID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
