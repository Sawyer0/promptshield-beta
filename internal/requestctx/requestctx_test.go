package requestctx

import (
	"context"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-1")
	ctx = WithTenantID(ctx, "tenant-1")
	ctx = WithUserID(ctx, "user-1")
	ctx = WithCorrelationID(ctx, "corr-1")

	for name, get := range map[string]func(context.Context) (string, bool){
		"req-1":    GetRequestID,
		"tenant-1": GetTenantID,
		"user-1":   GetUserID,
		"corr-1":   GetCorrelationID,
	} {
		if got, ok := get(ctx); !ok || got != name {
			t.Errorf("got (%q, %v), want (%q, true)", got, ok, name)
		}
	}
}

func TestMissingValues(t *testing.T) {
	if _, ok := GetUserID(context.Background()); ok {
		t.Fatal("expected no user ID on an empty context")
	}
	// A plain string key must not collide with the package's private keys.
	ctx := context.WithValue(context.Background(), "user_id", "spoofed") //nolint:staticcheck // deliberate collision check
	if _, ok := GetUserID(ctx); ok {
		t.Fatal("string key should not be visible through GetUserID")
	}
}
