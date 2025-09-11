package api

import (
    "crypto"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
    "encoding/json"
    "encoding/pem"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// generateTestKeyPair creates a temporary RSA keypair and returns (privPEM, pubPEM)
func generateTestKeyPair() ([]byte, []byte, error) {
    key, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return nil, nil, err
    }
    // Private key PKCS8
    privBytes, err := x509.MarshalPKCS8PrivateKey(key)
    if err != nil {
        return nil, nil, err
    }
    privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

    // Public key PKIX
    pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
    if err != nil {
        return nil, nil, err
    }
    pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
    return privPEM, pubPEM, nil
}

// Test keys are generated dynamically at runtime to avoid storing any PEM material in the repo.

func TestJWTAuthIntegration(t *testing.T) {
	// Generate an ephemeral RSA keypair for all subtests
	privPEM, pubPEM, err := generateTestKeyPair()
	require.NoError(t, err)
	// Set up test environment
	originalEnv := map[string]string{
		"PS_BFF_JWT_PUBLIC_KEY": os.Getenv("PS_BFF_JWT_PUBLIC_KEY"),
		"PS_BFF_JWT_ISSUER":     os.Getenv("PS_BFF_JWT_ISSUER"),
		"PS_BFF_JWT_AUDIENCE":   os.Getenv("PS_BFF_JWT_AUDIENCE"),
		"PS_DEV_BYPASS_AUTH":    os.Getenv("PS_DEV_BYPASS_AUTH"),
	}
	
// Set test environment
// Use the generated public key for all tests
os.Setenv("PS_BFF_JWT_PUBLIC_KEY", string(pubPEM))
	os.Setenv("PS_BFF_JWT_ISSUER", "test-issuer")
	os.Setenv("PS_BFF_JWT_AUDIENCE", "test-audience")
	os.Setenv("PS_DEV_BYPASS_AUTH", "false")
	
	defer func() {
		// Restore original environment
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("JWT Middleware Validation", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(jwtAuthMiddleware)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user_id":   r.Header.Get("X-PS-User-ID"),
				"tenant_id": r.Header.Get("X-PS-Tenant-ID"),
				"roles":     r.Header.Get("X-PS-User-Roles"),
				"is_admin":  r.Header.Get("X-PS-User-Admin"),
			})
		})

		t.Run("should reject requests without JWT", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			errorData := response["error"].(map[string]interface{})
			assert.Equal(t, "JWT_MISSING", errorData["code"])
		})

		t.Run("should reject requests with invalid JWT", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer invalid.jwt.token")
			w := httptest.NewRecorder()
			
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			errorData := response["error"].(map[string]interface{})
			assert.Equal(t, "JWT_INVALID", errorData["code"])
		})

		t.Run("should accept valid JWT and inject headers", func(t *testing.T) {
			// Use the already-generated keypair
			os.Setenv("PS_BFF_JWT_PUBLIC_KEY", string(pubPEM))
			// Build router with middleware bound to the current public key
			r2 := chi.NewRouter()
			r2.Use(jwtAuthMiddleware)
			r2.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"user_id":   r.Header.Get("X-PS-User-ID"),
					"tenant_id": r.Header.Get("X-PS-Tenant-ID"),
					"roles":     r.Header.Get("X-PS-User-Roles"),
					"is_admin":  r.Header.Get("X-PS-User-Admin"),
				})
			})
			// Sign a valid token with the generated private key
			validToken := generateTestJWT(t, privPEM)
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			w := httptest.NewRecorder()
			
			r2.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, "test-user-123", response["user_id"])
			assert.Equal(t, "test-tenant-123", response["tenant_id"])
			assert.Equal(t, "admin", response["roles"])
			assert.Equal(t, "true", response["is_admin"])
		})
	})

	t.Run("Dev Bypass Mode", func(t *testing.T) {
		// Enable dev bypass
		os.Setenv("PS_DEV_BYPASS_AUTH", "true")
		os.Setenv("PS_DEV_USER_ID", "dev-user-123")
		os.Setenv("PS_DEV_TENANT_ID", "dev-tenant-123")
		
		defer func() {
			os.Setenv("PS_DEV_BYPASS_AUTH", "false")
			os.Unsetenv("PS_DEV_USER_ID")
			os.Unsetenv("PS_DEV_TENANT_ID")
		}()

		r := chi.NewRouter()
		r.Use(jwtAuthMiddleware)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user_id":   r.Header.Get("X-PS-User-ID"),
				"tenant_id": r.Header.Get("X-PS-Tenant-ID"),
			})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		
		r.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "dev-user-123", response["user_id"])
		assert.Equal(t, "dev-tenant-123", response["tenant_id"])
	})
}

func TestDebugEndpoints(t *testing.T) {
	// Set up test environment for debug endpoints
	os.Setenv("PS_DEV_BYPASS_AUTH", "true")
os.Setenv("PS_BFF_JWT_PUBLIC_KEY", string(pubPEM))
	os.Setenv("PS_BFF_JWT_ISSUER", "test-issuer")
	os.Setenv("PS_BFF_JWT_AUDIENCE", "test-audience")
	
	defer func() {
		os.Unsetenv("PS_DEV_BYPASS_AUTH")
		os.Unsetenv("PS_BFF_JWT_PUBLIC_KEY")
		os.Unsetenv("PS_BFF_JWT_ISSUER")
		os.Unsetenv("PS_BFF_JWT_AUDIENCE")
	}()

	r := chi.NewRouter()
	r.Use(jwtAuthMiddleware)
	
	// Register debug endpoints
	opt := Options{} // Empty options for testing
	registerDebugEndpoints(r, opt)

	t.Run("Debug Auth Endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/debug/auth", nil)
		req.Header.Set("X-PS-User-ID", "test-user-123")
		req.Header.Set("X-PS-Tenant-ID", "test-tenant-123")
		w := httptest.NewRecorder()
		
		r.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		data := response["data"].(map[string]interface{})
		authData := data["auth"].(map[string]interface{})
		headers := authData["headers"].(map[string]interface{})
		assert.Equal(t, "test-user-123", headers["x_ps_user_id"])
		assert.Equal(t, "test-tenant-123", headers["x_ps_tenant_id"])
	})

	t.Run("Debug JWT Config Endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/debug/jwt-config", nil)
		w := httptest.NewRecorder()
		
		r.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		data := response["data"].(map[string]interface{})
		jwtConfig := data["jwt_config"].(map[string]interface{})
		assert.Equal(t, true, jwtConfig["configured"])
		assert.Equal(t, true, jwtConfig["public_key_valid"])
		assert.Equal(t, "test-issuer", jwtConfig["issuer"])
		assert.Equal(t, "test-audience", jwtConfig["audience"])
	})

	t.Run("Debug Headers Endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/debug/headers", nil)
		req.Header.Set("X-Test-Header", "test-value")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		
		r.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		data := response["data"].(map[string]interface{})
		headers := data["headers"].(map[string]interface{})
		assert.Contains(t, headers, "X-Test-Header")
		assert.Contains(t, headers, "Authorization")
	})
}

// generateTestJWT generates a valid JWT token for testing
// This simulates what the frontend BFF would do
func generateTestJWT(t *testing.T, privPEM []byte) string {
    t.Helper()

    // Parse RSA private key
    priv, err := parseRSAPrivateKeyFromPEM(privPEM)
    if err != nil {
        t.Fatalf("parse private key: %v", err)
    }

    // Header
    header := map[string]any{
        "alg": "RS256",
        "typ": "JWT",
    }

    now := time.Now().Unix()
    // Payload with required claims expected by middleware
    payload := map[string]any{
        "iss":        "test-issuer",
        "aud":        "test-audience",
        "sub":        "test-user-123",
        "name":       "Test User",
        "email":      "test@example.com",
        "tenant_id":  "test-tenant-123",
        "roles":      []string{"admin"},
        "admin":      true,
        "iat":        now,
        "nbf":        now - 5,
        "exp":        now + 3600,
    }

    enc := base64.URLEncoding
    // Encode header and payload
    hb, _ := json.Marshal(header)
    pb, _ := json.Marshal(payload)
    hEnc := enc.EncodeToString(hb)
    pEnc := enc.EncodeToString(pb)

    signingInput := hEnc + "." + pEnc
    sum := sha256.Sum256([]byte(signingInput))
    sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
    if err != nil {
        t.Fatalf("sign: %v", err)
    }
    sEnc := enc.EncodeToString(sig)

    return signingInput + "." + sEnc
}

func parseRSAPrivateKeyFromPEM(b []byte) (*rsa.PrivateKey, error) {
    block, _ := pem.Decode(b)
    if block == nil {
        return nil, fmt.Errorf("no PEM block")
    }
    switch block.Type {
    case "PRIVATE KEY":
        // PKCS8
        k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
        if err != nil {
            return nil, err
        }
        if pk, ok := k.(*rsa.PrivateKey); ok {
            return pk, nil
        }
        return nil, fmt.Errorf("not RSA private key: %T", k)
    case "RSA PRIVATE KEY":
        return x509.ParsePKCS1PrivateKey(block.Bytes)
    default:
        return nil, fmt.Errorf("unsupported key type %s", block.Type)
    }
}
