package api

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
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
			ctx = context.WithValue(ctx, tenantIDKey, tenantID.String())
			ctx = context.WithValue(ctx, tenantNameKey, tenant.Name)
			
			// Set database context for RLS - CRITICAL for multi-tenant security
			ctx = context.WithValue(ctx, "db.tenant_id", tenantID.String())
			
			// Set tenant context in the database for RLS policies
			if err := setTenantContextInDB(db, tenantID); err != nil {
				slog.Error("Failed to set tenant context for RLS", "tenant_id", tenantID, "error", err)
				// This is critical - if RLS context fails, reject the request
				writeError(w, http.StatusInternalServerError, "TENANT_CONTEXT_ERROR", "Failed to set tenant security context", nil)
				return
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