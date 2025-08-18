package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// registerAssignmentHandlers registers all assignment-related endpoints
func registerAssignmentHandlers(r chi.Router, opt Options) {
	// Assignment routes under tenant context
	r.Route("/v1/admin/tenants/{id}/assignments", func(ar chi.Router) {
		ar.Use(adminAuth(opt))
		
		ar.Post("/", createAssignmentHandler(opt))
		ar.Get("/", listAssignmentsHandler(opt))
	})
	
	// Direct assignment management
	r.Route("/v1/admin/assignments", func(ar chi.Router) {
		ar.Use(adminAuth(opt))
		
		ar.Put("/{assignmentId}", updateAssignmentHandler(opt))
		ar.Delete("/{assignmentId}", deleteAssignmentHandler(opt))
	})
}

// createAssignmentHandler handles POST /v1/admin/tenants/{id}/assignments
func createAssignmentHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AssignmentRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Assignment management not configured", nil, r)
			return
		}
		
		tenantIDStr := chi.URLParam(r, "id")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Invalid tenant ID format", map[string]interface{}{"id": tenantIDStr}, r)
			return
		}
		
		var req struct {
			RulepackID  uuid.UUID `json:"rulepack_id"`
			TargetScope string    `json:"target_scope"`
			Priority    int       `json:"priority"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Validate required fields
		if req.RulepackID == uuid.Nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Rulepack ID is required", nil, r)
			return
		}
		
		// Set defaults
		if req.TargetScope == "" {
			req.TargetScope = "*" // Default to all scopes
		}
		if req.Priority == 0 {
			req.Priority = 100 // Default priority
		}
		
		// Verify tenant exists if repository is available
		if opt.TenantRepository != nil {
			_, err = opt.TenantRepository.Get(r.Context(), tenantID)
			if err != nil {
				writeErrorJSON(w, http.StatusNotFound, "TENANT_NOT_FOUND", 
					"Tenant not found", map[string]interface{}{"tenant_id": tenantID.String()}, r)
				return
			}
		}
		
		// Create the assignment
		assignment := &domain.PolicyAssignment{
			ID:          uuid.New(),
			TenantID:    tenantID,
			PolicyID:    req.RulepackID, // Using RulepackID as PolicyID for now
			TargetScope: req.TargetScope,
			Priority:    req.Priority,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		
		err = opt.AssignmentRepository.Create(r.Context(), assignment)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to create assignment", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Audit log the creation
		if opt.AuditRepository != nil {
			metadata, _ := json.Marshal(map[string]interface{}{
				"tenant_id":    tenantID,
				"rulepack_id":  req.RulepackID,
				"target_scope": req.TargetScope,
				"priority":     req.Priority,
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				ID:         uuid.New(),
				Action:     "assignment.create",
				ObjectType: "assignment",
				ObjectID:   assignment.ID,
				TenantID:   &tenantID,
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}
		
		// Return the created assignment
		response := map[string]interface{}{
			"id":           assignment.ID,
			"tenant_id":    assignment.TenantID,
			"policy_id":    assignment.PolicyID,
			"target_scope": assignment.TargetScope,
			"priority":    assignment.Priority,
			"enabled":     assignment.Enabled,
			"created_at":  assignment.CreatedAt,
		}
		
		writeJSON(w, http.StatusCreated, response, r)
	}
}

// listAssignmentsHandler handles GET /v1/admin/tenants/{id}/assignments
func listAssignmentsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AssignmentRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Assignment management not configured", nil, r)
			return
		}
		
		tenantIDStr := chi.URLParam(r, "id")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Invalid tenant ID format", map[string]interface{}{"id": tenantIDStr}, r)
			return
		}
		
		// Get optional scope filter
		scope := r.URL.Query().Get("scope")
		
		var assignments []*domain.PolicyAssignment
		if scope != "" {
			assignments, err = opt.AssignmentRepository.ListByScope(r.Context(), tenantID, scope)
		} else {
			assignments, err = opt.AssignmentRepository.ListByTenant(r.Context(), tenantID)
		}
		
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to list assignments", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Return empty array if no assignments
		if assignments == nil {
			assignments = []*domain.PolicyAssignment{}
		}
		
		result := map[string]interface{}{
			"assignments": assignments,
			"count":       len(assignments),
			"tenant_id":   tenantID.String(),
		}
		
		if scope != "" {
			result["scope_filter"] = scope
		}
		
		writeJSON(w, http.StatusOK, result, r)
	}
}

// updateAssignmentHandler handles PUT /v1/admin/assignments/{assignmentId}
func updateAssignmentHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AssignmentRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Assignment management not configured", nil, r)
			return
		}
		
		assignmentIDStr := chi.URLParam(r, "assignmentId")
		assignmentID, err := uuid.Parse(assignmentIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Invalid assignment ID format", map[string]interface{}{"id": assignmentIDStr}, r)
			return
		}
		
		var req struct {
			Priority int `json:"priority"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Validate priority
		if req.Priority < 0 {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Priority must be non-negative", map[string]interface{}{"priority": req.Priority}, r)
			return
		}
		
		// Get existing assignment
		existing, err := opt.AssignmentRepository.Get(r.Context(), assignmentID)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "ASSIGNMENT_NOT_FOUND", 
				"Assignment not found", map[string]interface{}{"assignment_id": assignmentID.String()}, r)
			return
		}
		
		// Update the assignment
		existing.Priority = req.Priority
		existing.UpdatedAt = time.Now()
		
		err = opt.AssignmentRepository.Update(r.Context(), existing)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to update assignment", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Audit log the update
		if opt.AuditRepository != nil {
			metadata, _ := json.Marshal(map[string]interface{}{
				"priority": req.Priority,
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "assignment.update",
				ObjectType: "assignment",
				ObjectID:   assignmentID,
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}
		
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":       assignmentID.String(),
			"priority": req.Priority,
			"updated":  true,
		}, r)
	}
}

// deleteAssignmentHandler handles DELETE /v1/admin/assignments/{assignmentId}
func deleteAssignmentHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AssignmentRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Assignment management not configured", nil, r)
			return
		}
		
		assignmentIDStr := chi.URLParam(r, "assignmentId")
		assignmentID, err := uuid.Parse(assignmentIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Invalid assignment ID format", map[string]interface{}{"id": assignmentIDStr}, r)
			return
		}
		
		// Delete the assignment
		err = opt.AssignmentRepository.Delete(r.Context(), assignmentID)
		if err != nil {
			// Check if it's a not found error
			if err.Error() == "assignment not found" {
				writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", 
					"Assignment not found", map[string]interface{}{"id": assignmentID.String()}, r)
				return
			}
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to delete assignment", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Audit log the deletion
		if opt.AuditRepository != nil {
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "assignment.delete",
				ObjectType: "assignment",
				ObjectID:   assignmentID,
				CreatedAt:  time.Now(),
			})
		}
		
		w.WriteHeader(http.StatusNoContent)
	}
}