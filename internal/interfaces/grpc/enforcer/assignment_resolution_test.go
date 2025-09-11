package grpcenforcer

import (
    "context"
    "encoding/json"
    "testing"

    "github.com/google/uuid"
    dom "github.com/promptshield/promptshield/internal/domain"
    repomock "github.com/promptshield/promptshield/internal/repository"
    tmocks "github.com/promptshield/promptshield/internal/testutil/mocks"
)

func TestResolveAssignedRulepacks_MethodFiltering(t *testing.T) {
    ctx := context.Background()
    tenantID := uuid.New()

    // Prepare mock assignment repo with three assignments
    arepo := repomock.NewMockRulepackAssignmentRepository()
    rp1 := uuid.New()
    rp2 := uuid.New()
    rp3 := uuid.New()

    // POST assignment for /v1/* -> rp1
    _ = arepo.Create(ctx, &dom.RulepackAssignment{
        ID:          uuid.New(),
        TenantID:    tenantID,
        RulepackID:  rp1,
        Method:      "POST",
        TargetScope: "/v1/*",
        Priority:    10,
        Enabled:     true,
    })
    // GET assignment for /v1/* -> rp2
    _ = arepo.Create(ctx, &dom.RulepackAssignment{
        ID:          uuid.New(),
        TenantID:    tenantID,
        RulepackID:  rp2,
        Method:      "GET",
        TargetScope: "/v1/*",
        Priority:    10,
        Enabled:     true,
    })
    // ANY method assignment for /v2/* -> rp3
    _ = arepo.Create(ctx, &dom.RulepackAssignment{
        ID:          uuid.New(),
        TenantID:    tenantID,
        RulepackID:  rp3,
        Method:      "*",
        TargetScope: "/v2/*",
        Priority:    5,
        Enabled:     true,
    })

    // Mock rulepack repo to return minimal DSL for each pack
    rrepo := new(tmocks.MockRulepackRepository)
    dsl1 := []byte("apiVersion: promptshield.io/v1\nkind: RulePack\nmetadata:\n  name: rp1\nrules: []\n")
    dsl2 := []byte("apiVersion: promptshield.io/v1\nkind: RulePack\nmetadata:\n  name: rp2\nrules: []\n")
    dsl3 := []byte("apiVersion: promptshield.io/v1\nkind: RulePack\nmetadata:\n  name: rp3\nrules: []\n")
rrepo.On("GetActive", ctx, rp1).Return(json.RawMessage(dsl1), 1, nil)
rrepo.On("GetActive", ctx, rp2).Return(json.RawMessage(dsl2), 1, nil)
rrepo.On("GetActive", ctx, rp3).Return(json.RawMessage(dsl3), 1, nil)

    s := &Server{assignmentRepo: arepo, rulepackRepo: rrepo}

    // Match POST /v1/chat/completions -> should resolve rp1 only
    packs, err := s.resolveAssignedRulepacks(ctx, tenantID, "/v1/chat/completions", "POST")
    if err != nil {
        t.Fatalf("resolveAssignedRulepacks error: %v", err)
    }
    if len(packs) != 1 {
        t.Fatalf("expected 1 pack, got %d", len(packs))
    }
    if packs[0].Metadata.Name != "rp1" {
        t.Fatalf("expected pack rp1, got %s", packs[0].Metadata.Name)
    }

    // Match GET /v1/chat/completions -> should resolve rp2 only
    packs, err = s.resolveAssignedRulepacks(ctx, tenantID, "/v1/chat/completions", "GET")
    if err != nil {
        t.Fatalf("resolveAssignedRulepacks error: %v", err)
    }
    if len(packs) != 1 || packs[0].Metadata.Name != "rp2" {
        t.Fatalf("expected pack rp2, got %v", packs)
    }

    // Match DELETE /v2/models -> should resolve rp3 (method "*")
    packs, err = s.resolveAssignedRulepacks(ctx, tenantID, "/v2/models", "DELETE")
    if err != nil {
        t.Fatalf("resolveAssignedRulepacks error: %v", err)
    }
    if len(packs) != 1 || packs[0].Metadata.Name != "rp3" {
        t.Fatalf("expected pack rp3, got %v", packs)
    }
}

