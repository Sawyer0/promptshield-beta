package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantRepository_Create_Get_List_Update_Delete(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	pool := &Pool{inner: db.pool}
	repo := TenantRepo(pool)
	ctx := context.Background()

	// Create tenants (use unique names)
	idSuffix := uuid.New().String()
	tenants := []domain.Tenant{
		{ID: uuid.New(), Name: fmt.Sprintf("Tenant One %s", idSuffix), Status: domain.TenantStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), Name: fmt.Sprintf("Tenant Two %s", idSuffix), Status: domain.TenantStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for i := range tenants {
		require.NoError(t, repo.Create(ctx, &tenants[i]))
	}

	// GetByName works
	gotByName, err := repo.GetByName(ctx, tenants[0].Name)
	require.NoError(t, err)
	assert.Equal(t, tenants[0].Name, gotByName.Name)

	// Get works
	got, err := repo.Get(ctx, tenants[0].ID)
	require.NoError(t, err)
	assert.Equal(t, tenants[0].ID, got.ID)

	// List works (use a larger page size to avoid missing newly created tenants due to ordering)
	list, total, err := repo.List(ctx, 0, 1000)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)

	// Ensure our created tenants are present
	foundIDs := map[uuid.UUID]bool{}
	for _, tnt := range list {
		foundIDs[tnt.ID] = true
	}
	assert.True(t, foundIDs[tenants[0].ID])
	assert.True(t, foundIDs[tenants[1].ID])

	// Update
	gotByName.Name = tenants[0].Name + " Updated"
	require.NoError(t, repo.Update(ctx, gotByName))
	refetch, err := repo.Get(ctx, gotByName.ID)
	require.NoError(t, err)
	assert.Equal(t, tenants[0].Name+" Updated", refetch.Name)

	// Delete
	require.NoError(t, repo.Delete(ctx, gotByName.ID))
	_, err = repo.Get(ctx, gotByName.ID)
	require.Error(t, err)
}
