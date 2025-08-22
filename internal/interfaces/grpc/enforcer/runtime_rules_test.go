package grpcenforcer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockRepo for testing runtime rule loading
type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error) {
	args := m.Called(ctx, tenantID, name, desc)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockRepo) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, packID, version, dsl, status, createdBy)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockRepo) CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, packID, version, dsl, createdBy)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockRepo) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	args := m.Called(ctx, packID)
	return args.Get(0).(json.RawMessage), args.Int(1), args.Error(2)
}

func (m *mockRepo) Activate(ctx context.Context, packID, versionID uuid.UUID) error {
	args := m.Called(ctx, packID, versionID)
	return args.Error(0)
}

func (m *mockRepo) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	args := m.Called(ctx, packID, version)
	return args.Get(0).(json.RawMessage), args.String(1), args.Error(2)
}

func (m *mockRepo) GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error) {
	args := m.Called(ctx, packID)
	return args.Get(0).(uuid.UUID), args.Int(1), args.Error(2)
}

func (m *mockRepo) ActivateLatest(ctx context.Context, packID uuid.UUID) error {
	args := m.Called(ctx, packID)
	return args.Error(0)
}

func (m *mockRepo) Delete(ctx context.Context, packID uuid.UUID) error {
	args := m.Called(ctx, packID)
	return args.Error(0)
}

func (m *mockRepo) PurgeOldVersions(ctx context.Context, packID uuid.UUID, keep int) error {
	args := m.Called(ctx, packID, keep)
	return args.Error(0)
}

func (m *mockRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]contracts.RulepackInfo, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]contracts.RulepackInfo), args.Error(1)
}

func TestEnforcer_RuntimeRuleLoading(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	packID := uuid.New()
	
	// Mock repository with active rulepack
	repo := &mockRepo{}
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{
		{
			ID:          packID,
			Name:        "test-rules",
			Description: "Test rules",
			Version:     1,
			Active:      true,
		},
	}, nil)
	
	// Mock DSL content
	dslContent := `
metadata:
  name: test-rules
  description: Test rules
rules:
  - id: test-rule
    level: 1
    keywords: ["malicious"]
    severity: HIGH
    category: security
`
	
	repo.On("GetActive", mock.Anything, packID).Return(json.RawMessage(dslContent), 1, nil)
	
	// Create enforcer with database repository
	opts := Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
	}
	
	server := NewWithOptions(opts)
	
	// Verify server was created and rules loaded
	assert.NotNil(t, server)
	assert.Equal(t, tenantID, server.tenantID)
	assert.Equal(t, repo, server.rulepackRepo)
	
	// Test runtime rule reload
	ctx := context.Background()
	err := server.ReloadRules(ctx)
	assert.NoError(t, err)
	
	// Verify mocks were called
	repo.AssertExpectations(t)
}

func TestEnforcer_FailOpenWhenNoDatabaseRules(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	
	// Mock repository with no active rulepacks
	repo := &mockRepo{}
	repo.On("ListByTenant", mock.Anything, tenantID).Return([]contracts.RulepackInfo{}, nil)
	
	// Create enforcer with database repository
	opts := Options{
		Timeout:      100 * time.Millisecond,
		RulepackRepo: repo,
		TenantID:     tenantID,
	}
	
	server := NewWithOptions(opts)
	
	// Should fail-open (ready=true) even with no database rules
	assert.True(t, server.ready)
	
	repo.AssertExpectations(t)
}

func TestEnforcer_NoRepositoryFallsBackToFiles(t *testing.T) {
	// Create enforcer without database repository
	opts := Options{
		Timeout: 100 * time.Millisecond,
		// No RulepackRepo provided
	}
	
	server := NewWithOptions(opts)
	
	// Should still be created (may fail-open depending on file availability)
	assert.NotNil(t, server)
	assert.Nil(t, server.rulepackRepo)
}