package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func token(t *testing.T, lic License, priv ed25519.PrivateKey) string {
	t.Helper()
	payload, err := json.Marshal(lic)
	if err != nil {
		t.Fatal(err)
	}
	sig := []byte("unsigned")
	if priv != nil {
		sig = ed25519.Sign(priv, payload)
	}
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func resetState(t *testing.T) {
	t.Helper()
	mu.Lock()
	licensed, licenseInfo, loadedKey = false, License{}, ""
	mu.Unlock()
}

func validLicense(features ...string) License {
	f := map[string]bool{}
	for _, name := range features {
		f[name] = true
	}
	return License{
		Organization: "Acme",
		Tier:         "enterprise",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Entitlements: Entitlements{MaxRPS: 10, Features: f},
	}
}

func TestValidate(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	expired := validLicense()
	expired.ExpiresAt = time.Now().Add(-time.Hour)
	badTier := validLicense()
	badTier.Tier = "free"
	noOrg := validLicense()
	noOrg.Organization = ""

	tests := []struct {
		name    string
		token   string
		pubKey  string
		wantErr bool
	}{
		{name: "valid unsigned without public key", token: token(t, validLicense(), nil)},
		{name: "valid signed with public key", token: token(t, validLicense(), priv), pubKey: base64.RawURLEncoding.EncodeToString(pub)},
		{name: "bad signature with public key", token: token(t, validLicense(), nil), pubKey: base64.RawURLEncoding.EncodeToString(pub), wantErr: true},
		{name: "expired", token: token(t, expired, nil), wantErr: true},
		{name: "invalid tier", token: token(t, badTier, nil), wantErr: true},
		{name: "missing org", token: token(t, noOrg, nil), wantErr: true},
		{name: "malformed", token: "not-a-token", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PROMPTSHIELD_LICENSE_PUBLIC_KEY", tc.pubKey)
			_, err := validate(tc.token)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestEnsureLoadedPicksUpRotatedKey(t *testing.T) {
	resetState(t)
	t.Cleanup(func() { resetState(t) })
	t.Setenv("PROMPTSHIELD_LICENSE_PUBLIC_KEY", "")

	t.Setenv("PROMPTSHIELD_LICENSE_KEY", token(t, validLicense("l3_semantic"), nil))
	if !IsLicensed() || HasFeature("async_jobs") {
		t.Fatalf("first key: licensed=%v async_jobs=%v", IsLicensed(), HasFeature("async_jobs"))
	}

	t.Setenv("PROMPTSHIELD_LICENSE_KEY", token(t, validLicense("async_jobs"), nil))
	if !HasFeature("async_jobs") {
		t.Fatal("rotated key with async_jobs was not picked up")
	}

	// An invalid replacement keeps the current license.
	t.Setenv("PROMPTSHIELD_LICENSE_KEY", "garbage")
	if !HasFeature("async_jobs") {
		t.Fatal("invalid key should not drop the current license")
	}
}

func TestUnlicensedHasNoFeatures(t *testing.T) {
	resetState(t)
	t.Cleanup(func() { resetState(t) })
	t.Setenv("PROMPTSHIELD_LICENSE_KEY", "")
	if IsLicensed() || HasFeature("async_jobs") {
		t.Fatal("expected unlicensed with no features")
	}
}

func TestTokenBucket(t *testing.T) {
	b := newTokenBucket(3, time.Hour)
	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if b.Allow() {
		t.Fatal("fourth request within the window should be limited")
	}
}
