package api

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
)

// Tenant error codes for structured error responses
const (
	TenantErrorMissing         = "TENANT_MISSING"
	TenantErrorInvalidFormat   = "TENANT_INVALID_FORMAT"
	TenantErrorNotFound        = "TENANT_NOT_FOUND"
	TenantErrorInactive        = "TENANT_INACTIVE"
	TenantErrorAccessDenied    = "TENANT_ACCESS_DENIED"
	TenantErrorContextFailed   = "TENANT_CONTEXT_ERROR"
	TenantErrorValidationError = "TENANT_VALIDATION_ERROR"
)

// TenantValidationError represents a structured tenant validation error
type TenantValidationError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e TenantValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Additional context keys for tenant information
// contextKey type is defined in middleware_common.go
const (
	tenantNameKey contextKey = "tenant_name"
)

// TenantInfo contains validated tenant information
type TenantInfo struct {
	ID           uuid.UUID
	Name         string
	Status       string
	PlanID       uuid.UUID
	PlanName     string
	APICallLimit int64
}

// writeTenantError writes a structured tenant error response
func writeTenantError(w http.ResponseWriter, r *http.Request, tenantErr TenantValidationError) {
	correlationID := getCorrelationID(r)
	
	// Add correlation ID to error details
	if tenantErr.Details == nil {
		tenantErr.Details = make(map[string]interface{})
	}
	tenantErr.Details["correlation_id"] = correlationID
	tenantErr.Details["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	tenantErr.Details["path"] = r.URL.Path
	tenantErr.Details["method"] = r.Method

	// Log the error with context
	slog.Error("Tenant validation failed",
		"error_code", tenantErr.Code,
		"message", tenantErr.Message,
		"correlation_id", correlationID,
		"path", r.URL.Path,
		"method", r.Method,
		"details", tenantErr.Details,
	)

	// Map error codes to HTTP status codes
	var statusCode int
	switch tenantErr.Code {
	case TenantErrorMissing, TenantErrorInvalidFormat:
		statusCode = http.StatusBadRequest
	case TenantErrorNotFound:
		statusCode = http.StatusNotFound
	case TenantErrorInactive, TenantErrorAccessDenied:
		statusCode = http.StatusForbidden
	default:
		statusCode = http.StatusInternalServerError
	}

	writeErrorJSON(w, statusCode, tenantErr.Code, tenantErr.Message, tenantErr.Details, r)
}

// tenantValidationMiddleware validates and injects tenant context
func tenantValidationMiddleware(db postgres.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            correlationID := getCorrelationID(r)
            
            // Dev bypass: skip tenant validation entirely
            if strings.EqualFold(strings.TrimSpace(os.Getenv("PS_DEV_BYPASS_AUTH")), "true") {
                slog.Debug("Tenant validation bypassed - dev mode", 
                    "correlation_id", correlationID,
                    "path", r.URL.Path)
                next.ServeHTTP(w, r)
                return
            }

            // For public endpoints, allow no tenant
            if isPublicEndpoint(r.URL.Path) {
                slog.Debug("Tenant validation skipped - public endpoint", 
                    "correlation_id", correlationID,
                    "path", r.URL.Path)
                next.ServeHTTP(w, r)
                return
            }

            // Extract tenant ID - prioritize JWT claims (X-PS-Tenant-ID) over legacy headers
            tenantIDStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
            tenantSource := "jwt_header"
            
            if tenantIDStr == "" {
                // Fallback to alternative header for compatibility
                tenantIDStr = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
                tenantSource = "legacy_header"
            }

            if tenantIDStr == "" {
                writeTenantError(w, r, TenantValidationError{
                    Code:    TenantErrorMissing,
                    Message: "Tenant ID is required",
                    Details: map[string]interface{}{
                        "expected_headers": []string{"X-PS-Tenant-ID", "X-Tenant-ID"},
                        "source": "missing",
                    },
                })
                return
            }

            // Validate UUID format
            tenantID, err := uuid.Parse(tenantIDStr)
            if err != nil {
                writeTenantError(w, r, TenantValidationError{
                    Code:    TenantErrorInvalidFormat,
                    Message: "Invalid tenant ID format",
                    Details: map[string]interface{}{
                        "provided_tenant_id": tenantIDStr,
                        "expected_format": "UUID (e.g., 123e4567-e89b-12d3-a456-426614174000)",
                        "source": tenantSource,
                        "parse_error": err.Error(),
                    },
                })
                return
            }

            // Validate tenant exists and is active
            tenant, err := validateTenant(db, tenantID)
            if err != nil {
                if err == sql.ErrNoRows {
                    writeTenantError(w, r, TenantValidationError{
                        Code:    TenantErrorNotFound,
                        Message: "Tenant does not exist",
                        Details: map[string]interface{}{
                            "tenant_id": tenantID.String(),
                            "source": tenantSource,
                        },
                    })
                } else {
                    slog.Error("Database error during tenant validation", 
                        "tenant_id", tenantID, 
                        "error", err,
                        "correlation_id", correlationID)
                    writeTenantError(w, r, TenantValidationError{
                        Code:    TenantErrorValidationError,
                        Message: "Failed to validate tenant",
                        Details: map[string]interface{}{
                            "tenant_id": tenantID.String(),
                            "database_error": err.Error(),
                        },
                    })
                }
                return
            }

            // Check if tenant is active
            if tenant.Status != "active" && tenant.Status != "trial" {
                writeTenantError(w, r, TenantValidationError{
                    Code:    TenantErrorInactive,
                    Message: "Tenant is not active",
                    Details: map[string]interface{}{
                        "tenant_id": tenantID.String(),
                        "tenant_name": tenant.Name,
                        "current_status": tenant.Status,
                        "allowed_statuses": []string{"active", "trial"},
                    },
                })
                return
            }

            // Add tenant info to context
            ctx := r.Context()
            type dbCtxKey string
            ctx = context.WithValue(ctx, tenantIDKey, tenantID.String())
            ctx = context.WithValue(ctx, tenantNameKey, tenant.Name)

            // Set database context for RLS - CRITICAL for multi-tenant security
            ctx = context.WithValue(ctx, dbCtxKey("db.tenant_id"), tenantID.String())

            // Set tenant context in the database for RLS policies
            if err := setTenantContextInDB(db, tenantID); err != nil {
                slog.Error("Failed to set tenant context for RLS", 
                    "tenant_id", tenantID, 
                    "error", err,
                    "correlation_id", correlationID)
                writeTenantError(w, r, TenantValidationError{
                    Code:    TenantErrorContextFailed,
                    Message: "Failed to set tenant security context",
                    Details: map[string]interface{}{
                        "tenant_id": tenantID.String(),
                        "rls_error": err.Error(),
                    },
                })
                return
            }

            // Enforce user membership in tenant (skip for admin and specific onboarding endpoints)
            if !shouldBypassMembershipCheck(r) {
                // Allow platform admin
                if isAdminFromHeaders(r) {
                    slog.Debug("Tenant membership check bypassed - platform admin", 
                        "correlation_id", correlationID,
                        "tenant_id", tenantID.String())
                    next.ServeHTTP(w, r.WithContext(ctx))
                    return
                }
                
                userID := strings.TrimSpace(r.Header.Get("X-PS-User-ID"))
                if userID == "" {
                    writeTenantError(w, r, TenantValidationError{
                        Code:    TenantErrorAccessDenied,
                        Message: "User authentication required for tenant access",
                        Details: map[string]interface{}{
                            "tenant_id": tenantID.String(),
                            "missing_header": "X-PS-User-ID",
                        },
                    })
                    return
                }
                
                // Check membership using dedicated function
                if err := validateTenantMembership(db, tenantID, userID); err != nil {
                    slog.Error("Tenant membership validation failed", 
                        "tenant_id", tenantID, 
                        "user_id", userID,
                        "error", err,
                        "correlation_id", correlationID)
                    
                    writeTenantError(w, r, TenantValidationError{
                        Code:    TenantErrorAccessDenied,
                        Message: "User is not a member of this tenant",
                        Details: map[string]interface{}{
                            "tenant_id": tenantID.String(),
                            "tenant_name": tenant.Name,
                            "user_id": userID,
                            "membership_error": err.Error(),
                        },
                    })
                    return
                }
            }

            // Log successful tenant validation
            slog.Debug("Tenant validation successful",
                "correlation_id", correlationID,
                "tenant_id", tenantID.String(),
                "tenant_name", tenant.Name,
                "tenant_status", tenant.Status,
                "source", tenantSource,
                "path", r.URL.Path,
                "method", r.Method,
            )

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Removed - frontend authentication is handled by the frontend/auth service

// Removed - API key authentication is handled by the frontend/auth service

// Helper functions

func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/healthz",
		"/readyz",
		"/metrics",
		"/v1/health",
		"/v1/ready",
	}
	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}
	return false
}

func validateTenant(db postgres.DB, tenantID uuid.UUID) (*TenantInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			t.id, 
			t.name, 
			COALESCE(s.status, 'inactive') as status, 
			COALESCE(s.plan_id, '00000000-0000-0000-0000-000000000000'::uuid) as plan_id,
			COALESCE(sp.name, 'No Plan') as plan_name, 
			sp.limits
		FROM tenants t
		LEFT JOIN subscriptions s ON t.id = s.tenant_id AND s.deleted_at IS NULL
		LEFT JOIN subscription_plans sp ON s.plan_id = sp.id AND sp.deleted_at IS NULL
		WHERE t.id = $1 AND (t.deleted_at IS NULL OR t.deleted_at > NOW())
	`

	var tenant TenantInfo
	var limits sql.NullString

	err := db.QueryRowContext(ctx, query, tenantID).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Status,
		&tenant.PlanID,
		&tenant.PlanName,
		&limits,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant %s not found or deleted", tenantID.String())
		}
		return nil, fmt.Errorf("failed to query tenant %s: %w", tenantID.String(), err)
	}

	// Validate tenant data
	if tenant.Name == "" {
		return nil, fmt.Errorf("tenant %s has invalid data: empty name", tenantID.String())
	}

	// Parse limits from JSON
	if limits.Valid {
		// TODO: Parse JSON limits to get api_calls_monthly
		// For now, default to 10000
		tenant.APICallLimit = 10000
	} else {
		// Default limits for tenants without subscription
		tenant.APICallLimit = 1000
	}

	return &tenant, nil
}

// validateTenantMembership checks if a user is a member of a tenant with proper error handling
func validateTenantMembership(db postgres.DB, tenantID uuid.UUID, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT 
			tm.role,
			tm.created_at,
			tm.updated_at
		FROM tenant_memberships tm
		WHERE tm.tenant_id = $1 AND tm.user_id = $2 AND tm.deleted_at IS NULL
	`

	var role string
	var createdAt, updatedAt time.Time

	err := db.QueryRowContext(ctx, query, tenantID, userID).Scan(&role, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user %s is not a member of tenant %s", userID, tenantID.String())
		}
		return fmt.Errorf("failed to check membership for user %s in tenant %s: %w", userID, tenantID.String(), err)
	}

	// Log successful membership validation
	slog.Debug("Tenant membership validated",
		"user_id", userID,
		"tenant_id", tenantID.String(),
		"role", role,
		"member_since", createdAt.Format(time.RFC3339),
	)

	return nil
}

// Removed - user and API key validation is handled by frontend/auth service

// setTenantContextInDB sets the tenant context for RLS policies with proper error handling
func setTenantContextInDB(db postgres.DB, tenantID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Call the set_tenant_context() function created in our RLS migration
	query := "SELECT set_tenant_context($1::uuid)"
	
	var result bool
	err := db.QueryRowContext(ctx, query, tenantID).Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to set tenant context for RLS: %w", err)
	}
	
	if !result {
		return fmt.Errorf("set_tenant_context returned false for tenant %s", tenantID.String())
	}
	
	slog.Debug("Tenant RLS context set successfully", 
		"tenant_id", tenantID.String())
	
	return nil
}

// (removed) validateTenantAccessInDB — membership validation is handled inline

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) (uuid.UUID, bool) {
	if v := ctx.Value(tenantIDKey); v != nil {
		if idStr, ok := v.(string); ok {
			if id, err := uuid.Parse(idStr); err == nil {
				return id, true
			}
		}
	}
	return uuid.UUID{}, false
}

// Removed - user context is handled by frontend

// isAdminFromHeaders determines if request has admin role/header
func isAdminFromHeaders(r *http.Request) bool {
    roles := strings.Split(strings.TrimSpace(r.Header.Get("X-PS-User-Roles")), ",")
    for _, part := range roles {
        if strings.EqualFold(strings.TrimSpace(part), "admin") {
            return true
        }
    }
    return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-PS-User-Admin")), "true")
}

// shouldBypassMembershipCheck allows non-members to hit specific onboarding endpoints
func shouldBypassMembershipCheck(r *http.Request) bool {
    p := r.URL.Path
    // Allow self-membership creation: POST /v1/tenants/{id}/memberships/self
    if r.Method == http.MethodPost && strings.HasPrefix(p, "/v1/tenants/") && strings.HasSuffix(p, "/memberships/self") {
        return true
    }
    // Allow external org resolution and public endpoints
    if strings.HasPrefix(p, "/v1/tenants/resolve") {
        return true
    }
    // Admin and system endpoints are already guarded elsewhere
    if strings.HasPrefix(p, "/v1/admin/") || strings.HasPrefix(p, "/admin/") {
        return true
    }
    // Health/ready endpoints handled earlier as public
    return false
}
