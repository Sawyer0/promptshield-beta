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
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// JWT error codes for structured error responses
const (
	JWTErrorMissing           = "JWT_MISSING"
	JWTErrorInvalid           = "JWT_INVALID"
	JWTErrorExpired           = "JWT_EXPIRED"
	JWTErrorNotYetValid       = "JWT_NOT_YET_VALID"
	JWTErrorInvalidIssuer     = "JWT_INVALID_ISSUER"
	JWTErrorInvalidAudience   = "JWT_INVALID_AUDIENCE"
	JWTErrorUnsupportedAlg    = "JWT_UNSUPPORTED_ALG"
	JWTErrorSignatureInvalid  = "JWT_SIGNATURE_INVALID"
	JWTErrorConfigurationError = "JWT_CONFIGURATION_ERROR"
)

// JWTValidationError represents a structured JWT validation error
type JWTValidationError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e JWTValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

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
	var configError error
	
	if !devBypass {
		if pubKeyPEM != "" {
			if k, err := parseRSAPublicKeyFromPEM([]byte(pubKeyPEM)); err != nil {
				configError = fmt.Errorf("invalid PS_BFF_JWT_PUBLIC_KEY: %w", err)
				slog.Error("JWT configuration error", "error", configError)
			} else {
				verifyKey = k
				slog.Info("BFF JWT validation enabled", 
					"issuer", issuer, 
					"audience", audience, 
					"leeway_seconds", leewaySec,
					"key_size_bits", k.Size()*8)
			}
		} else {
			slog.Warn("PS_BFF_JWT_PUBLIC_KEY not set; JWT auth disabled (dev only)")
		}
	} else {
		slog.Warn("PS_DEV_BYPASS_AUTH enabled: injecting dev user and skipping JWT validation",
			"dev_user_id", devUserID,
			"dev_tenant_id", devTenantID)
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

		// If configuration error, return error
		if configError != nil {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorConfigurationError,
				Message: "JWT authentication configuration error",
				Details: map[string]interface{}{
					"error": configError.Error(),
				},
			})
			return
		}

		// If not configured, no-op (dev behavior)
		if verifyKey == nil {
			slog.Debug("JWT validation skipped - no public key configured", "path", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		// Public endpoints bypass auth
    if isPublicEndpoint(r.URL.Path) || isJWTBypassPath(r.URL.Path) {
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
		if authz == "" {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorMissing,
				Message: "Authorization header is required",
				Details: map[string]interface{}{
					"expected_format": "Authorization: Bearer <token>",
				},
			})
			return
		}
		
		if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorMissing,
				Message: "Bearer token is required",
				Details: map[string]interface{}{
					"provided_format": strings.Split(authz, " ")[0],
					"expected_format": "Bearer <token>",
				},
			})
			return
		}
		
		tokenString := strings.TrimSpace(authz[7:])
		if tokenString == "" {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorMissing,
				Message: "JWT token is empty",
			})
			return
		}

		// Verify RS256 signature and parse claims
		header, payload, sig, err := splitJWT(tokenString)
		if err != nil {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorInvalid,
				Message: "Invalid JWT token format",
				Details: map[string]interface{}{
					"parse_error": err.Error(),
				},
			})
			return
		}
		
		if strings.ToUpper(header.Alg) != "RS256" {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorUnsupportedAlg,
				Message: "Unsupported JWT algorithm",
				Details: map[string]interface{}{
					"provided_algorithm": header.Alg,
					"expected_algorithm": "RS256",
				},
			})
			return
		}
		
		// Verify signature
		signed := header.raw + "." + payload.raw
		h := sha256.Sum256([]byte(signed))
		if err := rsa.VerifyPKCS1v15(verifyKey, crypto.SHA256, h[:], sig); err != nil {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorSignatureInvalid,
				Message: "JWT signature verification failed",
				Details: map[string]interface{}{
					"verification_error": err.Error(),
				},
			})
			return
		}

		// Validate claims
		now := time.Now().Unix()
		if payload.Exp != 0 && now > payload.Exp+leewaySec {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorExpired,
				Message: "JWT token has expired",
				Details: map[string]interface{}{
					"expired_at":    time.Unix(payload.Exp, 0).UTC().Format(time.RFC3339),
					"current_time":  time.Unix(now, 0).UTC().Format(time.RFC3339),
					"leeway_seconds": leewaySec,
				},
			})
			return
		}
		
		if payload.Nbf != 0 && now+leewaySec < payload.Nbf {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorNotYetValid,
				Message: "JWT token is not yet valid",
				Details: map[string]interface{}{
					"valid_from":     time.Unix(payload.Nbf, 0).UTC().Format(time.RFC3339),
					"current_time":   time.Unix(now, 0).UTC().Format(time.RFC3339),
					"leeway_seconds": leewaySec,
				},
			})
			return
		}
		
		if issuer != "" && payload.Iss != issuer {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorInvalidIssuer,
				Message: "JWT issuer validation failed",
				Details: map[string]interface{}{
					"expected_issuer": issuer,
					"provided_issuer": payload.Iss,
				},
			})
			return
		}
		
		if audience != "" && !audMatches(payload.Aud, audience) {
			writeJWTError(w, r, JWTValidationError{
				Code:    JWTErrorInvalidAudience,
				Message: "JWT audience validation failed",
				Details: map[string]interface{}{
					"expected_audience": audience,
					"provided_audience": payload.Aud,
				},
			})
			return
		}

		// Inject trusted headers from JWT claims
		correlationID := getCorrelationID(r)
		
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
		
		// Add correlation ID to request headers for downstream services
		r.Header.Set("X-Correlation-ID", correlationID)

		// Log successful JWT validation
		slog.Debug("JWT validation successful",
			"correlation_id", correlationID,
			"user_id", payload.Sub,
			"tenant_id", payload.TenantID,
			"issuer", payload.Iss,
			"expires_at", time.Unix(payload.Exp, 0).UTC().Format(time.RFC3339),
			"roles", payload.Roles,
			"is_admin", payload.Admin,
			"path", r.URL.Path,
		)

		// Continue to next middleware
		next.ServeHTTP(w, r)
	})
}

// Helpers

func parseRSAPublicKeyFromPEM(b []byte) (*rsa.PublicKey, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("empty PEM data")
	}

	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("no valid PEM block found in public key data (length: %d)", len(b))
	}

	switch block.Type {
	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}
		if pk, ok := key.(*rsa.PublicKey); ok {
			// Validate key size
			if pk.Size() < 256 { // 2048 bits minimum
				return nil, fmt.Errorf("RSA key too small: %d bits (minimum 2048)", pk.Size()*8)
			}
			return pk, nil
		}
		return nil, fmt.Errorf("parsed key is not an RSA public key, got %T", key)
	case "RSA PUBLIC KEY":
		pk, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS1 RSA public key: %w", err)
		}
		if pk.Size() < 256 { // 2048 bits minimum
			return nil, fmt.Errorf("RSA key too small: %d bits (minimum 2048)", pk.Size()*8)
		}
		return pk, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s (expected 'PUBLIC KEY' or 'RSA PUBLIC KEY')", block.Type)
	}
}

// writeJWTError writes a structured JWT error response
func writeJWTError(w http.ResponseWriter, r *http.Request, jwtErr JWTValidationError) {
	correlationID := getCorrelationID(r)
	
	// Add correlation ID to error details
	if jwtErr.Details == nil {
		jwtErr.Details = make(map[string]interface{})
	}
	jwtErr.Details["correlation_id"] = correlationID
	jwtErr.Details["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	jwtErr.Details["path"] = r.URL.Path
	jwtErr.Details["method"] = r.Method

	// Log the error with context
	slog.Error("JWT validation failed",
		"error_code", jwtErr.Code,
		"message", jwtErr.Message,
		"correlation_id", correlationID,
		"path", r.URL.Path,
		"method", r.Method,
		"user_agent", r.Header.Get("User-Agent"),
		"details", jwtErr.Details,
	)

	writeErrorJSON(w, http.StatusUnauthorized, jwtErr.Code, jwtErr.Message, jwtErr.Details, r)
}

// getCorrelationID extracts or generates a correlation ID for request tracing
func getCorrelationID(r *http.Request) string {
	// Try various common correlation ID headers
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = r.Header.Get("X-Request-ID")
	}
	if correlationID == "" {
		correlationID = r.Header.Get("X-Trace-ID")
	}
	if correlationID == "" {
		// Generate a simple correlation ID if none exists
		correlationID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return correlationID
}

func isJWTBypassPath(path string) bool {
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
