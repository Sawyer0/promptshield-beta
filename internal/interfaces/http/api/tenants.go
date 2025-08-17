package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// registerTenantHandlers registers all tenant-related endpoints
func registerTenantHandlers(r chi.Router, opt Options) {
	r.Route("/v1/admin/tenants", func(tr chi.Router) {
		tr.Use(adminAuth(opt))
		tr.Use(correlationIDMiddleware)
		tr.Use(tenantContextMiddleware)
		
		tr.Post("/", createTenantHandler(opt))
		tr.Get("/", listTenantsHandler(opt))
		tr.Get("/{id}", getTenantHandler(opt))
		tr.Put("/{id}", updateTenantHandler(opt))
		tr.Delete("/{id}", deleteTenantHandler(opt))
	})
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