package controlplane

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
)

type ControlPlaneHandler struct {
	tenantRepo     domain.TenantRepository
	rulepackRepo   postgres.RulepackRepository
	assignmentRepo domain.PolicyAssignmentRepository
	auditRepo      domain.AuditRepository
	rulepackSvc    *services.RulepackService
	validationSvc  *services.ValidationService
	publisher      *nats.Publisher
}

func NewControlPlaneHandler(
	tenantRepo domain.TenantRepository,
	rulepackRepo postgres.RulepackRepository,
	assignmentRepo domain.PolicyAssignmentRepository,
	auditRepo domain.AuditRepository,
	rulepackSvc *services.RulepackService,
	validationSvc *services.ValidationService,
	publisher *nats.Publisher,
) *ControlPlaneHandler {
	return &ControlPlaneHandler{
		tenantRepo:     tenantRepo,
		rulepackRepo:   rulepackRepo,
		assignmentRepo: assignmentRepo,
		auditRepo:      auditRepo,
		rulepackSvc:    rulepackSvc,
		validationSvc:  validationSvc,
		publisher:      publisher,
	}
}

// CreateRulepack creates a new rulepack for a tenant
func (h *ControlPlaneHandler) CreateRulepack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID    string `json:"tenantId"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenant ID"})
		return
	}

	id, err := h.rulepackRepo.Create(r.Context(), tenantID, req.Name, req.Description)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Audit the creation (best effort, don't fail the request)
	_ = h.auditRepo.Create(r.Context(), &domain.AuditEntry{
		TenantID:   &tenantID,
		Action:     "create_rulepack",
		ObjectType: "rulepack",
		ObjectID:   id,
	})

	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

// UploadVersion uploads a new version of a rulepack
func (h *ControlPlaneHandler) UploadVersion(w http.ResponseWriter, r *http.Request) {
	packID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rulepack ID"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}

	var req struct {
		Version int             `json:"version"`
		DSL     json.RawMessage `json:"dsl"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// Validate and normalize the DSL
	if err := h.validationSvc.ValidateDSL(r.Context(), req.DSL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	normalizedDSL, err := h.validationSvc.NormalizeDSL(r.Context(), req.DSL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to normalize DSL: " + err.Error()})
		return
	}

	versionID, err := h.rulepackRepo.CreateVersion(r.Context(), packID, req.Version, normalizedDSL, "draft", uuid.Nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"versionId": versionID.String(),
		"status":    "draft",
	})
}

// ApproveVersion approves a version for activation
func (h *ControlPlaneHandler) ApproveVersion(w http.ResponseWriter, r *http.Request) {
	packID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rulepack ID"})
		return
	}

	version, err := strconv.Atoi(chi.URLParam(r, "ver"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid version"})
		return
	}

	if err := h.rulepackRepo.ApproveVersion(r.Context(), packID, version); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// ActivateVersion activates a rulepack version
func (h *ControlPlaneHandler) ActivateVersion(w http.ResponseWriter, r *http.Request) {
	packID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rulepack ID"})
		return
	}

	version, err := strconv.Atoi(chi.URLParam(r, "ver"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid version"})
		return
	}

	var req struct {
		TenantID string          `json:"tenantId"`
		DSL      json.RawMessage `json:"dsl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenant ID"})
		return
	}

	if err := h.rulepackSvc.CreateVersionActivate(r.Context(), tenantID, packID, version, req.DSL); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "activated"})
}

// CreateAssignment assigns a rulepack to a target scope
func (h *ControlPlaneHandler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID    string `json:"tenantId"`
		RulepackID  string `json:"rulepackId"`
		TargetScope string `json:"targetScope"`
		Priority    int    `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenant ID"})
		return
	}

	rulepackID, err := uuid.Parse(req.RulepackID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rulepack ID"})
		return
	}

	assignment := &domain.PolicyAssignment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		PolicyID:    rulepackID,
		TargetScope: req.TargetScope,
		Priority:    req.Priority,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = h.assignmentRepo.Create(r.Context(), assignment)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Publish update event
	if h.publisher != nil {
		dsl, version, _ := h.rulepackRepo.GetActive(r.Context(), rulepackID)
		if dsl != nil {
			update := nats.RuleUpdate{
				TenantID:      tenantID.String(),
				TargetScope:   req.TargetScope,
				RulepackID:    rulepackID.String(),
				Version:       version,
				ContentSHA256: checksumJSON(dsl),
			}
			// Publish update event (best effort)
			_ = h.publisher.PublishRuleUpdate(r.Context(), update)
		}
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": assignment.ID.String()})
}

// GetRulepack retrieves a rulepack with its current version
func (h *ControlPlaneHandler) GetRulepack(w http.ResponseWriter, r *http.Request) {
	packID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rulepack ID"})
		return
	}

	dsl, version, err := h.rulepackRepo.GetActive(r.Context(), packID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "rulepack not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      packID.String(),
		"version": version,
		"dsl":     dsl,
	})
}

// GetRulepackVersion retrieves a specific version of a rulepack
func (h *ControlPlaneHandler) GetRulepackVersion(w http.ResponseWriter, r *http.Request) {
	packID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rulepack ID"})
		return
	}

	version, err := strconv.Atoi(chi.URLParam(r, "ver"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid version"})
		return
	}

	dsl, status, err := h.rulepackRepo.GetVersion(r.Context(), packID, version)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": packID.String(), "version": version, "status": status, "dsl": dsl})
}

// StreamUpdates provides server-sent events for rule updates
func (h *ControlPlaneHandler) StreamUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// For now, send heartbeat events every 30 seconds
	// In a full implementation, this would subscribe to Redis streams or NATS
	// and forward real rulepack update events to connected clients
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func checksumJSON(raw json.RawMessage) string {
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
