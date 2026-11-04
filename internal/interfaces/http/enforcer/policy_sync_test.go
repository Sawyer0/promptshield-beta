package enforcerhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type countingTransport struct{ n atomic.Int32 }

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	c.n.Add(1)
	return nil, http.ErrUseLastResponse
}

func TestPolicySync_DisabledWithoutURL(t *testing.T) {
	t.Setenv("PS_FRONTEND_URL", "")
	t.Setenv("PS_DISABLE_POLICY_SYNC", "")

	if StartPolicySync(context.Background()) {
		t.Fatal("policy sync started with no PS_FRONTEND_URL")
	}

	c := NewPolicySyncClient()
	if c.frontendURL != "" {
		t.Fatalf("frontendURL = %q, want empty (no default)", c.frontendURL)
	}
	rt := &countingTransport{}
	c.httpClient.Transport = rt
	if err := c.SyncActivePolicies(context.Background()); err != nil {
		t.Fatalf("SyncActivePolicies: %v", err)
	}
	if n := rt.n.Load(); n != 0 {
		t.Fatalf("made %d HTTP requests with sync disabled", n)
	}
}

func TestPolicySync_DisabledByFlag(t *testing.T) {
	t.Setenv("PS_FRONTEND_URL", "http://127.0.0.1:1")
	t.Setenv("PS_DISABLE_POLICY_SYNC", "1")
	if StartPolicySync(context.Background()) {
		t.Fatal("policy sync started with PS_DISABLE_POLICY_SYNC=1")
	}
}

func TestPolicySync_FetchesConfiguredURL(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"not-a-uuid","name":"p","is_active":false}]`))
	}))
	defer srv.Close()

	t.Setenv("PS_FRONTEND_URL", srv.URL+"/")
	t.Setenv("PS_FRONTEND_TOKEN", "tok")

	c := NewPolicySyncClient()
	if c.frontendURL != srv.URL {
		t.Fatalf("frontendURL = %q, want %q", c.frontendURL, srv.URL)
	}
	if err := c.SyncActivePolicies(context.Background()); err != nil {
		t.Fatalf("SyncActivePolicies: %v", err)
	}
	if gotPath != "/api/policies" {
		t.Errorf("path = %q, want /api/policies", gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
	}
}
