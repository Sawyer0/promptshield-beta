package api

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
    enc "github.com/promptshield/promptshield/internal/security/crypto"
)

// registerProviderProfileHandlers mounts /api/providers/profiles CRUD
func registerProviderProfileHandlers(r chi.Router, opt Options) {
    r.Route("/api/providers/profiles", func(pr chi.Router) {
        if opt.DB == nil {
            // No DB configured: return empty list and 501 for mutations
            pr.Get("/", func(w http.ResponseWriter, r *http.Request) {
                _ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
            })
            pr.Post("/", func(w http.ResponseWriter, r *http.Request) { writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r) })
            pr.Get("/{id}", func(w http.ResponseWriter, r *http.Request) { writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r) })
            pr.Put("/{id}", func(w http.ResponseWriter, r *http.Request) { writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r) })
            pr.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) { writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r) })
            return
        }

        repo := pg.ProviderProfiles(opt.DB)

        // List
        pr.Get("/", withTenant(func(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
            items, err := repo.List(r.Context(), tenantID)
            if err != nil { writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r); return }
            // Redact api_key
            var out []map[string]any
            for _, p := range items {
                out = append(out, map[string]any{
                    "id": p.ID, "tenant_id": p.TenantID, "provider": p.Provider, "label": p.Label,
                    "base_url": p.BaseURL, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
                })
            }
            _ = json.NewEncoder(w).Encode(map[string]any{"data": out})
        }))

        // Create
        pr.Post("/", withTenant(func(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
            var body struct { Provider, Label, APIKey, BaseURL string; ExtraHeaders json.RawMessage }
            _ = json.NewDecoder(r.Body).Decode(&body)
            if strings.TrimSpace(body.Provider) == "" || strings.TrimSpace(body.Label) == "" || strings.TrimSpace(body.APIKey) == "" {
                writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "provider, label and apiKey are required", nil, r); return
            }
            encKey, err := enc.EncryptString(body.APIKey)
            if err != nil {
                writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "encryption failed – set PS_ENCRYPTION_KEY (base64 32 bytes) or PS_TENANT_SECRET", nil, r)
                return
            }
            p := &pg.ProviderProfile{ TenantID: tenantID, Provider: body.Provider, Label: body.Label, APIKeyEnc: encKey }
            if strings.TrimSpace(body.BaseURL) != "" { p.BaseURL = &body.BaseURL }
            if len(body.ExtraHeaders) > 0 { p.ExtraHdrs = body.ExtraHeaders }
            if err := repo.Create(r.Context(), p); err != nil { writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r); return }
            w.WriteHeader(http.StatusCreated)
            _ = json.NewEncoder(w).Encode(map[string]any{"id": p.ID, "provider": p.Provider, "label": p.Label, "base_url": p.BaseURL})
        }))

        // Get
        pr.Get("/{id}", withTenantAndID("id", func(w http.ResponseWriter, r *http.Request, tenantID, id uuid.UUID) {
            p, err := repo.Get(r.Context(), tenantID, id); if err != nil { writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "profile not found", nil, r); return }
            _ = json.NewEncoder(w).Encode(map[string]any{
                "id": p.ID, "tenant_id": p.TenantID, "provider": p.Provider, "label": p.Label,
                "base_url": p.BaseURL, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
            })
        }))

        // Update (apiKey optional)
        pr.Put("/{id}", withTenantAndID("id", func(w http.ResponseWriter, r *http.Request, tenantID, id uuid.UUID) {
            var body struct { Provider, Label, APIKey, BaseURL string; ExtraHeaders json.RawMessage }
            _ = json.NewDecoder(r.Body).Decode(&body)
            var encKey string
            var err error
            if strings.TrimSpace(body.APIKey) != "" {
                if encKey, err = enc.EncryptString(body.APIKey); err != nil {
                    writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "encryption failed – set PS_ENCRYPTION_KEY (base64 32 bytes) or PS_TENANT_SECRET", nil, r)
                    return
                }
            }
            p := &pg.ProviderProfile{ ID: id, TenantID: tenantID, Provider: body.Provider, Label: body.Label, APIKeyEnc: encKey }
            if strings.TrimSpace(body.BaseURL) != "" { p.BaseURL = &body.BaseURL }
            if len(body.ExtraHeaders) > 0 { p.ExtraHdrs = body.ExtraHeaders }
            if err := repo.Update(r.Context(), p); err != nil { writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r); return }
            w.WriteHeader(http.StatusNoContent)
        }))

        // Delete
        pr.Delete("/{id}", withTenantAndID("id", func(w http.ResponseWriter, r *http.Request, tenantID, id uuid.UUID) {
            if err := repo.Delete(r.Context(), tenantID, id); err != nil { writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r); return }
            w.WriteHeader(http.StatusNoContent)
        }))
    })
}
