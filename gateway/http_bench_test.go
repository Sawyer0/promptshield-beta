package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/repository"
)

// BenchmarkGatewayHTTPCheck64KB measures end-to-end HTTP /check throughput for a small body.
func BenchmarkGatewayHTTPCheck64KB(b *testing.B) {
	// Bypass evaluation rate-limit by setting a dummy license (no MaxRPS limiter).
	payload := `{"org":"Bench","tier":"enterprise","expires_at":"2099-01-01T00:00:00Z","entitlements":{}}`
	enc := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))
	b.Setenv("PROMPTSHIELD_LICENSE_KEY", enc)

	// Set test mode to ensure we get a test factory
	b.Setenv("PS_TEST_MODE", "true")

	// Setup with repository factory
	ctx := context.Background()
	repoFactory, err := repository.BuildWithFallback(ctx)
	if err != nil {
		b.Fatalf("Failed to create repository factory: %v", err)
	}
	defer repoFactory.Close()

	// Create NATS publisher
	publisher, err := nats.NewPublisher("")
	if err != nil {
		b.Fatalf("Failed to create NATS publisher: %v", err)
	}
	defer publisher.Close()

	// Create RulepackService using factory
	rulepackService := services.RulepackServiceFromFactory(repoFactory, publisher)
	
	options := api.Options{
		RulepackService: rulepackService,
	}
	h := enforcerhttp.NewMuxWithOptions(options)
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
