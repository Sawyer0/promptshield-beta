package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	services "github.com/promptshield/promptshield/internal/application/services"
	memrepo "github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
)

func newMuxRulepack(_ *testing.T) http.Handler {
	pdpClient = nil
	pdpOnce = sync.Once{}
	svc := services.RulepackServiceCstor(memrepo.NewRulepackRepository(), nil)
	return NewMux(Options{RulepackService: svc})
}

func TestRulepackEndpoints_PDP_Permit_UploadVersionExport(t *testing.T) {
	ts := setupPDPServer(t, func(p map[string]any) map[string]any { return map[string]any{"decision": "PERMIT"} })
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	// HMAC key for bundle and temp dir
	t.Setenv("PS_RULEPACK_HMAC_KEY", "YnVuZGxlLXRlc3Qtc2VjcmV0")
	t.Setenv("PS_BUNDLE_DIR", t.TempDir())
	mux := newMuxRulepack(t)

	tenantID := uuid.New()
	// Create rulepack (upload)
	body := []byte(`{"metadata":{"name":"permit-rp"}}`)
	req := httptest.NewRequest(http.MethodPost, "/rulepacks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-User-Admin", "true")
	req.Header.Set("X-PS-Tenant-ID", tenantID.String())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var cr struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &cr)

	// Create version 2 with a minimal valid DSL
	vb := []byte(`{"version":2,"dsl":{"apiVersion":"promptshield.io/v1","kind":"RulePack","metadata":{"name":"permit-rp-v2"},"rules":[{"id":"r1","name":"kw","level":1,"keywords":["foo"]}]}}`)
	reqV := httptest.NewRequest(http.MethodPost, "/rulepacks/"+cr.ID+"/versions", bytes.NewReader(vb))
	reqV.Header.Set("Content-Type", "application/json")
	reqV.Header.Set("X-PS-User-Admin", "true")
	reqV.Header.Set("X-PS-Tenant-ID", tenantID.String())
	recV := httptest.NewRecorder()
	mux.ServeHTTP(recV, reqV)
	if recV.Code != http.StatusCreated {
		t.Fatalf("version create expected 201, got %d body=%s", recV.Code, recV.Body.String())
	}

	// Export bundle for version 1
	reqB := httptest.NewRequest(http.MethodGet, "/rulepacks/"+cr.ID+"/versions/1/bundle", nil)
	reqB.Header.Set("X-PS-User-Admin", "true")
	reqB.Header.Set("X-PS-Tenant-ID", tenantID.String())
	recB := httptest.NewRecorder()
	mux.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusOK {
		t.Fatalf("export expected 200, got %d body=%s", recB.Code, recB.Body.String())
	}
}

func TestRulepack_ActivateLatest_MissingIfMatch(t *testing.T) {
	ts := setupPDPServer(t, func(p map[string]any) map[string]any { return map[string]any{"decision": "PERMIT"} })
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	mux := newMuxRulepack(t)

	tenantID := uuid.New()
	// create rulepack to get ID
	body := []byte(`{"metadata":{"name":"x"}}`)
	reqC := httptest.NewRequest(http.MethodPost, "/rulepacks", bytes.NewReader(body))
	reqC.Header.Set("Content-Type", "application/json")
	reqC.Header.Set("X-PS-User-Admin", "true")
	reqC.Header.Set("X-PS-Tenant-ID", tenantID.String())
	recC := httptest.NewRecorder()
	mux.ServeHTTP(recC, reqC)
	if recC.Code != http.StatusCreated {
		t.Fatalf("upload expected 201, got %d", recC.Code)
	}
	var cr struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(recC.Body.Bytes(), &cr)

	// Call activate-latest without If-Match -> 428
	reqA := httptest.NewRequest(http.MethodPut, "/rulepacks/active", bytes.NewReader([]byte(`{"id":"`+cr.ID+`"}`)))
	reqA.Header.Set("Content-Type", "application/json")
	reqA.Header.Set("X-PS-User-Admin", "true")
	reqA.Header.Set("X-PS-Tenant-ID", tenantID.String())
	recA := httptest.NewRecorder()
	mux.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing If-Match expected 428, got %d body=%s", recA.Code, recA.Body.String())
	}
}
