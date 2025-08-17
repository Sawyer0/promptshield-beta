package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// StandardResponse wraps all API responses
type StandardResponse struct {
	Data  interface{}            `json:"data,omitempty"`
	Error *ErrorResponse         `json:"error,omitempty"`
	Meta  map[string]interface{} `json:"meta"`
}

// ErrorResponse represents an error in the API
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// writeJSON writes a JSON response with proper headers
func writeJSON(w http.ResponseWriter, status int, data interface{}, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-PS-API-Version", "1")
	
	correlationID := getCorrelationID(r)
	if correlationID != "" {
		w.Header().Set("X-PS-Correlation-ID", correlationID)
	}
	
	w.WriteHeader(status)
	
	response := StandardResponse{
		Data: data,
		Meta: getMeta(r),
	}
	
	_ = json.NewEncoder(w).Encode(response)
}

// writeErrorJSON writes a structured error response
func writeErrorJSON(w http.ResponseWriter, status int, code, message string, details map[string]interface{}, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-PS-API-Version", "1")
	
	correlationID := getCorrelationID(r)
	if correlationID != "" {
		w.Header().Set("X-PS-Correlation-ID", correlationID)
	}
	
	w.WriteHeader(status)
	
	response := StandardResponse{
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: getMeta(r),
	}
	
	_ = json.NewEncoder(w).Encode(response)
}

// getMeta returns metadata for responses
func getMeta(r *http.Request) map[string]interface{} {
	meta := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1",
	}
	
	if r != nil {
		if correlationID := getCorrelationID(r); correlationID != "" {
			meta["correlation_id"] = correlationID
		}
	}
	
	return meta
}

// getCorrelationID retrieves correlation ID from context
func getCorrelationID(r *http.Request) string {
	if r == nil {
		return ""
	}
	
	if id := r.Context().Value(correlationIDKey); id != nil {
		if strID, ok := id.(string); ok {
			return strID
		}
	}
	
	// Fallback to header
	if id := r.Header.Get("X-PS-Correlation-ID"); id != "" {
		return id
	}
	
	return uuid.New().String()
}

// getTenantID retrieves tenant ID from context
func getTenantID(r *http.Request) string {
	if r == nil {
		return ""
	}
	
	if id := r.Context().Value(tenantIDKey); id != nil {
		if strID, ok := id.(string); ok {
			return strID
		}
	}
	
	// Fallback to header
	return r.Header.Get("X-PS-Tenant-ID")
}