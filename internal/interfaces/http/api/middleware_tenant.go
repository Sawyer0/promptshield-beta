package api

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
)

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

// tenantValidationMiddleware validates and injects tenant context
func tenantValidationMiddleware(db postgres.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Dev bypass: skip tenant validation entirely
            if strings.EqualFold(strings.TrimSpace(os.Getenv("PS_DEV_BYPASS_AUTH")), "true") {
                next.ServeHTTP(w, r)
                return
            }
			// Extract tenant ID from header
			tenantIDStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
			if tenantIDStr == "" {
				// Check alternative header for compatibility
				tenantIDStr = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
			}

			// For public endpoints, allow no tenant
			if isPublicEndpoint(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			if tenantIDStr == "" {
				writeError(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID header is required", nil)
				return
			}

			// Validate UUID format
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "INVALID_TENANT", "Invalid tenant ID format", map[string]any{"tenant_id": tenantIDStr})
				return
			}

			// Validate tenant exists and is active
			tenant, err := validateTenant(db, tenantID)
			if err != nil {
				if err == sql.ErrNoRows {
					writeError(w, http.StatusNotFound, "TENANT_NOT_FOUND", "Tenant does not exist", map[string]any{"tenant_id": tenantID})
				} else {
					slog.Error("Failed to validate tenant", "tenant_id", tenantID, "error", err)
					writeError(w, http.StatusInternalServerError, "TENANT_VALIDATION_ERROR", "Failed to validate tenant", nil)
				}
				return
			}

			// Check if tenant is active
			if tenant.Status != "active" && tenant.Status != "trial" {
				writeError(w, http.StatusForbidden, "TENANT_INACTIVE", "Tenant is not active", map[string]any{
					"tenant_id": tenantID,
					"status":    tenant.Status,
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
                slog.Error("Failed to set tenant context for RLS", "tenant_id", tenantID, "error", err)
                // This is critical - if RLS context fails, reject the request
                writeError(w, http.StatusInternalServerError, "TENANT_CONTEXT_ERROR", "Failed to set tenant security context", nil)
                return
            }

            // Enforce user membership in tenant (skip for admin and specific onboarding endpoints)
            if !shouldBypassMembershipCheck(r) {
                // Allow platform admin
                if isAdminFromHeaders(r) {
                    next.ServeHTTP(w, r.WithContext(ctx))
                    return
                }
                userID := strings.TrimSpace(r.Header.Get("X-PS-User-ID"))
                if userID == "" {
                    writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user authentication required", nil)
                    return
                }
                // Membership exists?
                var dummy int
                row := db.QueryRowContext(ctx, "SELECT 1 FROM tenant_memberships WHERE tenant_id=$1 AND user_id=$2", tenantID, userID)
                if err := row.Scan(&dummy); err != nil {
                    writeError(w, http.StatusForbidden, "FORBIDDEN", "user is not a member of tenant", map[string]any{"tenant_id": tenantID})
                    return
                }
            }

            // Update metrics if available
            // Note: HTTPRequestsTotal metric should be defined in metrics package
            // For now, log the tenant access
            slog.Debug("Tenant request",
                "tenant_id", tenantID.String(),
                "method", r.Method,
                "path", r.URL.Path,
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
	query := `
		SELECT t.id, t.name, s.status, s.plan_id, sp.name, sp.limits
		FROM tenants t
		LEFT JOIN subscriptions s ON t.id = s.tenant_id
		LEFT JOIN subscription_plans sp ON s.plan_id = sp.id
		WHERE t.id = $1 AND t.deleted_at IS NULL
	`

	var tenant TenantInfo
	var limits sql.NullString

	err := db.QueryRowContext(context.Background(), query, tenantID).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Status,
		&tenant.PlanID,
		&tenant.PlanName,
		&limits,
	)

	if err != nil {
		return nil, err
	}

	// Parse limits from JSON
	if limits.Valid {
		// Parse JSON limits to get api_calls_monthly
		// For now, default to 10000
		tenant.APICallLimit = 10000
	}

	return &tenant, nil
}

// Removed - user and API key validation is handled by frontend/auth service

// setTenantContextInDB sets the tenant context for RLS policies
func setTenantContextInDB(db postgres.DB, tenantID uuid.UUID) error {
	// Call the set_tenant_context() function created in our RLS migration
	query := "SELECT set_tenant_context($1::uuid)"
	_, err := db.ExecContext(context.Background(), query, tenantID)
	return err
}

// validateTenantAccessInDB validates tenant access using RLS function
func validateTenantAccessInDB(db postgres.DB, tenantID uuid.UUID) error {
	// Call the validate_tenant_access() function created in our RLS migration
	var hasAccess bool
	query := "SELECT validate_tenant_access($1::uuid)"
	err := db.QueryRowContext(context.Background(), query, tenantID).Scan(&hasAccess)
	if err != nil {
		return err
	}

	if !hasAccess {
		return fmt.Errorf("tenant access denied by RLS policy")
	}

	return nil
}

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
    if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-PS-User-Admin")), "true") {
        return true
    }
    return false
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
