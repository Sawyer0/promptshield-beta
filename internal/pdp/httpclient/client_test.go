package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/pdp"
)

func TestHTTPClient_Evaluate_SetsAPIKeyAndParsesResponse(t *testing.T) {
	var gotAuth string
	var gotBody pdp.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"decision":  "PERMIT",
			"reason":    "ok",
			"provider":  "mock",
			"ttlMs":     1234,
			"cacheable": true,
		})
	}))
	defer ts.Close()
	c := New(Config{Endpoint: ts.URL, APIKey: "k", Timeout: time.Second})
resp, err := c.Evaluate(context.Background(), pdp.Request{Action: "x"})
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if gotAuth != "Bearer k" { t.Fatalf("expected auth header, got %q", gotAuth) }
	if resp.Decision != pdp.Permit || resp.Reason != "ok" || resp.Provider != "mock" || resp.TTL != 1234*time.Millisecond || !resp.Cacheable {
		t.Fatalf("unexpected resp: %#v", resp)
	}
}

func TestHTTPClient_Evaluate_500Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	c := New(Config{Endpoint: ts.URL, Timeout: time.Second})
	_, err := c.Evaluate(context.Background(), pdp.Request{})
	if err == nil { t.Fatalf("expected error on 500") }
}

func TestHTTPClient_Evaluate_InvalidDecision(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"decision": "MAYBE"})
	}))
	defer ts.Close()
	c := New(Config{Endpoint: ts.URL})
	_, err := c.Evaluate(context.Background(), pdp.Request{})
	if err == nil { t.Fatalf("expected error for invalid decision") }
}
