package api

import (
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
)

// registerDebugEndpoints registers debugging endpoints for authentication troubleshooting
func registerDebugEndpoints(r chi.Router, opt Options) {
	r.Route("/debug", func(dr chi.Router) {
		// Only enable debug endpoints in development or when explicitly enabled
		dr.Use(debugAuthMiddleware(opt))
		
		dr.Get("/auth", debugAuthHandler(opt))
		dr.Get("/jwt-config", debugJWTConfigHandler(opt))
		dr.Get("/tenant-context", debugTenantContextHandler(opt))
		dr.Get("/headers", debugHeadersHandler())
			// PDP config status (no network calls)
			dr.Get("/pdp-config", debugPDPConfigHandler())
	})
}

// debugAuthMiddleware protects debug endpoints
func debugAuthMiddleware(opt Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow in dev bypass mode
			if strings.EqualFold(strings.TrimSpace(os.Getenv("PS_DEV_BYPASS_AUTH")), "true") {
				next.ServeHTTP(w, r)
				return
			}
			
			// Allow if debug endpoints are explicitly enabled
			if strings.EqualFold(strings.TrimSpace(os.Getenv("PS_ENABLE_DEBUG_ENDPOINTS")), "true") {
				next.ServeHTTP(w, r)
				return
			}
			
			// Allow in non-production environments
			env := strings.ToLower(strings.TrimSpace(os.Getenv("NODE_ENV")))
			if env != "production" {
				next.ServeHTTP(w, r)
				return
			}
			
			// Otherwise, require admin auth
			adminAuth(opt)(next).ServeHTTP(w, r)
		})
	}
}

// debugAuthHandler provides authentication debugging information
func debugAuthHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlationID := getCorrelationID(r)
		
		// Extract auth information from headers
		authInfo := map[string]interface{}{
			"headers": map[string]string{
				"authorization":    r.Header.Get("Authorization"),
				"x_ps_user_id":     r.Header.Get("X-PS-User-ID"),
				"x_ps_user_name":   r.Header.Get("X-PS-User-Name"),
				"x_ps_tenant_id":   r.Header.Get("X-PS-Tenant-ID"),
				"x_ps_user_roles":  r.Header.Get("X-PS-User-Roles"),
				"x_ps_user_admin":  r.Header.Get("X-PS-User-Admin"),
				"x_correlation_id": correlationID,
			},
			"context": map[string]interface{}{
				"correlation_id": correlationID,
				"path":          r.URL.Path,
				"method":        r.Method,
				"remote_addr":   r.RemoteAddr,
				"user_agent":    r.UserAgent(),
			},
		}
		
		// Check tenant context if available
		if tenantID, ok := GetTenantID(r.Context()); ok {
			authInfo["tenant_context"] = map[string]interface{}{
				"tenant_id": tenantID.String(),
				"from_context": true,
			}
		}
		
		response := map[string]interface{}{
			"auth":      authInfo,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"server":    "go-gateway",
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// debugPDPConfigHandler shows PDP integration configuration
func debugPDPConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		endpoint := strings.TrimSpace(os.Getenv("PS_PDP_ENDPOINT"))
		apiKeySet := strings.TrimSpace(os.Getenv("PS_PDP_API_KEY")) != ""
		to := strings.TrimSpace(os.Getenv("PS_PDP_TIMEOUT_MS"))
		status := map[string]any{
			"configured": endpoint != "",
			"endpoint":   endpoint,
			"has_api_key": apiKeySet,
			"timeout_ms": to,
		}
		writeJSON(w, http.StatusOK, map[string]any{"pdp": status, "timestamp": time.Now().UTC().Format(time.RFC3339)}, r)
	}
}

// debugJWTConfigHandler provides JWT configuration debugging information
func debugJWTConfigHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pubKeyPEM := strings.TrimSpace(os.Getenv("PS_BFF_JWT_PUBLIC_KEY"))
		issuer := strings.TrimSpace(os.Getenv("PS_BFF_JWT_ISSUER"))
		audience := strings.TrimSpace(os.Getenv("PS_BFF_JWT_AUDIENCE"))
		leeway := strings.TrimSpace(os.Getenv("PS_BFF_JWT_LEEWAY"))
		
		jwtConfig := map[string]interface{}{
			"configured":        pubKeyPEM != "",
			"has_public_key":    pubKeyPEM != "",
			"public_key_length": len(pubKeyPEM),
			"issuer":           issuer,
			"audience":         audience,
			"leeway":           leeway,
		}
		
		// Test public key parsing if available
		if pubKeyPEM != "" {
			if _, err := parseRSAPublicKeyFromPEM([]byte(pubKeyPEM)); err != nil {
				jwtConfig["public_key_valid"] = false
				jwtConfig["public_key_error"] = err.Error()
			} else {
				jwtConfig["public_key_valid"] = true
			}
		}
		
		response := map[string]interface{}{
			"jwt_config": jwtConfig,
			"environment": map[string]interface{}{
				"dev_bypass_auth": os.Getenv("PS_DEV_BYPASS_AUTH"),
				"node_env":       os.Getenv("NODE_ENV"),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// debugTenantContextHandler provides tenant context debugging information
func debugTenantContextHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantInfo := map[string]interface{}{
            "headers": map[string]string{
                "x_ps_tenant_id": r.Header.Get("X-PS-Tenant-ID"),
            },
		}
		
		// Check tenant context
		if tenantID, ok := GetTenantID(r.Context()); ok {
			tenantInfo["context"] = map[string]interface{}{
				"tenant_id":    tenantID.String(),
				"from_context": true,
			}
			
			// Try to validate tenant if database is available
			if opt.DB != nil {
				if tenant, err := validateTenant(opt.DB, tenantID); err != nil {
					tenantInfo["validation"] = map[string]interface{}{
						"valid": false,
						"error": err.Error(),
					}
				} else {
					tenantInfo["validation"] = map[string]interface{}{
						"valid":       true,
						"name":        tenant.Name,
						"status":      tenant.Status,
						"plan_name":   tenant.PlanName,
						"api_limit":   tenant.APICallLimit,
					}
				}
			}
		} else {
			tenantInfo["context"] = map[string]interface{}{
				"tenant_id":    nil,
				"from_context": false,
			}
		}
		
		response := map[string]interface{}{
			"tenant": tenantInfo,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// debugHeadersHandler shows all request headers for debugging
func debugHeadersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		headers := make(map[string][]string)
		for name, values := range r.Header {
			headers[name] = values
		}
		
		response := map[string]interface{}{
			"headers":   headers,
			"method":    r.Method,
			"path":      r.URL.Path,
			"query":     r.URL.RawQuery,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}
