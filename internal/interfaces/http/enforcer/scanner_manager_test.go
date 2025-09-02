package enforcerhttp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/contracts"
	ckeys "github.com/promptshield/promptshield/internal/shared/contextkeys"
)

// minimal mock implementing contracts.RulepackRepository
type mockRulepackRepo struct {
	active bool
}

func (m *mockRulepackRepo) Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockRulepackRepo) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockRulepackRepo) CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockRulepackRepo) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	return json.RawMessage(`{"metadata":{"name":"t"},"rules":[{"id":"kw","level":1,"keywords":["signal"]}]}`), 1, nil
}
func (m *mockRulepackRepo) Activate(ctx context.Context, packID, versionID uuid.UUID) error {
	return nil
}
func (m *mockRulepackRepo) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	return json.RawMessage("{}"), "approved", nil
}
func (m *mockRulepackRepo) GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error) {
	return uuid.New(), 1, nil
}
func (m *mockRulepackRepo) ActivateLatest(ctx context.Context, packID uuid.UUID) error { return nil }
func (m *mockRulepackRepo) Delete(ctx context.Context, packID uuid.UUID) error         { return nil }
func (m *mockRulepackRepo) PurgeOldVersions(ctx context.Context, packID uuid.UUID, keep int) error {
	return nil
}
func (m *mockRulepackRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]contracts.RulepackInfo, error) {
	if m.active {
		return []contracts.RulepackInfo{{ID: uuid.New(), Name: "t", Description: "", Version: 1, Active: true}}, nil
	}
	return []contracts.RulepackInfo{}, nil
}

func TestScannerManager_ActivationAndDeactivation(t *testing.T) {
	repo := &mockRulepackRepo{active: true}
	svc := services.RulepackServiceCstor(repo, nil)
	mgr := NewScannerManagerWithRulepackService(svc, nil)

	// HasActivePolicies means service available
	if !mgr.HasActivePolicies() {
		t.Fatal("expected HasActivePolicies true when service is present")
	}

	tenant := uuid.New().String()
	ctx := context.WithValue(context.Background(), ckeys.TenantID, tenant)
	res, err := mgr.ScanReader(ctx, strings.NewReader("this contains signal"), "x")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(res.Violations) == 0 {
		t.Fatal("expected a violation with active rulepack")
	}

	// Deactivate repo and reload; should yield zero violations
	repo.active = false
	_ = mgr.ReloadRulepacks()
	res2, err := mgr.ScanReader(ctx, strings.NewReader("this contains signal"), "x")
	if err != nil {
		t.Fatalf("scan2: %v", err)
	}
	if len(res2.Violations) != 0 {
		t.Fatalf("expected no violations after deactivation, got %d", len(res2.Violations))
	}
}
