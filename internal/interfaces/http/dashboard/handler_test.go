package dashboard

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":       {Data: []byte("<!doctype html><title>PromptShield</title>")},
		"assets/app-1.js":  {Data: []byte("console.log(1)")},
		"assets/app-1.css": {Data: []byte("body{}")},
	}
}

func get(t *testing.T, h http.Handler, method, target string) *http.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec.Result()
}

func TestServesIndexWithSecurityHeaders(t *testing.T) {
	resp := get(t, NewHandler(testFS()), http.MethodGet, "/")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "PromptShield") {
		t.Fatalf("index: %d %q", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("missing CSP, got %q", got)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("unexpected headers: %v", resp.Header)
	}
}

func TestServesHashedAssetsWithLongCache(t *testing.T) {
	resp := get(t, NewHandler(testFS()), http.MethodGet, "/assets/app-1.js")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "console.log(1)" {
		t.Fatalf("asset: %d %q", resp.StatusCode, body)
	}
	if !strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("want immutable cache, got %q", resp.Header.Get("Cache-Control"))
	}
}

func TestUnknownPathsFallBackToIndex(t *testing.T) {
	for _, target := range []string{"/decisions/abc", "/rules", "/../../etc/passwd", "/assets/"} {
		resp := get(t, NewHandler(testFS()), http.MethodGet, target)
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "PromptShield") {
			t.Errorf("%s: want index, got %d %q", target, resp.StatusCode, body)
		}
	}
}

func TestRejectsWrites(t *testing.T) {
	resp := get(t, NewHandler(testFS()), http.MethodPost, "/")
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", resp.StatusCode)
	}
}

func TestMissingBuildAnswers503(t *testing.T) {
	resp := get(t, NewHandler(fstest.MapFS{".gitkeep": {}}), http.MethodGet, "/")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), "make dashboard") {
		t.Fatalf("want 503 with hint, got %d %q", resp.StatusCode, body)
	}
}

func TestEmbeddedHandlerBuilds(t *testing.T) {
	resp := get(t, Handler(), http.MethodGet, "/")
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("embedded handler: unexpected status %d", resp.StatusCode)
	}
}
