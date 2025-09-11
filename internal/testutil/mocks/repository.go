package mocks

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/contracts"
	"github.com/stretchr/testify/mock"
)

// MockRulepackRepository mocks the RulepackRepository interface
type MockRulepackRepository struct {
	mock.Mock
}

func (m *MockRulepackRepository) Create(ctx context.Context, tenantID uuid.UUID, name, description string) (uuid.UUID, error) {
	args := m.Called(ctx, tenantID, name, description)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockRulepackRepository) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, packID, version, dsl, status, createdBy)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// Atomic create+activate
func (m *MockRulepackRepository) CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, packID, version, dsl, createdBy)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockRulepackRepository) Activate(ctx context.Context, packID, versionID uuid.UUID) error {
	args := m.Called(ctx, packID, versionID)
	return args.Error(0)
}

func (m *MockRulepackRepository) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	args := m.Called(ctx, packID)
	return args.Get(0).(json.RawMessage), args.Int(1), args.Error(2)
}

func (m *MockRulepackRepository) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	args := m.Called(ctx, packID, version)
	return args.Get(0).(json.RawMessage), args.String(1), args.Error(2)
}

func (m *MockRulepackRepository) GetVersionIDByNumber(ctx context.Context, packID uuid.UUID, version int) (uuid.UUID, error) {
	args := m.Called(ctx, packID, version)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockRulepackRepository) GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error) {
	args := m.Called(ctx, packID)
	return args.Get(0).(uuid.UUID), args.Int(1), args.Error(2)
}

func (m *MockRulepackRepository) ActivateLatest(ctx context.Context, packID uuid.UUID) error {
	args := m.Called(ctx, packID)
	return args.Error(0)
}

func (m *MockRulepackRepository) Delete(ctx context.Context, packID uuid.UUID) error {
	args := m.Called(ctx, packID)
	return args.Error(0)
}

func (m *MockRulepackRepository) PurgeOldVersions(ctx context.Context, packID uuid.UUID, keep int) error {
	args := m.Called(ctx, packID, keep)
	return args.Error(0)
}

func (m *MockRulepackRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]contracts.RulepackInfo, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]contracts.RulepackInfo), args.Error(1)
}

// MockTenantRepository mocks the TenantRepository interface
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) Create(ctx context.Context, name string) (uuid.UUID, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockTenantRepository) Get(ctx context.Context, id uuid.UUID) (interface{}, error) {
	args := m.Called(ctx, id)
	return args.Get(0), args.Error(1)
}

func (m *MockTenantRepository) GetByName(ctx context.Context, name string) (interface{}, error) {
	args := m.Called(ctx, name)
	return args.Get(0), args.Error(1)
}

func (m *MockTenantRepository) List(ctx context.Context) ([]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
}

// MockAssignmentRepository mocks the AssignmentRepository interface
type MockAssignmentRepository struct {
	mock.Mock
}

func (m *MockAssignmentRepository) Create(ctx context.Context, tenantID, rulepackID uuid.UUID, targetScope string, priority int) (uuid.UUID, error) {
	args := m.Called(ctx, tenantID, rulepackID, targetScope, priority)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockAssignmentRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]interface{}, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockAssignmentRepository) GetByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]interface{}, error) {
	args := m.Called(ctx, tenantID, scope)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
}

// MockAuditRepository mocks the AuditRepository interface
type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) Create(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockAuditRepository) List(ctx context.Context, tenantID uuid.UUID, limit int) ([]interface{}, error) {
	args := m.Called(ctx, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
}
