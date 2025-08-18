package interfaces_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/contracts"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository for testing without real database
type MockRepository struct {
	mock.Mock
	mu           sync.RWMutex
	rulepacks    map[uuid.UUID]*RulepackData
	activeByPack map[uuid.UUID]uuid.UUID // packID -> versionID
}

type RulepackData struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	Versions    map[uuid.UUID]VersionData
}

type VersionData struct {
	ID      uuid.UUID
	Version int
	DSL     json.RawMessage
	Status  string
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		rulepacks:    make(map[uuid.UUID]*RulepackData),
		activeByPack: make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *MockRepository) Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New()
	m.rulepacks[id] = &RulepackData{
		ID:          id,
		TenantID:    tenantID,
		Name:        name,
		Description: desc,
		Versions:    make(map[uuid.UUID]VersionData),
	}

	return id, nil
}

func (m *MockRepository) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pack, exists := m.rulepacks[packID]
	if !exists {
		return uuid.Nil, assert.AnError
	}

	versionID := uuid.New()
	pack.Versions[versionID] = VersionData{
		ID:      versionID,
		Version: version,
		DSL:     dsl,
		Status:  status,
	}

	return versionID, nil
}

func (m *MockRepository) CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error) {
	versionID, err := m.CreateVersion(ctx, packID, version, dsl, "approved", createdBy)
	if err != nil {
		return uuid.Nil, err
	}

	m.mu.Lock()
	m.activeByPack[packID] = versionID
	m.mu.Unlock()

	return versionID, nil
}

func (m *MockRepository) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versionID, hasActive := m.activeByPack[packID]
	if !hasActive {
		return nil, 0, assert.AnError
	}

	pack, exists := m.rulepacks[packID]
	if !exists {
		return nil, 0, assert.AnError
	}

	version, exists := pack.Versions[versionID]
	if !exists {
		return nil, 0, assert.AnError
	}

	return version.DSL, version.Version, nil
}

func (m *MockRepository) Activate(ctx context.Context, packID, versionID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeByPack[packID] = versionID
	return nil
}

func (m *MockRepository) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pack, exists := m.rulepacks[packID]
	if !exists {
		return nil, "", assert.AnError
	}

	for _, v := range pack.Versions {
		if v.Version == version {
			return v.DSL, v.Status, nil
		}
	}

	return nil, "", assert.AnError
}

func (m *MockRepository) GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pack, exists := m.rulepacks[packID]
	if !exists {
		return uuid.Nil, 0, assert.AnError
	}

	var latestID uuid.UUID
	var latestVersion int

	for id, v := range pack.Versions {
		if v.Version > latestVersion {
			latestVersion = v.Version
			latestID = id
		}
	}

	return latestID, latestVersion, nil
}

func (m *MockRepository) ActivateLatest(ctx context.Context, packID uuid.UUID) error {
	latestID, _, err := m.GetLatestVersion(ctx, packID)
	if err != nil {
		return err
	}

	return m.Activate(ctx, packID, latestID)
}

func (m *MockRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]contracts.RulepackInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []contracts.RulepackInfo

	for packID, pack := range m.rulepacks {
		if pack.TenantID == tenantID {
			activeID, hasActive := m.activeByPack[packID]
			var activeVersion int

			if hasActive {
				if version, exists := pack.Versions[activeID]; exists {
					activeVersion = version.Version
				}
			}

			result = append(result, contracts.RulepackInfo{
				ID:          pack.ID,
				Name:        pack.Name,
				Description: pack.Description,
				Version:     activeVersion,
				Active:      hasActive,
			})
		}
	}

	return result, nil
}

func (m *MockRepository) Delete(ctx context.Context, packID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.rulepacks, packID)
	delete(m.activeByPack, packID)
	return nil
}

func (m *MockRepository) PurgeOldVersions(ctx context.Context, packID uuid.UUID, keep int) error {
	// No-op for mock
	return nil
}

// TestIntegration_RulePropagation tests rule propagation without external dependencies
func TestIntegration_RulePropagation(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Create mock repository
	repo := NewMockRepository()

	// Create publisher with no Redis (becomes no-op)
	publisher, err := nats.NewPublisher("")
	require.NoError(t, err)

	// Message tracking would require wrapping the publisher

	// Create service
	service := services.RulepackServiceCstor(repo, publisher)

	// Create rulepack
	packID, err := service.Create(ctx, tenantID, "test-pack", "Test rulepack")
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, packID)

	// Create multiple enforcers
	numEnforcers := 3
	enforcers := make([]*grpcenforcer.Server, numEnforcers)

	for i := 0; i < numEnforcers; i++ {
		opts := grpcenforcer.Options{
			Timeout:      100 * time.Millisecond,
			RulepackRepo: repo,
			TenantID:     tenantID,
			// No RedisAddr - will poll from database
		}
		enforcers[i] = grpcenforcer.NewWithOptions(opts)
		defer enforcers[i].Shutdown()
	}

	// Upload version 1
	dslV1 := json.RawMessage(`{
		"metadata": {"name": "test-rules", "version": "1.0.0"},
		"rules": [{"id": "rule-1", "level": 1, "keywords": ["test"], "severity": "LOW"}]
	}`)

	versionID, err := service.Upload(ctx, tenantID, packID, 1, dslV1, true)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, versionID)

	// Message publishing is tested separately
	// Here we focus on the rule reload flow

	// Simulate enforcers reloading rules (in production, Redis would trigger this)
	for i, enforcer := range enforcers {
		err := enforcer.ReloadRules(ctx)
		assert.NoError(t, err, "Enforcer %d should reload rules", i)
	}

	// Upload version 2
	dslV2 := json.RawMessage(`{
		"metadata": {"name": "test-rules", "version": "2.0.0"},
		"rules": [
			{"id": "rule-1", "level": 1, "keywords": ["test"], "severity": "LOW"},
			{"id": "rule-2", "level": 1, "keywords": ["new"], "severity": "HIGH"}
		]
	}`)

	versionID2, err := service.Upload(ctx, tenantID, packID, 2, dslV2, true)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, versionID2)

	// Message publishing is tested separately

	// Simulate enforcers reloading again
	for i, enforcer := range enforcers {
		err := enforcer.ReloadRules(ctx)
		assert.NoError(t, err, "Enforcer %d should reload rules v2", i)
	}

	// Delete rulepack
	err = service.Delete(ctx, tenantID, packID)
	require.NoError(t, err)

	// Message publishing is tested separately

	// Enforcers should handle deletion gracefully (fail-open)
	for i, enforcer := range enforcers {
		err := enforcer.ReloadRules(ctx)
		assert.NoError(t, err, "Enforcer %d should handle deletion gracefully", i)
		// Enforcers remain operational even without rules (fail-open)
	}
}

// TestIntegration_ConcurrentRuleUpdates tests concurrent rule updates don't cause races
func TestIntegration_ConcurrentRuleUpdates(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Create mock repository
	repo := NewMockRepository()

	// Create enforcer
	opts := grpcenforcer.Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
	}
	enforcer := grpcenforcer.NewWithOptions(opts)
	defer enforcer.Shutdown()

	// Create rulepack with initial version
	packID, _ := repo.Create(ctx, tenantID, "concurrent-test", "Test concurrent updates")

	dsl := json.RawMessage(`{"metadata": {"name": "test"}, "rules": []}`)
	_, err := repo.CreateVersionActivateTx(ctx, packID, 1, dsl, uuid.Nil)
	require.NoError(t, err)

	// Simulate concurrent rule reloads (like multiple Redis messages arriving)
	var wg sync.WaitGroup
	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(version int) {
			defer wg.Done()

			// Create new version
			versionDSL := json.RawMessage(`{"metadata": {"name": "test"}, "rules": []}`)
			_, _ = repo.CreateVersionActivateTx(ctx, packID, version, versionDSL, uuid.Nil)

			// Trigger reload
			err := enforcer.ReloadRules(ctx)
			assert.NoError(t, err)
		}(i + 2)
	}

	wg.Wait()

	// Enforcer should still be in consistent state
	// (ready field is private, but we can verify it doesn't panic)
}
