package enforcerhttp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"io"

	"github.com/promptshield/promptshield/internal/application/services"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	api "github.com/promptshield/promptshield/internal/interfaces/http/api"
)

// mintRS256 creates a minimal RS256 JWT string for tests
func mintRS256(priv *rsa.PrivateKey, claims map[string]any) (string, error) {
	header := map[string]any{"alg": "RS256", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(claims)
	enc := func(b []byte) string { return strings.TrimRight(base64.RawURLEncoding.EncodeToString(b), "=") }
	h := enc(hb)
	p := enc(pb)
	signed := []byte(h + "." + p)
	sum := sha256.Sum256(signed)
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	s := enc(sig)
	return h + "." + p + "." + s, nil
}

func ensureSSLModeRequire(dsn string) string {
	if dsn == "" {
		return dsn
	}
	if strings.Contains(dsn, "sslmode=") {
		return dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "sslmode=require"
}

func TestTenantIsolation_WithJWT(t *testing.T) {
	// Generate RSA keypair
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	pub := priv.PublicKey
	pubDer, _ := x509.MarshalPKIXPublicKey(&pub)
	pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDer})
	os.Setenv("PS_BFF_JWT_PUBLIC_KEY", string(pubPem))
	os.Setenv("PS_DEV_BYPASS_AUTH", "false")
	os.Setenv("PS_ENFORCER_ADMIN_TOKEN", "admintest")

	// Build server with explicit DB
	dsn := os.Getenv("PS_PG_DSN")
	if dsn == "" {
		dsn = os.Getenv("PS_TEST_PG_DSN")
	}
	if dsn == "" {
		t.Skip("PS_PG_DSN/PS_TEST_PG_DSN not set")
	}
	dsn = ensureSSLModeRequire(dsn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pg.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("db pool: %v", err)
	}

	// For integration test, continue using direct repository creation
	// This tests the actual PostgreSQL integration
	rulepackRepo := pg.RulepackRepo(pool)
	rulepackSvc := services.RulepackServiceCstor(rulepackRepo, nil)
	tenantRepo := pg.TenantRepo(pool)
	assignRepo := pg.RulepackAssignmentRepo(pool)
	auditRepo := pg.AuditRepo(pool)
	manager := NewScannerManagerWithRulepackService(rulepackSvc, pool)
	opt := api.Options{
		AdminToken:           "admintest",
		RulepackService:      rulepackSvc,
		TenantRepository:     tenantRepo,
		AssignmentRepository: assignRepo,
		AuditRepository:      auditRepo,
		SettingsRepository:   pg.NewSettingsRepository(pool),
		DB:                   pool,
		ScannerManager:       manager,
	}
	srv := httptest.NewServer(NewMuxWithOptions(opt))
	defer srv.Close()

	// Create Tenants A and B via admin API
	createTenant := func(name string) string {
		reqBody := []byte(fmt.Sprintf(`{"name":"%s"}`, name))
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/admin/tenants/", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer admintest")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("create tenant %s: %v", name, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("create tenant status %d body: %s", resp.StatusCode, string(b))
		}
		var obj map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&obj)
		return fmt.Sprint(obj["id"])
	}
	tenantA := createTenant("Tenant A")
	tenantB := createTenant("Tenant B")

	// Upload + activate rulepack for Tenant A
	rulepackYAML := []byte("metadata:\n  name: test-rules\n\nrules:\n  - id: deny-keyword\n    level: 1\n    keywords: [forbidden]\n    severity: ERROR\n    response:\n      action: deny\n")
	upload := func(tenant string, activate bool) string {
		url := srv.URL + "/v1/rulepacks/?activate="
		if activate {
			url += "true"
		} else {
			url += "false"
		}
		req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(rulepackYAML))
		req.Header.Set("Authorization", "Bearer admintest")
		req.Header.Set("Content-Type", "application/x-yaml")
		req.Header.Set("X-PS-Tenant-ID", tenant)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("upload: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("upload status %d body: %s", resp.StatusCode, string(b))
		}
		var meta map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&meta)
		return fmt.Sprint(meta["id"])
	}
	_ = upload(tenantA, true)

	// Helper to mint JWT with tenant claim
	now := time.Now().Unix()
	mkJWT := func(user, tenant string) string {
		claims := map[string]any{"iss": "ps-bff", "sub": user, "aud": "ps-enforcer", "iat": now, "nbf": now - 1, "exp": now + 3600, "tenant_id": tenant, "name": user, "roles": []string{"user"}}
		tok, err := mintRS256(priv, claims)
		if err != nil {
			t.Fatalf("jwt: %v", err)
		}
		return tok
	}

	// Tenant A should be denied for matching content
	reqA, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", bytes.NewBufferString("this has forbidden token"))
	reqA.Header.Set("Authorization", "Bearer "+mkJWT("userA", tenantA))
	reqA.Header.Set("X-PS-Tenant-ID", tenantA)
	respA, err := http.DefaultClient.Do(reqA)
	if err != nil {
		t.Fatalf("check A: %v", err)
	}
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(respA.Body)
		t.Fatalf("tenant A expected 403, got %d body: %s", respA.StatusCode, string(b))
	}

	// Tenant B should be allowed (no active rulepack)
	reqB, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", bytes.NewBufferString("this has forbidden token"))
	reqB.Header.Set("Authorization", "Bearer "+mkJWT("userB", tenantB))
	reqB.Header.Set("X-PS-Tenant-ID", tenantB)
	respB, err := http.DefaultClient.Do(reqB)
	if err != nil {
		t.Fatalf("check B: %v", err)
	}
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(respB.Body)
		t.Fatalf("tenant B expected 200, got %d body: %s", respB.StatusCode, string(b))
	}
}
