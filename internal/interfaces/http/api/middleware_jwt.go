package api

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// jwtAuthMiddleware validates RS256 JWTs issued by the BFF and injects
// trusted user and tenant context for downstream handlers.
//
// Configuration via environment variables:
// - PS_BFF_JWT_PUBLIC_KEY: PEM-encoded RSA public key (required to enable)
// - PS_BFF_JWT_ISSUER: expected issuer (optional)
// - PS_BFF_JWT_AUDIENCE: expected audience (optional)
// - PS_BFF_JWT_LEEWAY: clock skew allowance in seconds (default 60)
func jwtAuthMiddleware(next http.Handler) http.Handler {
	// Dev bypass mode
	devBypass := strings.EqualFold(strings.TrimSpace(os.Getenv("PS_DEV_BYPASS_AUTH")), "true")
	var devUserID = strings.TrimSpace(os.Getenv("PS_DEV_USER_ID"))
	if devUserID == "" {
		devUserID = "dev-user"
	}
	var devUserName = strings.TrimSpace(os.Getenv("PS_DEV_USER_NAME"))
	if devUserName == "" {
		devUserName = "Dev User"
	}
	var devTenantID = strings.TrimSpace(os.Getenv("PS_DEV_TENANT_ID"))
	var devRoles = strings.TrimSpace(os.Getenv("PS_DEV_ROLES")) // comma-separated
	devIsAdmin := strings.EqualFold(strings.TrimSpace(os.Getenv("PS_DEV_IS_ADMIN")), "true")

	// Load JWT validation config once (only when not bypassing)
	pubKeyPEM := strings.TrimSpace(os.Getenv("PS_BFF_JWT_PUBLIC_KEY"))
	issuer := strings.TrimSpace(os.Getenv("PS_BFF_JWT_ISSUER"))
	audience := strings.TrimSpace(os.Getenv("PS_BFF_JWT_AUDIENCE"))
	leewaySec := int64(60)
	if v := strings.TrimSpace(os.Getenv("PS_BFF_JWT_LEEWAY")); v != "" {
		if n, err := time.ParseDuration(v + "s"); err == nil {
			leewaySec = int64(n / time.Second)
		}
	}

	var verifyKey *rsa.PublicKey
	if !devBypass {
		if pubKeyPEM != "" {
			if k, err := parseRSAPublicKeyFromPEM([]byte(pubKeyPEM)); err != nil {
				slog.Error("Invalid PS_BFF_JWT_PUBLIC_KEY", "error", err)
			} else {
				verifyKey = k
				slog.Info("BFF JWT validation enabled")
			}
		} else {
			slog.Warn("PS_BFF_JWT_PUBLIC_KEY not set; JWT auth disabled (dev only)")
		}
	} else {
		slog.Warn("PS_DEV_BYPASS_AUTH enabled: injecting dev user and skipping JWT validation")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Dev bypass: inject stub identity and continue
		if devBypass {
			// Only set if not already present to allow overrides
			if r.Header.Get("X-PS-User-ID") == "" {
				r.Header.Set("X-PS-User-ID", devUserID)
			}
			if r.Header.Get("X-PS-User-Name") == "" {
				r.Header.Set("X-PS-User-Name", devUserName)
			}
			if devTenantID != "" && r.Header.Get("X-PS-Tenant-ID") == "" {
				r.Header.Set("X-PS-Tenant-ID", devTenantID)
			}
			if devRoles != "" {
				r.Header.Set("X-PS-User-Roles", devRoles)
			}
			if devIsAdmin {
				r.Header.Set("X-PS-User-Admin", "true")
			}
			next.ServeHTTP(w, r)
			return
		}

		// If not configured, no-op (dev behavior)
		if verifyKey == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Public endpoints bypass auth
		if isPublicEndpoint(r.URL.Path) || isJWTBypassPath(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Admin and debug endpoints are protected separately
		if strings.HasPrefix(r.URL.Path, "/admin/") || strings.HasPrefix(r.URL.Path, "/v1/admin/") ||
			strings.HasPrefix(r.URL.Path, "/debug/") || strings.HasPrefix(r.URL.Path, "/v1/debug/") {
			next.ServeHTTP(w, r)
			return
		}

		// Expect Authorization: Bearer <token>
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token", nil, r)
			return
		}
		tokenString := strings.TrimSpace(authz[7:])

		// Verify RS256 signature and parse claims
		header, payload, sig, err := splitJWT(tokenString)
		if err != nil {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		if strings.ToUpper(header.Alg) != "RS256" {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "unsupported alg", map[string]interface{}{"alg": header.Alg}, r)
			return
		}
		signed := header.raw + "." + payload.raw
		h := sha256.Sum256([]byte(signed))
		if err := rsa.VerifyPKCS1v15(verifyKey, crypto.SHA256, h[:], sig); err != nil {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "signature verification failed", nil, r)
			return
		}

		// Validate claims
		now := time.Now().Unix()
		if payload.Exp != 0 && now > payload.Exp+leewaySec {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "token expired", nil, r)
			return
		}
		if payload.Nbf != 0 && now+leewaySec < payload.Nbf {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "token not yet valid", nil, r)
			return
		}
		if issuer != "" && payload.Iss != issuer {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid issuer", map[string]interface{}{"expected": issuer, "got": payload.Iss}, r)
			return
		}
		if audience != "" && !audMatches(payload.Aud, audience) {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid audience", map[string]interface{}{"expected": audience, "got": payload.Aud}, r)
			return
		}

		// Inject trusted headers
		if payload.TenantID != "" {
			r.Header.Set("X-PS-Tenant-ID", payload.TenantID)
		}
		if payload.Sub != "" {
			r.Header.Set("X-PS-User-ID", payload.Sub)
		}
		if payload.Name != "" {
			r.Header.Set("X-PS-User-Name", payload.Name)
		} else if payload.Email != "" {
			r.Header.Set("X-PS-User-Name", payload.Email)
		}
		// Roles/admin flags
		if len(payload.Roles) > 0 {
			r.Header.Set("X-PS-User-Roles", strings.Join(payload.Roles, ","))
		}
		if payload.Admin {
			r.Header.Set("X-PS-User-Admin", "true")
		}

		// Continue
		next.ServeHTTP(w, r)
	})
}

// Helpers

func parseRSAPublicKeyFromPEM(b []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	switch block.Type {
	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if pk, ok := key.(*rsa.PublicKey); ok {
			return pk, nil
		}
		return nil, errors.New("not RSA public key")
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return nil, errors.New("unsupported key type: " + block.Type)
	}
}

func isJWTBypassPath(method, path string) bool {
	// Additional paths that should bypass JWT (e.g., static business demo)
	switch path {
	case "/version":
		return true
	}
	return false
}

// Minimal JWT parsing/verification helpers
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	raw string
}

type jwtPayload struct {
	Iss      string      `json:"iss"`
	Sub      string      `json:"sub"`
	Aud      interface{} `json:"aud"`
	Exp      int64       `json:"exp"`
	Nbf      int64       `json:"nbf"`
	Iat      int64       `json:"iat"`
	TenantID string      `json:"tenant_id"`
	Name     string      `json:"name"`
	Email    string      `json:"email"`
	Roles    []string    `json:"roles"`
	Admin    bool        `json:"admin"`
	raw      string
}

func splitJWT(token string) (jwtHeader, jwtPayload, []byte, error) {
	var h jwtHeader
	var p jwtPayload
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return h, p, nil, errors.New("invalid token format")
	}
	hb, err := b64url(parts[0])
	if err != nil {
		return h, p, nil, err
	}
	pb, err := b64url(parts[1])
	if err != nil {
		return h, p, nil, err
	}
	sb, err := b64url(parts[2])
	if err != nil {
		return h, p, nil, err
	}
	if err := json.Unmarshal(hb, &h); err != nil {
		return h, p, nil, err
	}
	if err := json.Unmarshal(pb, &p); err != nil {
		return h, p, nil, err
	}
	h.raw = parts[0]
	p.raw = parts[1]
	return h, p, sb, nil
}

func b64url(s string) ([]byte, error) {
	// Base64 URL without padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.RawURLEncoding.DecodeString(s)
}

func audMatches(aud interface{}, target string) bool {
	switch v := aud.(type) {
	case string:
		return v == target
	case []interface{}:
		for _, e := range v {
			if s, ok := e.(string); ok && s == target {
				return true
			}
		}
	default:
		// unknown type – treat as mismatch
	}
	return false
}
