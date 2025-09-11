package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"sync"

	"github.com/google/uuid"
	repo "github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
	services "github.com/promptshield/promptshield/internal/application/services"
)

// helper to create mux with memory repo (duplicate for clarity in this file)
func newMuxWithMemoryRepo(t *testing.T) http.Handler {
	pdpClient = nil
	pdpOnce = sync.Once{}
	svc := services.RulepackServiceCstor(repo.NewRulepackRepository(), nil)
	return NewMux(Options{RulepackService: svc, ToolRunner: &stubToolRunner{}})
}

func TestRulepackEndpoints_PDP_DenyMatrix(t *testing.T) {
	// Start with permit PDP to create an initial rulepack
	tsPermit := setupPDPServer(t, func(p map[string]any) map[string]any { return map[string]any{"decision":"PERMIT"} })
	defer tsPermit.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", tsPermit.URL)
	mux := newMuxWithMemoryRepo(t)

	tenantID := uuid.New()
	// Create a rulepack to obtain an ID
	body := []byte(`{"metadata":{"name":"rp1"}}`)
	reqCreate := httptest.NewRequest(http.MethodPost, "/rulepacks", bytes.NewReader(body))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("X-PS-User-Admin", "true")
	reqCreate.Header.Set("X-PS-Tenant-ID", tenantID.String())
	recCreate := httptest.NewRecorder()
	mux.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated { t.Fatalf("create rulepack expected 201, got %d body=%s", recCreate.Code, recCreate.Body.String()) }
	var cr struct{ ID string `json:"id"` }
	_ = json.Unmarshal(recCreate.Body.Bytes(), &cr)
	packID := cr.ID

	// Switch to deny PDP via admin reload
	tsDeny := setupPDPServer(t, func(p map[string]any) map[string]any {
		if act, _ := p["action"].(string); act != "" {
			return map[string]any{"decision":"DENY","reason":"blocked"}
		}
		return map[string]any{"decision":"DENY"}
	})
	defer tsDeny.Close()
	t.Setenv("PS_PDP_ENDPOINT", tsDeny.URL)
	reqReload := httptest.NewRequest(http.MethodPost, "/admin/pdp/reload", nil)
	reqReload.Header.Set("X-PS-User-Admin", "true")
	recReload := httptest.NewRecorder()
	mux.ServeHTTP(recReload, reqReload)
	if recReload.Code != http.StatusNoContent { t.Fatalf("pdp reload expected 204, got %d", recReload.Code) }

	tests := []struct{
		name string
		method string
		url string
		body []byte
		expected int
		headers map[string]string
	}{
		{"upload_deny", http.MethodPost, "/rulepacks", []byte(`{"metadata":{"name":"x"}}`), http.StatusForbidden, map[string]string{"Content-Type":"application/json"}},
		{"delete_deny", http.MethodDelete, "/rulepacks/"+packID, nil, http.StatusForbidden, nil},
		{"version_create_deny", http.MethodPost, "/rulepacks/"+packID+"/versions", []byte(`{"version":2,"dsl":{}}`), http.StatusForbidden, map[string]string{"Content-Type":"application/json"}},
		{"bundle_export_deny", http.MethodGet, "/rulepacks/"+packID+"/versions/1/bundle", nil, http.StatusForbidden, nil},
		{"bundle_publish_deny", http.MethodPost, "/rulepacks/"+packID+"/versions/1/publish", nil, http.StatusForbidden, nil},
		{"bundle_list_deny", http.MethodGet, "/rulepacks/"+packID+"/bundles", nil, http.StatusForbidden, nil},
		{"bundle_get_deny", http.MethodGet, "/rulepacks/"+packID+"/bundles/1", nil, http.StatusForbidden, nil},
		{"bundle_verify_deny", http.MethodPost, "/rulepacks/"+packID+"/bundles/verify", []byte(`{}`), http.StatusForbidden, map[string]string{"Content-Type":"application/json"}},
		{"bundle_activate_deny", http.MethodPost, "/rulepacks/"+packID+"/bundles/1/activate", nil, http.StatusForbidden, nil},
		{"activate_latest_deny", http.MethodPut, "/rulepacks/active", []byte(`{"id":"`+packID+`"}`), http.StatusForbidden, map[string]string{"If-Match":"*","Content-Type":"application/json"}},
		{"version_activate_deny", http.MethodPost, "/rulepacks/"+packID+"/versions/1/activate", []byte(`{"tenantId":"`+tenantID.String()+`","dsl":{}}`), http.StatusForbidden, map[string]string{"Content-Type":"application/json"}},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.url, bytes.NewReader(tc.body))
		req.Header.Set("X-PS-User-Admin", "true")
		req.Header.Set("X-PS-Tenant-ID", tenantID.String())
		for k, v := range tc.headers { req.Header.Set(k, v) }
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != tc.expected {
			t.Fatalf("%s: expected %d, got %d body=%s", tc.name, tc.expected, rec.Code, rec.Body.String())
		}
	}
}
