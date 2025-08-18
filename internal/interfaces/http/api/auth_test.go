package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// User endpoints: /check and /scan are protected by userAuth when PS_ENFORCER_AUTH_TOKEN is set.
func TestUserAuth_Behavior(t *testing.T) {
	// When token is unset, endpoints allow unauthenticated access
	t.Run("allow when token unset", func(t *testing.T) {
		// Ensure env var is unset for this subtest
		t.Setenv("PS_ENFORCER_AUTH_TOKEN", "")
		srv := httptest.NewServer(testRouterWithOptions(Options{AllowInsecureAdmin: true}))
		defer srv.Close()
		resp, err := http.Post(srv.URL+"/check", "text/plain", bytes.NewBufferString("hello"))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", resp.StatusCode)
		}
	})

	// When token is set, missing/invalid auth should 401
	t.Run("reject without token when set", func(t *testing.T) {
		t.Setenv("PS_ENFORCER_AUTH_TOKEN", "secret")
		srv := httptest.NewServer(testRouterWithOptions(Options{AllowInsecureAdmin: true}))
		defer srv.Close()
		resp, err := http.Post(srv.URL+"/check", "text/plain", bytes.NewBufferString("hello"))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", resp.StatusCode)
		}
	})

	t.Run("accept bearer and custom header", func(t *testing.T) {
		t.Setenv("PS_ENFORCER_AUTH_TOKEN", "secret")
		srv := httptest.NewServer(testRouterWithOptions(Options{AllowInsecureAdmin: true}))
		defer srv.Close()

		// Bearer header
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/check", bytes.NewBufferString("hello"))
		req.Header.Set("Authorization", "Bearer secret")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("bearer: want 200, got %d", res.StatusCode)
		}

		// Custom header
		req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/check", bytes.NewBufferString("hello"))
		req2.Header.Set("X-PS-Token", "secret")
		res2, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Fatal(err)
		}
		if res2.StatusCode != http.StatusOK {
			t.Fatalf("custom header: want 200, got %d", res2.StatusCode)
		}
	})
}

// Billing/admin endpoints: /usage and POST /license are protected by adminAuth.
func TestAdminAuth_Behavior(t *testing.T) {
	// GET /license is public
	t.Run("license get is public", func(t *testing.T) {
		srv := httptest.NewServer(testRouterWithOptions(Options{}))
		defer srv.Close()
		resp, err := http.Get(srv.URL + "/license")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", resp.StatusCode)
		}
	})

	// GET /usage requires admin token
	t.Run("usage requires admin token", func(t *testing.T) {
		srv := httptest.NewServer(testRouterWithOptions(Options{}))
		defer srv.Close()
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/usage", nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", res.StatusCode)
		}
	})

	// Authorized via Bearer and custom admin header
	t.Run("usage accepts bearer and custom header", func(t *testing.T) {
		srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: "x"}))
		defer srv.Close()

		// Bearer
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/usage", nil)
		req.Header.Set("Authorization", "Bearer x")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("bearer: want 200, got %d", res.StatusCode)
		}

		// Custom header
		req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/usage", nil)
		req2.Header.Set("X-PS-Admin-Token", "x")
		res2, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Fatal(err)
		}
		if res2.StatusCode != http.StatusOK {
			t.Fatalf("custom header: want 200, got %d", res2.StatusCode)
		}
	})

	// POST /license rotates license; must be admin-protected
	t.Run("license post requires admin and succeeds with bearer", func(t *testing.T) {
		// Unauthenticated should fail
		{
			srv := httptest.NewServer(testRouterWithOptions(Options{}))
			defer srv.Close()
			body, _ := json.Marshal(map[string]string{"key": "test-key"})
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/license", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("want 401, got %d", res.StatusCode)
			}
		}

		// With admin token should succeed
		{
			srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: "x"}))
			defer srv.Close()
			body, _ := json.Marshal(map[string]string{"key": "test-key"})
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/license", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer x")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusNoContent {
				t.Fatalf("want 204, got %d", res.StatusCode)
			}
		}
	})
}
