package api

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "sync"
    "testing"

    "github.com/google/uuid"
    services "github.com/promptshield/promptshield/internal/application/services"
    memrepo "github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
)

// buildTestMuxWithMemoryRepo resets PDP client and returns a mux with an in-memory repo-backed RulepackService
func buildTestMuxWithMemoryRepo(t *testing.T) http.Handler {
    t.Helper()
    // Reset PDP client between tests (package-level vars)
    pdpClient = nil
    pdpOnce = sync.Once{}
    svc := services.RulepackServiceCstor(memrepo.NewRulepackRepository(), nil)
    opt := Options{ RulepackService: svc, ToolRunner: &stubToolRunner{} }
    return NewMux(opt)
}

// createRulepackViaAPI creates a rulepack by calling POST /rulepacks and returns its ID
func createRulepackViaAPI(t *testing.T, mux http.Handler, tenantID uuid.UUID) uuid.UUID {
    t.Helper()
    body := []byte(`{"metadata":{"name":"bundletest"}}`)
    req := httptest.NewRequest(http.MethodPost, "/rulepacks", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-PS-User-Admin", "true")
    req.Header.Set("X-PS-Tenant-ID", tenantID.String())
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec, req)
    if rec.Code != http.StatusCreated {
        t.Fatalf("create rulepack: expected 201, got %d body=%s", rec.Code, rec.Body.String())
    }
    var resp struct { ID string `json:"id"` }
    if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
        t.Fatalf("unmarshal rulepack response: %v", err)
    }
    id, err := uuid.Parse(resp.ID)
    if err != nil { t.Fatalf("invalid id returned: %v", err) }
    return id
}

func TestBundle_Publish_List_Get_Verify_Activate(t *testing.T) {
    // PDP server that PERMITs everything by default
    ts := setupPDPServer(t, func(p map[string]any) map[string]any { return map[string]any{"decision":"PERMIT"} })
    defer ts.Close()

    // Env setup
    t.Setenv("PS_DEV_BYPASS_AUTH", "true")
    t.Setenv("PS_PDP_ENDPOINT", ts.URL)
    t.Setenv("PS_RULEPACK_HMAC_KEY", "YnVuZGxlLXRlc3Qtc2VjcmV0") // base64("bundle-test-secret")
    tmp := t.TempDir()
    t.Setenv("PS_BUNDLE_DIR", tmp)

    mux := buildTestMuxWithMemoryRepo(t)

    tenantID := uuid.New()
    packID := createRulepackViaAPI(t, mux, tenantID)

    // Publish bundle for version 1
    pubReq := httptest.NewRequest(http.MethodPost, "/rulepacks/"+packID.String()+"/versions/1/publish", nil)
    pubReq.Header.Set("X-PS-User-Admin", "true")
    pubReq.Header.Set("X-PS-Tenant-ID", tenantID.String())
    pubRec := httptest.NewRecorder()
    mux.ServeHTTP(pubRec, pubReq)
    if pubRec.Code != http.StatusOK && pubRec.Code != http.StatusCreated {
        t.Fatalf("publish bundle: expected 200/201, got %d body=%s", pubRec.Code, pubRec.Body.String())
    }

    // List bundles -> expect 1
    listReq := httptest.NewRequest(http.MethodGet, "/rulepacks/"+packID.String()+"/bundles", nil)
    listReq.Header.Set("X-PS-User-Admin", "true")
    listReq.Header.Set("X-PS-Tenant-ID", tenantID.String())
    listRec := httptest.NewRecorder()
    mux.ServeHTTP(listRec, listReq)
    if listRec.Code != http.StatusOK {
        t.Fatalf("list bundles: expected 200, got %d body=%s", listRec.Code, listRec.Body.String())
    }
    var listResp struct { Bundles []map[string]any `json:"bundles"` }
    if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
        t.Fatalf("unmarshal list: %v", err)
    }
    if len(listResp.Bundles) != 1 {
        t.Fatalf("expected 1 bundle, got %d", len(listResp.Bundles))
    }

    // Get bundle
    getReq := httptest.NewRequest(http.MethodGet, "/rulepacks/"+packID.String()+"/bundles/1", nil)
    getReq.Header.Set("X-PS-User-Admin", "true")
    getReq.Header.Set("X-PS-Tenant-ID", tenantID.String())
    getRec := httptest.NewRecorder()
    mux.ServeHTTP(getRec, getReq)
    if getRec.Code != http.StatusOK {
        t.Fatalf("get bundle: expected 200, got %d body=%s", getRec.Code, getRec.Body.String())
    }
    var bundle map[string]any
    if err := json.Unmarshal(getRec.Body.Bytes(), &bundle); err != nil { t.Fatalf("unmarshal bundle: %v", err) }

    // Verify-only endpoint with the retrieved bundle
    verifyBody := bytes.NewBuffer(getRec.Body.Bytes())
    verReq := httptest.NewRequest(http.MethodPost, "/rulepacks/"+packID.String()+"/bundles/verify", verifyBody)
    verReq.Header.Set("Content-Type", "application/json")
    verReq.Header.Set("X-PS-User-Admin", "true")
    verReq.Header.Set("X-PS-Tenant-ID", tenantID.String())
    verRec := httptest.NewRecorder()
    mux.ServeHTTP(verRec, verReq)
    if verRec.Code != http.StatusOK {
        t.Fatalf("verify bundle: expected 200, got %d body=%s", verRec.Code, verRec.Body.String())
    }
    var verResp struct { Valid bool `json:"valid"` }
    if err := json.Unmarshal(verRec.Body.Bytes(), &verResp); err != nil { t.Fatalf("unmarshal verify: %v", err) }
    if !verResp.Valid { t.Fatalf("expected valid verification") }

    // Activate stored bundle
    actReq := httptest.NewRequest(http.MethodPost, "/rulepacks/"+packID.String()+"/bundles/1/activate", nil)
    actReq.Header.Set("X-PS-User-Admin", "true")
    actReq.Header.Set("X-PS-Tenant-ID", tenantID.String())
    actRec := httptest.NewRecorder()
    mux.ServeHTTP(actRec, actReq)
    if actRec.Code != http.StatusOK {
        t.Fatalf("activate bundle: expected 200, got %d body=%s", actRec.Code, actRec.Body.String())
    }

    // Ensure a file was written in PS_BUNDLE_DIR
    if entries, err := os.ReadDir(filepath.Join(tmp, tenantID.String(), packID.String())); err != nil || len(entries) == 0 {
        t.Fatalf("expected stored bundle files; err=%v entries=%d", err, len(entries))
    }
}
