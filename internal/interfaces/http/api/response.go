package api

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "time"
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
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log encoding error but don't fail the request since headers are already written
		// In production, this indicates a serious serialization issue
		logger := getLogger(r)
		logger.Error("Failed to encode JSON response", "error", err, "correlation_id", getCorrelationID(r))
	}
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
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log encoding error but don't fail the request since headers are already written
		// In production, this indicates a serious serialization issue
		logger := getLogger(r)
		logger.Error("Failed to encode error response", "error", err, "correlation_id", getCorrelationID(r))
	}
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

// getCorrelationID is defined in middleware_common.go

// (removed) getTenantID, writeDomainError — no longer needed

// getLogger retrieves logger from request context or returns default
func getLogger(r *http.Request) *slog.Logger {
	if r != nil {
		if logger := r.Context().Value("logger"); logger != nil {
			if l, ok := logger.(*slog.Logger); ok {
				return l
			}
		}
	}
	return slog.Default()
}

// writeError is a convenience function for writing simple errors
func writeError(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	response := StandardResponse{
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "1",
		},
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode error response", "error", err)
	}
}
