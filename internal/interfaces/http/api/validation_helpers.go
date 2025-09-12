package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ValidationHelpers provides common validation patterns to reduce code duplication

// parseTenantID extracts and validates tenant ID from request headers
func parseTenantID(r *http.Request) (uuid.UUID, error) {
	tenantStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
	return uuid.Parse(tenantStr)
}

// parseUUIDParam extracts and validates a UUID from URL parameters
func parseUUIDParam(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, param))
}

// requireTenantID validates tenant ID and writes error if invalid
func requireTenantID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing or invalid tenant id", nil)
		return uuid.Nil, false
	}
	return tenantID, true
}

// requireUUIDParam validates UUID parameter and writes error if invalid
func requireUUIDParam(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := parseUUIDParam(r, param)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid "+param+" format", nil)
		return uuid.Nil, false
	}
	return id, true
}

// TenantAndIDHandler is a helper for handlers that need both tenant ID and a UUID parameter
type TenantAndIDHandler func(w http.ResponseWriter, r *http.Request, tenantID, id uuid.UUID)

// withTenantAndID wraps a handler to automatically validate tenant ID and UUID parameter
func withTenantAndID(param string, handler TenantAndIDHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireTenantID(w, r)
		if !ok {
			return
		}

		id, ok := requireUUIDParam(w, r, param)
		if !ok {
			return
		}

		handler(w, r, tenantID, id)
	}
}

// TenantHandler is a helper for handlers that need only tenant ID
type TenantHandler func(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID)

// withTenant wraps a handler to automatically validate tenant ID
func withTenant(handler TenantHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireTenantID(w, r)
		if !ok {
			return
		}

		handler(w, r, tenantID)
	}
}

// getTenantID is a helper for inline tenant validation in complex handlers
// Returns the tenant ID and true if valid, or writes error and returns false
func getTenantID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return requireTenantID(w, r)
}

// validateTenantIDString validates a tenant ID string and writes error if invalid
// Returns the parsed UUID and true if valid, or writes error and returns false
func validateTenantIDString(w http.ResponseWriter, tenantIDStr string) (uuid.UUID, bool) {
	if tenantIDStr == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing tenant id", map[string]any{"hint": "pass X-PS-Tenant-ID"})
		return uuid.Nil, false
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid tenant id", nil)
		return uuid.Nil, false
	}

	return tenantID, true
}
