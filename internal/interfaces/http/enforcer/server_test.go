package enforcerhttp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestCheckDecisionHeadersAndBody(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()
	// Send a simple body with no violations expected
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", bytes.NewBufferString("hello world"))
	req.Header.Set("X-Request-ID", "test-id-1234")
	req.Header.Set("X-PS-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	// Headers
	if v := resp.Header.Get("x-ps-decision"); v == "" {
		t.Fatal("missing x-ps-decision header")
	}
	if v := resp.Header.Get("x-ps-reason"); v == "" {
		t.Fatal("missing x-ps-reason header")
	}
	if v := resp.Header.Get("x-ps-request-id"); v == "" {
		t.Fatal("missing x-ps-request-id header")
	}
	// Body JSON
	body, _ := io.ReadAll(resp.Body)
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if obj["decision"] == "" {
		t.Fatal("missing decision in body")
	}
}
