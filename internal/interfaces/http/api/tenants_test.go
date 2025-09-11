package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
)

// Test resolveExternalOrgHandler in dev bypass and normal mode
func TestResolveExternalOrgHandler_DevBypassAndNormal(t *testing.T) {
	// Require DB for repository-backed operations
	dsn := os.Getenv("PS_PG_DSN")
	if dsn == "" {
		dsn = os.Getenv("PS_TEST_PG_DSN")
	}
	if dsn == "" {
		t.Skip("PS_PG_DSN/PS_TEST_PG_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pg.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("db: %v", err)
	}

	tenantRepo := pg.TenantRepo(pool)

	// Build mux with repositories
	opt := Options{TenantRepository: tenantRepo, DB: pool}
	mux := NewMux(opt)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Ensure dev bypass does not interfere (handler isn't gated by dev flag)
	_ = os.Setenv("PS_DEV_BYPASS_AUTH", "true")

	body := map[string]any{"provider": "clerk", "external_org_id": "org_test_123", "fallback_name": "Org Test"}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/v1/tenants/resolve", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var data map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&data)
	if data["tenant_id"] == nil || data["tenant_id"] == "" {
		t.Fatalf("missing tenant_id in response: %+v", data)
	}

	// Second call is idempotent; expect same mapping
	resp2, err := http.Post(srv.URL+"/v1/tenants/resolve", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post2: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
}
