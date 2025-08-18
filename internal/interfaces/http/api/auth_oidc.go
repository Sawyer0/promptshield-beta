package api

import (
    "context"
    "fmt"
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

// OIDCVerifier wraps OIDC token verification with thread-safe initialization
type OIDCVerifier struct {
    verifier *oidc.IDTokenVerifier
    once     sync.Once
    initErr  error
    config   OIDCConfig
}

// oidcVerifier creates a new OIDC verifier instance
func oidcVerifier(config OIDCConfig) *OIDCVerifier {
    return &OIDCVerifier{
        config: config,
    }
}

// init ensures the verifier is initialized exactly once per instance
func (v *OIDCVerifier) init(ctx context.Context) error {
    v.once.Do(func() {
        if v.config.Issuer == "" {
            return
        }
        
        provider, err := oidc.NewProvider(ctx, v.config.Issuer)
        if err != nil {
            v.initErr = fmt.Errorf("failed to create OIDC provider: %w", err)
            return
        }
        
        cfg := &oidc.Config{
            ClientID:          v.config.Audience,
            SkipClientIDCheck: v.config.Audience == "",
        }
        v.verifier = provider.Verifier(cfg)
    })
    return v.initErr
}

// Verify validates a JWT token and returns the claims
func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (map[string]any, error) {
    if err := v.init(ctx); err != nil {
        return nil, err
    }
    
    if v.verifier == nil {
        return nil, fmt.Errorf("OIDC not configured")
    }
    
    idToken, err := v.verifier.Verify(ctx, rawToken)
    if err != nil {
        return nil, fmt.Errorf("token verification failed: %w", err)
    }
    
    var claims map[string]any
    if err := idToken.Claims(&claims); err != nil {
        return nil, fmt.Errorf("failed to extract claims: %w", err)
    }
    
    return claims, nil
}

// oidcAuth validates Bearer JWTs against configured issuer/audience and attaches claims to context.
func oidcAuth(verifier *OIDCVerifier) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip OIDC if not configured
            if verifier == nil || verifier.config.Issuer == "" {
                next.ServeHTTP(w, r)
                return
            }
            
            authz := r.Header.Get("Authorization")
            if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
                writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token", nil)
                return
            }
            
            rawToken := strings.TrimSpace(authz[7:])
            claims, err := verifier.Verify(r.Context(), rawToken)
            if err != nil {
                writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token", map[string]any{"error": err.Error()})
                return
            }
            
            ctx := context.WithValue(r.Context(), oidcClaimsKey{}, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Context helpers
type oidcClaimsKey struct{}

func claimsFromCtx(ctx context.Context) map[string]any {
    if v := ctx.Value(oidcClaimsKey{}); v != nil {
        if m, ok := v.(map[string]any); ok { 
            return m 
        }
    }
    return nil
}

// hasScope checks if claims contain a specific scope
func hasScope(claims map[string]any, requiredScope string) bool {
    if claims == nil {
        return false
    }
    
    // Check 'scope' claim (space-separated, OAuth standard)
    if scopeStr, ok := claims["scope"].(string); ok {
        for _, scope := range strings.Fields(scopeStr) {
            if scope == requiredScope {
                return true
            }
        }
    }
    
    // Check 'scopes' claim (array)
    if scopesSlice, ok := claims["scopes"].([]interface{}); ok {
        for _, scope := range scopesSlice {
            if s, ok := scope.(string); ok && s == requiredScope {
                return true
            }
        }
    }
    
    return false
}

// requireScope creates middleware that requires a specific OAuth scope
func requireScope(requiredScope string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := claimsFromCtx(r.Context())
            if claims == nil {
                // No OIDC claims, skip scope check
                next.ServeHTTP(w, r)
                return
            }
            
            if !hasScope(claims, requiredScope) {
                writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient scope", map[string]any{
                    "required_scope": requiredScope,
                })
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}


