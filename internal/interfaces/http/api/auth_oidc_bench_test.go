package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
)

// setupFakeOIDC spins up a local OIDC discovery + JWKS endpoint and returns issuer URL and a signed JWT.
func setupFakeOIDC(tb testing.TB, audience string) (issuer string, token string, closeFn func()) {
	tb.Helper()

	// Generate RSA signing key and wrap into JWK
	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatalf("generate key: %v", err)
	}
	kid := uuid.NewString()
	jwk := jose.JSONWebKey{Key: rsaPriv, KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"}

	// Prepare HTTP server for discovery and JWKS
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	issuer = srv.URL

	// JWKS handler exposes the public key
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		pub := jwk.Public()
		jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{pub}}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	// Discovery document
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"jwks_uri":                              issuer + "/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	// Create a signed JWT with matching iss/aud
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", kid))
	if err != nil {
		tb.Fatalf("new signer: %v", err)
	}

	claims := map[string]any{
		"iss": issuer,
		"aud": audience,
		"sub": "test-subject",
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
		"exp": time.Now().Add(10 * time.Minute).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		tb.Fatalf("marshal claims: %v", err)
	}
	jws, err := signer.Sign(payload)
	if err != nil {
		tb.Fatalf("sign token: %v", err)
	}
	raw, err := jws.CompactSerialize()
	if err != nil {
		tb.Fatalf("serialize token: %v", err)
	}

	closeFn = func() { srv.Close() }
	return issuer, raw, closeFn
}

// Benchmark the middleware with OIDC disabled (pass-through baseline).
func BenchmarkOIDCAuth_Disabled(b *testing.B) {
	h := oidcAuth(Options{OIDC: OIDCConfig{}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/check", nil)
	req.Header.Set("Authorization", "Bearer ignored")
	rw := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(rw, req)
		_ = rw.Result().Body.Close()
	}
}

// Benchmark the middleware with OIDC enabled and a valid token.
func BenchmarkOIDCAuth_Enabled(b *testing.B) {
	audience := "bench-aud"
	issuer, token, closeFn := setupFakeOIDC(b, audience)
	defer closeFn()

	opt := Options{OIDC: OIDCConfig{Issuer: issuer, Audience: audience, CacheTTL: time.Minute}}

	// Initialize verifier once and warm up JWKS cache.
	if err := (&opt).initOIDCVerifier(context.Background()); err != nil {
		b.Fatalf("init verifier: %v", err)
	}
	v := opt.oidcVerifier.(*oidc.IDTokenVerifier)
	if _, err := v.Verify(context.Background(), token); err != nil {
		b.Fatalf("warmup verify: %v", err)
	}

	// Build middleware/handler chain
	h := oidcAuth(opt)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/check", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, req)
		_ = rw.Result().Body.Close()
		if rw.Result().StatusCode != http.StatusOK {
			b.Fatalf("unexpected status: %d", rw.Result().StatusCode)
		}
	}
}
