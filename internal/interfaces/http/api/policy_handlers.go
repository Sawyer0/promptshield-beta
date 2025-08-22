package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// PolicyHandlers handles HTTP requests for policy management
type PolicyHandlers struct {
	policyService contracts.PolicyService
}

// NewPolicyHandlers creates a new policy handlers instance
func NewPolicyHandlers(policyService contracts.PolicyService) *PolicyHandlers {
	return &PolicyHandlers{
		policyService: policyService,
	}
}

// RegisterRoutes registers policy management routes
func (h *PolicyHandlers) RegisterRoutes(r chi.Router, opt Options) {
	r.Route("/admin/policies", func(pr chi.Router) {
		pr.Use(adminAuth(opt))
		
		pr.Get("/", h.ListPolicies)
		pr.Post("/", h.CreatePolicy)
		pr.Get("/{id}", h.GetPolicy)
		pr.Put("/{id}", h.UpdatePolicy)
		pr.Delete("/{id}", h.DeletePolicy)
		pr.Post("/{id}/activate", h.ActivatePolicy)
		pr.Post("/{id}/deactivate", h.DeactivatePolicy)
		pr.Post("/{id}/test", h.TestPolicy)
		pr.Post("/validate", h.ValidatePolicy)
	})
}

// ListPolicies handles GET /v1/admin/policies
func (h *PolicyHandlers) ListPolicies(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := h.parseListFilter(r)

	// Get policies from service
	policies, total, err := h.policyService.ListPolicies(r.Context(), filter)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "LIST_FAILED", 
			"Failed to list policies", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Convert to response format
	response := map[string]interface{}{
		"policies":    h.convertPoliciesToResponse(policies),
		"count":       len(policies),
		"total_count": total,
	}

	writeJSON(w, http.StatusOK, response, r)
}

// CreatePolicy handles POST /v1/admin/policies
func (h *PolicyHandlers) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req PolicyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
			"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Convert request to policy
	policy := h.convertRequestToPolicy(&req)

	// Create policy via service
	createdPolicy, err := h.policyService.CreatePolicy(r.Context(), policy)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", 
			"Failed to create policy", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Return created policy
	response := h.convertPolicyToResponse(createdPolicy)
	writeJSON(w, http.StatusCreated, response, r)
}

// GetPolicy handles GET /v1/admin/policies/{id}
func (h *PolicyHandlers) GetPolicy(w http.ResponseWriter, r *http.Request) {
	policyIDStr := chi.URLParam(r, "id")
	if policyIDStr == "" {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Policy ID is required", nil, r)
		return
	}

	// Parse UUID
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Invalid policy ID format", map[string]interface{}{"policy_id": policyIDStr}, r)
		return
	}

	// Get policy from service
	policy, err := h.policyService.GetPolicy(r.Context(), policyID)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "POLICY_NOT_FOUND", 
			"Policy not found", map[string]interface{}{"policy_id": policyID.String()}, r)
		return
	}

	// Return policy
	response := h.convertPolicyToResponse(policy)
	writeJSON(w, http.StatusOK, response, r)
}

// UpdatePolicy handles PUT /v1/admin/policies/{id}
func (h *PolicyHandlers) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	policyIDStr := chi.URLParam(r, "id")
	if policyIDStr == "" {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Policy ID is required", nil, r)
		return
	}

	// Parse UUID
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Invalid policy ID format", map[string]interface{}{"policy_id": policyIDStr}, r)
		return
	}

	var req PolicyUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
			"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Convert request to policy
	policy := h.convertUpdateRequestToPolicy(policyID, &req)

	// Update policy via service
	updatedPolicy, err := h.policyService.UpdatePolicy(r.Context(), policy)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "UPDATE_FAILED", 
			"Failed to update policy", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Return updated policy
	response := h.convertPolicyToResponse(updatedPolicy)
	writeJSON(w, http.StatusOK, response, r)
}

// DeletePolicy handles DELETE /v1/admin/policies/{id}
func (h *PolicyHandlers) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	policyIDStr := chi.URLParam(r, "id")
	if policyIDStr == "" {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Policy ID is required", nil, r)
		return
	}

	// Parse UUID
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Invalid policy ID format", map[string]interface{}{"policy_id": policyIDStr}, r)
		return
	}

	// Delete policy via service
	if err := h.policyService.DeletePolicy(r.Context(), policyID); err != nil {
		writeErrorJSON(w, http.StatusNotFound, "DELETE_FAILED", 
			"Failed to delete policy", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Return success
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"policy_id": policyID,
		"status":    "deleted",
	}, r)
}

// ActivatePolicy handles POST /v1/admin/policies/{id}/activate
func (h *PolicyHandlers) ActivatePolicy(w http.ResponseWriter, r *http.Request) {
	policyIDStr := chi.URLParam(r, "id")
	if policyIDStr == "" {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Policy ID is required", nil, r)
		return
	}

	// Parse UUID
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Invalid policy ID format", map[string]interface{}{"policy_id": policyIDStr}, r)
		return
	}

	// Activate policy via service
	if err := h.policyService.ActivatePolicy(r.Context(), policyID); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "ACTIVATION_FAILED", 
			"Failed to activate policy", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Return success
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"policy_id": policyID,
		"status":    "activated",
	}, r)
}

// DeactivatePolicy handles POST /v1/admin/policies/{id}/deactivate  
func (h *PolicyHandlers) DeactivatePolicy(w http.ResponseWriter, r *http.Request) {
	policyIDStr := chi.URLParam(r, "id")
	if policyIDStr == "" {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Policy ID is required", nil, r)
		return
	}

	// Parse UUID
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Invalid policy ID format", map[string]interface{}{"policy_id": policyIDStr}, r)
		return
	}

	// Deactivate policy via service
	if err := h.policyService.DeactivatePolicy(r.Context(), policyID); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "DEACTIVATION_FAILED", 
			"Failed to deactivate policy", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Return success
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"policy_id": policyID,
		"status":    "deactivated",
	}, r)
}

// TestPolicy handles POST /v1/admin/policies/{id}/test
func (h *PolicyHandlers) TestPolicy(w http.ResponseWriter, r *http.Request) {
	policyIDStr := chi.URLParam(r, "id")
	if policyIDStr == "" {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Policy ID is required", nil, r)
		return
	}

	// Parse UUID
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
			"Invalid policy ID format", map[string]interface{}{"policy_id": policyIDStr}, r)
		return
	}

	var req PolicyTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
			"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Test policy via service
	result, err := h.policyService.TestPolicy(r.Context(), policyID, req.Content)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "TEST_FAILED", 
			"Failed to test policy", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Return test result
	writeJSON(w, http.StatusOK, result, r)
}

// ValidatePolicy handles POST /v1/admin/policies/validate
func (h *PolicyHandlers) ValidatePolicy(w http.ResponseWriter, r *http.Request) {
	var req PolicyValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
			"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
		return
	}

	// Convert request to policy
	policy := h.convertValidateRequestToPolicy(&req)

	// Validate policy via service
	if err := h.policyService.ValidatePolicy(r.Context(), policy); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"errors": []string{err.Error()},
		}, r)
		return
	}

	// Return validation success
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":  true,
		"errors": []string{},
	}, r)
}

// Helper methods for the handlers (keeping them focused and single-responsibility)

func (h *PolicyHandlers) parseListFilter(r *http.Request) map[string]interface{} {
	filter := make(map[string]interface{})
	
	query := r.URL.Query()
	
	// Parse limit
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filter["limit"] = limit
		} else {
			filter["limit"] = 50 // default
		}
	}
	
	// Parse type filter
	if policyType := query.Get("type"); policyType != "" {
		filter["type"] = types.PolicyType(policyType)
	}
	
	// Parse name filter
	if name := query.Get("name"); name != "" {
		filter["name"] = name
	}
	
	return filter
}

func (h *PolicyHandlers) convertPolicyToResponse(policy *types.Policy) PolicyResponse {
	return PolicyResponse{
		ID:          policy.ID.String(),
		Name:        policy.Name,
		Version:     policy.Version,
		Type:        string(policy.Type),
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}
}

func (h *PolicyHandlers) convertPoliciesToResponse(policies []*types.Policy) []PolicyResponse {
	result := make([]PolicyResponse, len(policies))
	for i, policy := range policies {
		result[i] = h.convertPolicyToResponse(policy)
	}
	return result
}

func (h *PolicyHandlers) convertRequestToPolicy(req *PolicyCreateRequest) *types.Policy {
	return &types.Policy{
		Name:    req.Name,
		Content: req.Content,
		Type:    types.PolicyType(req.Type),
	}
}

func (h *PolicyHandlers) convertUpdateRequestToPolicy(id uuid.UUID, req *PolicyUpdateRequest) *types.Policy {
	return &types.Policy{
		ID:      id,
		Name:    req.Name,
		Content: req.Content,
	}
}

func (h *PolicyHandlers) convertValidateRequestToPolicy(req *PolicyValidateRequest) *types.Policy {
	return &types.Policy{
		Name:    req.Name,
		Content: req.Content,
		Type:    types.PolicyType(req.Type),
	}
}