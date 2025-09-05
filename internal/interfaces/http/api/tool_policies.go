package api

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "strings"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/promptshield/promptshield/internal/domain"
)

// registerToolPolicyHandlers exposes per-tenant tool policies CRUD under /api/tools/policies
// Data is stored in tenant_settings with key = 'tool_policies' (RLS applies).
func registerToolHandlers(r chi.Router, opt Options) {
    r.Route("/api/tools", func(tr chi.Router) {
        if opt.DB == nil {
            tr.Get("/policies", func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(map[string]any{"policies": []any{}}) })
            tr.Put("/policies", func(w http.ResponseWriter, r *http.Request) { writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r) })
            return
        }

        tr.Get("/policies", func(w http.ResponseWriter, r *http.Request) {
            tenantStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
            if tenantStr == "" { writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r); return }
            tenantID, err := uuid.Parse(tenantStr)
            if err != nil { writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r); return }
            // Read from tenant_settings
            const q = `SELECT value FROM tenant_settings WHERE tenant_id=$1 AND key='tool_policies' LIMIT 1`
            var val sql.NullString
            if err := opt.DB.QueryRowContext(r.Context(), q, tenantID).Scan(&val); err != nil && err != sql.ErrNoRows {
                writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r); return
            }
            if !val.Valid || strings.TrimSpace(val.String) == "" {
                _ = json.NewEncoder(w).Encode(map[string]any{"policies": []any{}})
                return
            }
            // Expect stored JSON to be { policies: [...] } or []
            var out any
            if err := json.Unmarshal([]byte(val.String), &out); err == nil {
                switch t := out.(type) {
                case map[string]any:
                    _ = json.NewEncoder(w).Encode(t)
                case []any:
                    _ = json.NewEncoder(w).Encode(map[string]any{"policies": t})
                default:
                    _ = json.NewEncoder(w).Encode(map[string]any{"policies": []any{}})
                }
                return
            }
            _ = json.NewEncoder(w).Encode(map[string]any{"policies": []any{}})
        })

        tr.Put("/policies", func(w http.ResponseWriter, r *http.Request) {
            tenantStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
            if tenantStr == "" { writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID required", nil, r); return }
            tenantID, err := uuid.Parse(tenantStr)
            if err != nil { writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "bad tenant id", nil, r); return }
            // Accept either { policies: [...] } or plain []
            var raw any
            if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
                writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json", nil, r); return
            }
            var payload map[string]any
            if m, ok := raw.(map[string]any); ok {
                payload = m
            } else if arr, ok := raw.([]any); ok {
                payload = map[string]any{"policies": arr}
            } else {
                payload = map[string]any{"policies": []any{}}
            }
            // Minimal validation: coerce to canonical form
            b, _ := json.Marshal(payload)
            const up = `INSERT INTO tenant_settings (tenant_id, key, value)
                        VALUES ($1,'tool_policies', $2::jsonb)
                        ON CONFLICT (tenant_id, key)
                        DO UPDATE SET value=EXCLUDED.value, updated_at=NOW()`
            if _, err := opt.DB.ExecContext(r.Context(), up, tenantID, string(b)); err != nil {
                writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, r); return
            }
            // Optionally emit audit event
            if opt.AuditRepository != nil {
                _ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
                    Action:     "tool.policies.updated",
                    ObjectType: "tool_policies",
                    ObjectID:   tenantID,
                    Metadata:   json.RawMessage(b),
                })
            }
            w.WriteHeader(http.StatusNoContent)
        })
    })
}
