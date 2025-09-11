package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise real Postgres repository behavior for rulepacks.
// They will be skipped automatically when PS_TEST_PG_DSN is not set.

func TestRulepackRepository_FullFlow(t *testing.T) {
	// Arrange
	db := NewTestDB(t)
	defer db.Close()

	pool := &Pool{inner: db.pool}
	repo := RulepackRepo(pool)
	ctx := context.Background()

	// Seed tenant and user
	tenantID, userID, _, _ := db.SeedTestData(t)
	db.EnsureTestSchema(t)

	// Create rulepack
	packID, err := repo.Create(ctx, tenantID, "Security Policy", "Basic rules")
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, packID)

	// Create version 1 (draft) and approve it
	dslV1 := json.RawMessage(`{"rules":[]}`)
	ver1ID, err := repo.CreateVersion(ctx, packID, 1, dslV1, "draft", userID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, ver1ID)

	require.NoError(t, repo.ApproveVersion(ctx, packID, 1))

	// Activate version 1
	require.NoError(t, repo.ActivateLatest(ctx, packID))

	// Get active
	activeDSL, activeVer, err := repo.GetActive(ctx, packID)
	require.NoError(t, err)
	assert.Equal(t, 1, activeVer)
	assert.JSONEq(t, string(dslV1), string(activeDSL))

	// Create version 2 via atomic create+activate
	dslV2 := json.RawMessage(`{"rules":[{"id":"r1"}]}`)
	ver2ID, err := repo.CreateVersionActivateTx(ctx, packID, 2, dslV2, userID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, ver2ID)

	// Active should now be v2
	activeDSL2, activeVer2, err := repo.GetActive(ctx, packID)
	require.NoError(t, err)
	assert.Equal(t, 2, activeVer2)
	assert.JSONEq(t, string(dslV2), string(activeDSL2))

	// List by tenant should include our pack with active true and version >=2
	infos, err := repo.ListByTenant(ctx, tenantID)
	require.NoError(t, err)
	found := false
	for _, info := range infos {
		if info.ID == packID {
			found = true
			assert.True(t, info.Active)
			assert.GreaterOrEqual(t, info.Version, 2)
		}
	}
	assert.True(t, found, "created rulepack must be present in list")

	// Get latest version metadata should point to v2
	latestID, latestNum, err := repo.GetLatestVersion(ctx, packID)
	require.NoError(t, err)
	assert.Equal(t, ver2ID, latestID)
	assert.Equal(t, 2, latestNum)

	// Purge old versions, retain last 1 (should keep v2)
	require.NoError(t, repo.PurgeOldVersions(ctx, packID, 1))

	// v1 may or may not be deleted depending on active pointer; ensure active still valid
	_, verAfterPurge, err := repo.GetActive(ctx, packID)
	require.NoError(t, err)
	assert.Equal(t, 2, verAfterPurge)

	// Delete rulepack (null out current_version_id first to satisfy FK)
	_, _ = db.pool.Exec(ctx, `UPDATE rulepacks SET current_version_id = NULL WHERE id = $1`, packID)
	require.NoError(t, repo.Delete(ctx, packID))

	// Ensure it is gone
	_, _, err = repo.GetActive(ctx, packID)
	assert.Error(t, err)
}

func TestRulepackRepository_GetVersion_NotFound(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	pool := &Pool{inner: db.pool}
	repo := RulepackRepo(pool)
	ctx := context.Background()

	// Query non-existent version
	_, status, err := repo.GetVersion(ctx, uuid.New(), 99)
	assert.Error(t, err)
	assert.Equal(t, "", status)
}

func TestRulepackRepository_HealthCheck(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	pool := &Pool{inner: db.pool}
	repo := RulepackRepo(pool)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, repo.HealthCheck(ctx))
}
