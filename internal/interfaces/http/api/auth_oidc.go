package api

import (
    "context"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/coreos/go-oidc/v3/oidc"
)

// OIDCConfig holds optional OIDC settings. When Issuer is empty, OIDC is disabled.
type OIDCConfig struct {
    Issuer       string
    Audience     string
    CacheTTL     time.Duration
}

var (
    verifierOnce sync.Once
)

func (opt *Options) initOIDCVerifier(ctx context.Context) error {
    if opt == nil || opt.OIDC.Issuer == "" {
        return nil
    }
    var initErr error
    verifierOnce.Do(func() {
        provider, err := oidc.NewProvider(ctx, opt.OIDC.Issuer)
        if err != nil {
            initErr = err
            return
        }
        cfg := &oidc.Config{ClientID: opt.OIDC.Audience, SkipClientIDCheck: opt.OIDC.Audience == ""}
        opt.oidcVerifier = provider.Verifier(cfg)
    })
    return initErr
}

// oidcAuth validates Bearer JWTs against configured issuer/audience and attaches claims to context.
func oidcAuth(opt Options) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if opt.OIDC.Issuer == "" {
                next.ServeHTTP(w, r)
                return
            }
            if err := (&opt).initOIDCVerifier(r.Context()); err != nil || opt.oidcVerifier == nil {
                writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "oidc verifier init failed", map[string]any{"error": err})
                return
            }
            authz := r.Header.Get("Authorization")
            if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
                writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token", nil)
                return
            }
            raw := strings.TrimSpace(authz[7:])
            v := opt.oidcVerifier.(*oidc.IDTokenVerifier)
            idt, err := v.Verify(r.Context(), raw)
            if err != nil {
                writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token", map[string]any{"error": err.Error()})
                return
            }
            var claims map[string]any
            _ = idt.Claims(&claims)
            ctx := context.WithValue(r.Context(), oidcClaimsKey{}, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Context helpers
type oidcClaimsKey struct{}

func claimsFromCtx(ctx context.Context) map[string]any {
    if v := ctx.Value(oidcClaimsKey{}); v != nil {
        if m, ok := v.(map[string]any); ok { return m }
    }
    return nil
}


