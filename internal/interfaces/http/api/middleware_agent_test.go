package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
)

// stub RowScanner
type testRow struct {
	tool  *testTool
	query string
	args  []interface{}
}

func (r *testRow) Scan(dest ...interface{}) error {
	// If no tool configured, simulate no rows
	if r.tool == nil {
		return sql.ErrNoRows
	}
	// Populate fields expected by Tools.GetByToolID
	if len(dest) >= 13 && strings.Contains(strings.ToLower(r.query), "from tools") {
		idPtr := dest[0].(*uuid.UUID)
		tenantPtr := dest[1].(*uuid.UUID)
		toolIDPtr := dest[2].(*string)
		namePtr := dest[3].(*string)
		descPtr := dest[4].(*string)
		capsPtr := dest[5].(*string)
		domsPtr := dest[6].(*string)
		sidePtr := dest[7].(*string)
		authPtr := dest[8].(*string)
		argPtr := dest[9].(*json.RawMessage)
		riskPtr := dest[10].(*sql.NullInt32)
		createdPtr := dest[11].(*time.Time)
		updatedPtr := dest[12].(*time.Time)

		*idPtr = r.tool.ID
		*tenantPtr = r.tool.TenantID
		*toolIDPtr = r.tool.ToolID
		*namePtr = r.tool.Name
		*descPtr = r.tool.Description
		*capsPtr = r.tool.CapabilityTagsJSON
		*domsPtr = r.tool.DataDomainsJSON
		*sidePtr = r.tool.SideEffect
		*authPtr = r.tool.AuthScope
		*argPtr = json.RawMessage(r.tool.ArgSchemaJSON)
		*riskPtr = sql.NullInt32{Valid: false}
		*createdPtr = time.Now().UTC()
		*updatedPtr = time.Now().UTC()
		return nil
	}
	return sql.ErrNoRows
}

// stub DB implementing postgres.DB
type testDB struct {
	tool *testTool
}

func (db *testDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) pg.RowScanner {
	return &testRow{tool: db.tool, query: query, args: args}
}
func (db *testDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (db *testDB) QueryContext(ctx context.Context, query string, args ...interface{}) (pg.RowsScanner, error) {
	return nil, nil
}

// test tool record
type testTool struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	ToolID             string
	Name               string
	Description        string
	CapabilityTagsJSON string
	DataDomainsJSON    string
	SideEffect         string
	AuthScope          string
	ArgSchemaJSON      string
}

func newMuxWith(t *testing.T, db pg.DB) http.Handler {
	repo := memory.NewRulepackRepository()
	svc := services.RulepackServiceCstor(repo, nil)
	opt := Options{RulepackService: svc, DB: db}
	return NewMux(opt)
}

func activateRulepack(t *testing.T, svc *services.RulepackService, tenantID uuid.UUID, dsl any) {
	data, _ := json.Marshal(dsl)
	packID, err := svc.Create(context.Background(), tenantID, "p", "")
	if err != nil {
		t.Fatalf("create pack: %v", err)
	}
	if err := svc.CreateVersionActivate(context.Background(), tenantID, packID, 1, data); err != nil {
		t.Fatalf("activate pack: %v", err)
	}
}

func TestAgentMiddleware_DenyUnknownTool(t *testing.T) {
	tenant := uuid.New()
	h := newMuxWith(t, &testDB{tool: nil})
	req := httptest.NewRequest(http.MethodPost, "/v1/anything", bytes.NewReader([]byte(`{"x":1}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tool-ID", "unknown-tool")
	req.Header.Set("X-PS-Tenant-ID", tenant.String())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
	if v := rr.Header().Get("X-PS-Decision"); v != "deny" {
		t.Fatalf("want decision=deny, got %q", v)
	}
	if v := rr.Header().Get("X-PS-Policy"); v != "action_selector" {
		t.Fatalf("want policy=action_selector, got %q", v)
	}
}

func TestAgentMiddleware_AllowWithPatternsAndTool(t *testing.T) {
	tenant := uuid.New()
	tool := &testTool{
		ID: uuid.New(), TenantID: tenant, ToolID: "search_web", Name: "Search",
		Description:        "",
		CapabilityTagsJSON: `["read","network_get"]`,
		DataDomainsJSON:    `[]`,
		SideEffect:         "none", AuthScope: "user-delegated",
		ArgSchemaJSON: `{"params":[]}`,
	}
	db := &testDB{tool: tool}
	repo := memory.NewRulepackRepository()
	svc := services.RulepackServiceCstor(repo, nil)
	opt := Options{RulepackService: svc, DB: db}
	h := NewMux(opt)
	// Activate rulepack with action selector allowing read AND network_get
	activateRulepack(t, svc, tenant, map[string]any{
		"apiVersion": "unit/v1", "kind": "RulePack", "metadata": map[string]any{"name": "p"},
		"rules": []any{}, "composition": map[string]any{"priority": 10},
		"patterns": map[string]any{"action_selector": map[string]any{"enabled": true, "allowed_tool_query": "read AND network_get"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/do", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-PS-Tool-ID", tool.ToolID)
	req.Header.Set("X-PS-Tenant-ID", tenant.String())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-PS-Decision") != "allow" {
		t.Fatalf("expected allow, got %q", rr.Header().Get("X-PS-Decision"))
	}
}

func TestAgentMiddleware_ArgContracts_DenyFreeText(t *testing.T) {
	tenant := uuid.New()
	tool := &testTool{
		ID: uuid.New(), TenantID: tenant, ToolID: "update_ticket", Name: "Update",
		CapabilityTagsJSON: `[]`, DataDomainsJSON: `[]`, SideEffect: "reversible", AuthScope: "user-delegated",
		ArgSchemaJSON: `{"params":[{"name":"id","type":"string","required":true},{"name":"text","type":"string"}]}`,
	}
	db := &testDB{tool: tool}
	repo := memory.NewRulepackRepository()
	svc := services.RulepackServiceCstor(repo, nil)
	opt := Options{RulepackService: svc, DB: db}
	h := NewMux(opt)
	activateRulepack(t, svc, tenant, map[string]any{
		"apiVersion": "unit/v1", "kind": "RulePack", "metadata": map[string]any{"name": "p"},
		"rules":    []any{},
		"patterns": map[string]any{"action_selector": map[string]any{"enabled": true, "allowed_tool_query": ""}},
		"preset":   map[string]any{"arg_contracts": []any{"ids_only_updates"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/do", bytes.NewReader([]byte(`{"id":"123","text":"hello"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tool-ID", tool.ToolID)
	req.Header.Set("X-PS-Tenant-ID", tenant.String())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
	if rr.Header().Get("X-PS-Policy") != "arg_contracts" {
		t.Fatalf("want policy=arg_contracts, got %q", rr.Header().Get("X-PS-Policy"))
	}
}

func TestAgentMiddleware_DualLLM_QuarantinedDenied(t *testing.T) {
	tenant := uuid.New()
	tool := &testTool{ID: uuid.New(), TenantID: tenant, ToolID: "writer", Name: "Writer",
		CapabilityTagsJSON: `["write"]`, DataDomainsJSON: `[]`, SideEffect: "irreversible", AuthScope: "user-delegated", ArgSchemaJSON: `{"params":[]}`,
	}
	db := &testDB{tool: tool}
	repo := memory.NewRulepackRepository()
	svc := services.RulepackServiceCstor(repo, nil)
	opt := Options{RulepackService: svc, DB: db}
	h := NewMux(opt)
	activateRulepack(t, svc, tenant, map[string]any{
		"apiVersion": "unit/v1", "kind": "RulePack", "metadata": map[string]any{"name": "p"},
		"rules":    []any{},
		"patterns": map[string]any{"dual_llm": map[string]any{"enabled": true, "quarantined_tools_disabled": true}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/do", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-PS-Tool-ID", tool.ToolID)
	req.Header.Set("X-PS-Tenant-ID", tenant.String())
	req.Header.Set("X-PS-Lane", "quarantined")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
	if rr.Header().Get("X-PS-Policy") != "dual_llm" {
		t.Fatalf("want dual_llm, got %q", rr.Header().Get("X-PS-Policy"))
	}
}

func TestAgentMiddleware_PlanRequired(t *testing.T) {
	tenant := uuid.New()
	tool := &testTool{ID: uuid.New(), TenantID: tenant, ToolID: "writer", Name: "Writer",
		CapabilityTagsJSON: `[]`, DataDomainsJSON: `[]`, SideEffect: "none", AuthScope: "user-delegated", ArgSchemaJSON: `{"params":[]}`,
	}
	db := &testDB{tool: tool}
	repo := memory.NewRulepackRepository()
	svc := services.RulepackServiceCstor(repo, nil)
	opt := Options{RulepackService: svc, DB: db}
	h := NewMux(opt)
	activateRulepack(t, svc, tenant, map[string]any{
		"apiVersion": "unit/v1", "kind": "RulePack", "metadata": map[string]any{"name": "p"},
		"rules":    []any{},
		"patterns": map[string]any{"plan_then_execute": map[string]any{"enabled": true, "max_steps": 5}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/do", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-PS-Tool-ID", tool.ToolID)
	req.Header.Set("X-PS-Tenant-ID", tenant.String())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-PS-Policy") != "plan_then_execute" {
		t.Fatalf("want plan_then_execute, got %q", rr.Header().Get("X-PS-Policy"))
	}
}
