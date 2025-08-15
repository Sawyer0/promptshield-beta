package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
)

// BenchmarkGatewayHTTPCheck64KB measures end-to-end HTTP /check throughput for a small body.
func BenchmarkGatewayHTTPCheck64KB(b *testing.B) {
	// Bypass evaluation rate-limit by setting a dummy license (no MaxRPS limiter).
	payload := `{"org":"Bench","tier":"enterprise","expires_at":"2099-01-01T00:00:00Z","entitlements":{}}`
	enc := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))
	b.Setenv("PROMPTSHIELD_LICENSE_KEY", enc)

	h := enforcerhttp.NewMux()
	ts := httptest.NewServer(h)
	defer ts.Close()

	body := bytes.Repeat([]byte("x"), 64*1024)
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := http.Post(ts.URL+"/check", "text/plain", bytes.NewReader(body))
		if err != nil {
			b.Fatalf("POST /check error: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusForbidden {
			b.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	}
}
