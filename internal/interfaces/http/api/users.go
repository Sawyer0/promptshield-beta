package api

import (
    "encoding/json"
    "net/http"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
)

// registerUserHandlers registers user-related endpoints
func registerUserHandlers(r chi.Router, opt Options) {
    // Sync the authenticated user into platform_users (upsert)
    r.Post("/users/sync", userSyncHandler(opt))
}

func userSyncHandler(opt Options) http.HandlerFunc {
	type reqBody struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
return func(w http.ResponseWriter, r *http.Request) {
        if ok, reason := authorizePDP(r, "user.sync", "user", strings.TrimSpace(r.Header.Get("X-PS-User-ID")), nil, true); !ok {
            writeErrorJSON(w, http.StatusForbidden, "PDP_DENY", "not authorized: "+reason, nil, r); return
        }
		if opt.DB == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "database not configured", nil, r)
			return
		}
		userID := strings.TrimSpace(r.Header.Get("X-PS-User-ID"))
		if userID == "" {
			writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context", nil, r)
			return
		}
		var in reqBody
		_ = json.NewDecoder(r.Body).Decode(&in)
		email := strings.TrimSpace(in.Email)
		first := strings.TrimSpace(in.FirstName)
		last := strings.TrimSpace(in.LastName)

		// Upsert platform user
		_, err := opt.DB.ExecContext(r.Context(), `
            INSERT INTO platform_users (id, email, first_name, last_name, created_at, updated_at)
            VALUES ($1,$2,$3,$4,NOW(),NOW())
            ON CONFLICT (id) DO UPDATE SET
                email=EXCLUDED.email,
                first_name=EXCLUDED.first_name,
                last_name=EXCLUDED.last_name,
                updated_at=NOW()
        `, userID, email, first, last)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to upsert user", map[string]any{"error": err.Error()}, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": userID, "email": email, "first_name": first, "last_name": last, "synced_at": time.Now().UTC()}, r)
	}
}
