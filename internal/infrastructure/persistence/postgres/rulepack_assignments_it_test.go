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

// Integration tests against Postgres for RulepackAssignmentRepository

func TestRulepackAssignmentRepository_CRUD_AndOrdering(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	pool := &Pool{inner: db.pool}
	repo := RulepackAssignmentRepo(pool)
	ctx := context.Background()

	tenantID, _, rulepackID, _ := db.SeedTestData(t)
	db.EnsureTestSchema(t)

	// Insert a second rulepack for same tenant to allow two assignments on same scope
	rulepackID2 := uuid.New()
	db.Exec(t, `INSERT INTO rulepacks (id, tenant_id, name, description, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW())`,
		rulepackID2, tenantID, "Another Pack", "desc",
	)

	// Create three assignments (two share the same scope)
	a1 := &domain.RulepackAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RulepackID:  rulepackID,
		TargetScope: "/api/v1/users",
		Priority:    500,
		Enabled:     true,
		CreatedAt:   time.Now().Add(-3 * time.Second),
		UpdatedAt:   time.Now().Add(-3 * time.Second),
	}
	a2 := &domain.RulepackAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RulepackID:  rulepackID2,
		TargetScope: "/api/v1/users",
		Priority:    750,
		Enabled:     true,
		CreatedAt:   time.Now().Add(-2 * time.Second),
		UpdatedAt:   time.Now().Add(-2 * time.Second),
	}
	a3 := &domain.RulepackAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RulepackID:  rulepackID,
		TargetScope: "/api/v1/admin",
		Priority:    250,
		Enabled:     true,
		CreatedAt:   time.Now().Add(-1 * time.Second),
		UpdatedAt:   time.Now().Add(-1 * time.Second),
	}

	require.NoError(t, repo.Create(ctx, a1))
	require.NoError(t, repo.Create(ctx, a2))
	require.NoError(t, repo.Create(ctx, a3))

	// List by tenant - expect sorted by priority DESC then created_at ASC
	list, err := repo.ListByTenant(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, a2.ID, list[0].ID) // highest priority first
	assert.Equal(t, a1.ID, list[1].ID)
	assert.Equal(t, a3.ID, list[2].ID)

	// List by scope for /api/v1/users
	scoped, err := repo.ListByScope(ctx, tenantID, "/api/v1/users")
	require.NoError(t, err)
	require.Len(t, scoped, 2)
	assert.Equal(t, a2.ID, scoped[0].ID)
	assert.Equal(t, a1.ID, scoped[1].ID)

	// Update: raise a1 above a2
	a1.Priority = 900
	a1.UpdatedAt = time.Now()
	require.NoError(t, repo.Update(ctx, a1))

	// Check ordering updated
	scopedAfter, err := repo.ListByScope(ctx, tenantID, "/api/v1/users")
	require.NoError(t, err)
	require.Len(t, scopedAfter, 2)
	assert.Equal(t, a1.ID, scopedAfter[0].ID)
	assert.Equal(t, a2.ID, scopedAfter[1].ID)

	// Delete a3
	require.NoError(t, repo.Delete(ctx, a3.ID))
	listAfterDelete, err := repo.ListByTenant(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, listAfterDelete, 2)
}
