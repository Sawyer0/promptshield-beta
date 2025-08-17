package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/promptshield/promptshield/internal/usage"
)

func httpAuthOK(r *http.Request, want string) bool {
	if want == "" {
		return true
	}
	if v := r.Header.Get("Authorization"); v != "" {
		if len(v) >= 7 && (strings.HasPrefix(v, "Bearer ") || strings.HasPrefix(v, "bearer ")) {
			if v[7:] == want {
				return true
			}
		}
		if v == want {
			return true
		}
	}
	if v := r.Header.Get("X-PS-Token"); v != "" {
		if v == want {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func parseInt64Default(s string, def int64) int64 {
	if s == "" {
		return def
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return def
}

func parseFloatDefault(s string, def float64) float64 {
	if s == "" {
		return def
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n
	}
	return def
}

// Context key for usage store
type usageStoreKey struct{}

func withUsageStore(ctx context.Context, s usage.UsageStore) context.Context {
	return context.WithValue(ctx, usageStoreKey{}, s)
}

func getUsageStoreFromCtx(ctx context.Context) usage.UsageStore {
	if v := ctx.Value(usageStoreKey{}); v != nil {
		if s, ok := v.(usage.UsageStore); ok {
			return s
		}
	}
	return nil
}

// intFromAny attempts to coerce an any to int with safe defaults.
func intFromAny(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float64:
		return int(t)
	case float32:
		return int(t)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i)
		}
		if f, err := t.Float64(); err == nil {
			return int(f)
		}
	}
	return 0
}

// tenantFromRequest resolves a tenant id using (in order):
// 1) OIDC claims (tid | tenant | org | azp)
// 2) Header x-tenant-id
func tenantFromRequest(r *http.Request) string {
	if m := claimsFromCtx(r.Context()); m != nil {
		try := func(keys ...string) string {
			for _, k := range keys {
				if v, ok := m[k]; ok {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						return s
					}
				}
			}
			return ""
		}
		if s := try("tid", "tenant", "org", "azp"); s != "" {
			return s
		}
	}
	if v := strings.TrimSpace(r.Header.Get("x-tenant-id")); v != "" {
		return v
	}
	return ""
}
