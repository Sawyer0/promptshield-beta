package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// registerTenantHandlers registers all tenant-related endpoints
func registerTenantHandlers(r chi.Router, opt Options) {
	r.Route("/admin/tenants", func(tr chi.Router) {
		tr.Use(adminAuth(opt))

		tr.Post("/", createTenantHandler(opt))
		tr.Get("/", listTenantsHandler(opt))
		tr.Get("/{id}", getTenantHandler(opt))
		tr.Put("/{id}", updateTenantHandler(opt))
		tr.Delete("/{id}", deleteTenantHandler(opt))
	})
    // Non-admin: list tenants for current user (based on membership)
    r.Get("/tenants/my", listMyTenantsHandler(opt))
    // Public mapping endpoint used by BFF to resolve Clerk org -> tenant
    r.Post("/tenants/resolve", resolveExternalOrgHandler(opt))
    // Self-membership upsert (bypass enforced in tenant middleware)
    r.Post("/tenants/{id}/memberships/self", upsertSelfMembershipHandler(opt))
}

// createTenantHandler handles POST /v1/admin/tenants
func createTenantHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.TenantRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"domain.Tenant management not configured", nil, r)
			return
		}

		var req struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		if req.Name == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"domain.Tenant name is required", nil, r)
			return
		}

		// Check if tenant already exists
		existing, _ := opt.TenantRepository.GetByName(r.Context(), req.Name)
		if existing != nil {
			writeErrorJSON(w, http.StatusConflict, "ALREADY_EXISTS",
				"domain.Tenant with this name already exists", map[string]interface{}{"name": req.Name}, r)
			return
		}

		// Create the tenant
		tenant := &domain.Tenant{
			ID:        uuid.New(),
			Name:      req.Name,
			Status:    domain.TenantStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := opt.TenantRepository.Create(r.Context(), tenant)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to create tenant", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Create owner membership for requesting user when available
		if opt.DB != nil {
			if userID := strings.TrimSpace(r.Header.Get("X-PS-User-ID")); userID != "" {
				_, _ = opt.DB.ExecContext(r.Context(),
					"INSERT INTO tenant_memberships (tenant_id, user_id, role) VALUES ($1,$2,'owner') ON CONFLICT DO NOTHING",
					tenant.ID, userID,
				)
			}
		}

		// Audit log the creation
		if opt.AuditRepository != nil {
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "tenant.create",
				ObjectType: "tenant",
				ObjectID:   tenant.ID,
				Metadata:   json.RawMessage(`{"name":"` + req.Name + `"}`),
				CreatedAt:  time.Now(),
			})
		}

		writeJSON(w, http.StatusCreated, tenant, r)
	}
}

// listTenantsHandler handles GET /v1/admin/tenants
func listTenantsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.TenantRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"domain.Tenant management not configured", nil, r)
			return
		}

		tenants, _, err := opt.TenantRepository.List(r.Context(), 0, 100)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to list tenants", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Return empty array if no tenants
		if tenants == nil {
			tenants = []*domain.Tenant{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"tenants": tenants,
			"count":   len(tenants),
		}, r)
	}
}

// getTenantHandler handles GET /v1/admin/tenants/{id}
func getTenantHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.TenantRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"domain.Tenant management not configured", nil, r)
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid tenant ID format", map[string]interface{}{"id": idStr}, r)
			return
		}

		tenant, err := opt.TenantRepository.Get(r.Context(), id)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND",
				"domain.Tenant not found", map[string]interface{}{"id": id.String()}, r)
			return
		}

		writeJSON(w, http.StatusOK, tenant, r)
	}
}

// updateTenantHandler handles PUT /v1/admin/tenants/{id}
func updateTenantHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.TenantRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"domain.Tenant management not configured", nil, r)
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid tenant ID format", map[string]interface{}{"id": idStr}, r)
			return
		}

		var req struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Verify tenant exists
		tenant, err := opt.TenantRepository.Get(r.Context(), id)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND",
				"domain.Tenant not found", map[string]interface{}{"id": id.String()}, r)
			return
		}

		// Update the name if provided
		if req.Name != "" && req.Name != tenant.Name {
			// Check if new name is already taken
			existing, _ := opt.TenantRepository.GetByName(r.Context(), req.Name)
			if existing != nil && existing.ID != id {
				writeErrorJSON(w, http.StatusConflict, "ALREADY_EXISTS",
					"domain.Tenant with this name already exists", map[string]interface{}{"name": req.Name}, r)
				return
			}

			tenant.Name = req.Name
			tenant.UpdatedAt = time.Now()

			// Note: The actual update would need to be implemented in the repository
			// For now, we're just returning the modified tenant
		}

		// Audit log the update
		if opt.AuditRepository != nil {
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "tenant.update",
				ObjectType: "tenant",
				ObjectID:   tenant.ID,
				Metadata:   json.RawMessage(`{"name":"` + req.Name + `"}`),
				CreatedAt:  time.Now(),
			})
		}

		writeJSON(w, http.StatusOK, tenant, r)
	}
}

// deleteTenantHandler handles DELETE /v1/admin/tenants/{id}
func deleteTenantHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.TenantRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"domain.Tenant management not configured", nil, r)
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid tenant ID format", map[string]interface{}{"id": idStr}, r)
			return
		}

		// Verify tenant exists
		tenant, err := opt.TenantRepository.Get(r.Context(), id)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND",
				"Tenant not found", map[string]interface{}{"id": id.String()}, r)
			return
		}

		// Delete the tenant
		err = opt.TenantRepository.Delete(r.Context(), id)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to delete tenant", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Audit log the deletion
		if opt.AuditRepository != nil {
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "tenant.delete",
				ObjectType: "tenant",
				ObjectID:   tenant.ID,
				CreatedAt:  time.Now(),
			})
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// listMyTenantsHandler handles GET /v1/tenants/my
func listMyTenantsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.DB == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"database not configured", nil, r)
			return
		}
		userID := strings.TrimSpace(r.Header.Get("X-PS-User-ID"))
		if userID == "" {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED",
				"user context required", nil, r)
			return
		}
		rows, err := opt.DB.QueryContext(r.Context(), `
            SELECT t.id, t.name, t.created_at, t.updated_at
            FROM tenant_memberships m
            JOIN tenants t ON t.id = m.tenant_id
            WHERE m.user_id = $1 AND (t.deleted_at IS NULL OR t.deleted_at > NOW())
            ORDER BY t.created_at DESC
        `, userID)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"failed to query memberships", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		defer rows.Close()
		var tenants []map[string]interface{}
		for rows.Next() {
			var id uuid.UUID
			var name string
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&id, &name, &createdAt, &updatedAt); err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
					"failed to scan row", map[string]interface{}{"error": err.Error()}, r)
				return
			}
			tenants = append(tenants, map[string]interface{}{
				"id":         id,
				"name":       name,
				"created_at": createdAt,
				"updated_at": updatedAt,
			})
		}
		// In dev bypass mode, if user has no memberships, return all tenants to simplify UI work
		if len(tenants) == 0 && strings.EqualFold(strings.TrimSpace(os.Getenv("PS_DEV_BYPASS_AUTH")), "true") {
			rows2, err := opt.DB.QueryContext(r.Context(), `
                SELECT id, name, created_at, updated_at
                FROM tenants
                WHERE deleted_at IS NULL OR deleted_at > NOW()
                ORDER BY created_at DESC
            `)
			if err == nil {
				defer rows2.Close()
				for rows2.Next() {
					var id uuid.UUID
					var name string
					var createdAt, updatedAt time.Time
					if err := rows2.Scan(&id, &name, &createdAt, &updatedAt); err != nil {
						break
					}
					tenants = append(tenants, map[string]interface{}{
						"id":         id,
						"name":       name,
						"created_at": createdAt,
						"updated_at": updatedAt,
					})
				}
			}
		}
		if tenants == nil {
			tenants = []map[string]interface{}{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"tenants": tenants,
			"count":   len(tenants),
		}, r)
	}
}

// upsertSelfMembershipHandler handles POST /v1/tenants/{id}/memberships/self
// It adds the current user as a member of the tenant if not already.
func upsertSelfMembershipHandler(opt Options) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if opt.DB == nil {
            writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
                "database not configured", nil, r)
            return
        }
        idStr := chi.URLParam(r, "id")
        id, err := uuid.Parse(idStr)
        if err != nil {
            writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
                "Invalid tenant ID format", map[string]interface{}{"id": idStr}, r)
            return
        }
        userID := strings.TrimSpace(r.Header.Get("X-PS-User-ID"))
        if userID == "" {
            writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "user context required", nil, r)
            return
        }
        // Upsert membership
        _, err = opt.DB.ExecContext(r.Context(), `
            INSERT INTO tenant_memberships (tenant_id, user_id, role, created_at, updated_at)
            VALUES ($1,$2,'member', NOW(), NOW())
            ON CONFLICT (tenant_id, user_id) DO UPDATE SET updated_at = NOW()
        `, id, userID)
        if err != nil {
            writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
                "failed to upsert membership", map[string]interface{}{"error": err.Error()}, r)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    }
}

// resolveExternalOrgHandler ensures a tenant exists for an external organization
// and returns the mapped tenant ID. Idempotent.
// POST /v1/tenants/resolve { provider: "clerk", external_org_id: "org_123", fallback_name?: "Acme" }
func resolveExternalOrgHandler(opt Options) http.HandlerFunc {
	type req struct {
		Provider      string `json:"provider"`
		ExternalOrgID string `json:"external_org_id"`
		FallbackName  string `json:"fallback_name"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.TenantRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"domain.Tenant management not configured", nil, r)
			return
		}
		var in req
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", "bad json", nil, r)
			return
		}
		in.Provider = strings.TrimSpace(in.Provider)
		in.ExternalOrgID = strings.TrimSpace(in.ExternalOrgID)
		if in.Provider == "" || in.ExternalOrgID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "provider and external_org_id required", nil, r)
			return
		}

		// Fast path: mapping already exists
		if t, err := opt.TenantRepository.GetByExternalOrg(r.Context(), in.Provider, in.ExternalOrgID); err == nil && t != nil {
			writeJSON(w, http.StatusOK, map[string]any{"tenant_id": t.ID}, r)
			return
		}

		// Otherwise, create/find tenant by fallback name if provided
		var tenant *domain.Tenant
		name := strings.TrimSpace(in.FallbackName)
		if name != "" {
			if existing, _ := opt.TenantRepository.GetByName(r.Context(), name); existing != nil {
				tenant = existing
			}
		}
		if tenant == nil {
			// Create a new tenant using fallback or external id as name
			nm := name
			if nm == "" {
				nm = in.ExternalOrgID
			}
			tenant = &domain.Tenant{
				ID:        uuid.New(),
				Name:      nm,
				Status:    domain.TenantStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := opt.TenantRepository.Create(r.Context(), tenant); err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "create tenant failed", map[string]any{"error": err.Error()}, r)
				return
			}
		}

		// Link mapping (upsert)
		if err := opt.TenantRepository.LinkExternalOrg(r.Context(), in.Provider, in.ExternalOrgID, tenant.ID); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "link mapping failed", map[string]any{"error": err.Error()}, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tenant_id": tenant.ID}, r)
	}
}
