package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewMux(h *ControlPlaneHandler) http.Handler {
	r := chi.NewRouter()
	
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Rulepack management
	r.Post("/v1/rulepacks", h.CreateRulepack)
	r.Get("/v1/rulepacks/{id}", h.GetRulepack)
	r.Post("/v1/rulepacks/{id}/versions", h.UploadVersion)
	r.Get("/v1/rulepacks/{id}/versions/{ver}", h.GetRulepackVersion)
	r.Post("/v1/rulepacks/{id}/versions/{ver}/approve", h.ApproveVersion)
	r.Post("/v1/rulepacks/{id}/versions/{ver}/activate", h.ActivateVersion)

	// Assignments
	r.Post("/v1/assignments", h.CreateAssignment)

	// Streaming updates
	r.Get("/v1/stream", h.StreamUpdates)

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return r
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
