package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/usage"
)

type errPayload struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

func TestOIDC_VerifierInitFailure_JSONShape(t *testing.T) {
	// Enable OIDC with an invalid issuer to force init failure.
	srv := httptest.NewServer(NewMux(Options{OIDC: OIDCConfig{Issuer: "http://invalid-issuer.local"}}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/check", bytes.NewBufferString("hello"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
	var e errPayload
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if e.Code != "UNAUTHORIZED" {
		t.Fatalf("code=%q", e.Code)
	}
	if e.Message != "oidc verifier init failed" {
		t.Fatalf("message=%q", e.Message)
	}
}

func TestUserAuth_JSONErrorShape(t *testing.T) {
	t.Setenv("PS_ENFORCER_AUTH_TOKEN", "secret")
	// Licensed to avoid evaluation-mode 429s interfering with auth
	cleanup := withAsyncJobsLicense(t)
	defer cleanup()

	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true}))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/check", "text/plain", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
	var e errPayload
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if e.Code != "UNAUTHORIZED" {
		t.Fatalf("code=%q", e.Code)
	}
	if e.Message != "authentication required" {
		t.Fatalf("message=%q", e.Message)
	}
}

func TestAdminAuth_JSONErrorShape(t *testing.T) {
	srv := httptest.NewServer(NewMux(Options{AdminToken: "x"}))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/usage", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
	var e errPayload
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if e.Code != "UNAUTHORIZED" {
		t.Fatalf("code=%q", e.Code)
	}
	if e.Message != "invalid admin token" {
		t.Fatalf("message=%q", e.Message)
	}
}

func TestTenantQuota_RateLimit_JSONShape(t *testing.T) {
	t.Setenv("PS_ENFORCER_AUTH_TOKEN", "")
	cleanup := withAsyncJobsLicense(t)
	defer cleanup()

	// Use higher RPS and a short wait to allow token refill because the store drains on init.
	quota := usage.NewInMemoryQuota(100, 1)
	srv := httptest.NewServer(NewMux(Options{QuotaStore: quota, AllowInsecureAdmin: true}))
	defer srv.Close()

	// allow tokens to refill after construction
	time.Sleep(20 * time.Millisecond)

	// Prime limiter: first request will 429 due to store's warm-drain behavior on creation
	primeReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/check", bytes.NewBufferString("hello"))
	primeReq.Header.Set("x-tenant-id", "acme")
	primeRes, err := http.DefaultClient.Do(primeReq)
	if err != nil {
		t.Fatal(err)
	}
	if primeRes.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("prime want 429, got %d", primeRes.StatusCode)
	}

	// Wait for tokens to refill
	time.Sleep(15 * time.Millisecond)

	// Next request should pass
	reqOK, _ := http.NewRequest(http.MethodPost, srv.URL+"/check", bytes.NewBufferString("hello"))
	reqOK.Header.Set("x-tenant-id", "acme")
	resOK, err := http.DefaultClient.Do(reqOK)
	if err != nil {
		t.Fatal(err)
	}
	if resOK.StatusCode != http.StatusOK {
		t.Fatalf("ok want 200, got %d", resOK.StatusCode)
	}

	// Immediate subsequent request should be rate-limited again with proper JSON shape
	reqLimit, _ := http.NewRequest(http.MethodPost, srv.URL+"/check", bytes.NewBufferString("hello"))
	reqLimit.Header.Set("x-tenant-id", "acme")
	resLimit, err := http.DefaultClient.Do(reqLimit)
	if err != nil {
		t.Fatal(err)
	}
	if resLimit.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("limit want 429, got %d", resLimit.StatusCode)
	}
	var e errPayload
	if err := json.NewDecoder(resLimit.Body).Decode(&e); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if e.Code != "RESOURCE_EXHAUSTED" {
		t.Fatalf("code=%q", e.Code)
	}
	if e.Message != "rate limit exceeded" {
		t.Fatalf("message=%q", e.Message)
	}
	if e.Details == nil || e.Details["tenant"] != "acme" {
		t.Fatalf("missing tenant in details: %+v", e.Details)
	}
}
