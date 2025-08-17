package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
)

// APIError represents a structured API error
type APIError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error codes and constructors
var (
	ErrInvalidRequest = func(msg string, details map[string]interface{}) *APIError {
		return &APIError{
			Code:       "INVALID_REQUEST",
			Message:    msg,
			Details:    details,
			StatusCode: http.StatusBadRequest,
		}
	}
	
	ErrInvalidArgument = func(field, msg string) *APIError {
		return &APIError{
			Code:    "INVALID_ARGUMENT",
			Message: fmt.Sprintf("Invalid %s: %s", field, msg),
			Details: map[string]interface{}{"field": field},
			StatusCode: http.StatusBadRequest,
		}
	}
	
	ErrNotFound = func(resource, id string) *APIError {
		return &APIError{
			Code:    "NOT_FOUND",
			Message: fmt.Sprintf("%s not found", resource),
			Details: map[string]interface{}{"resource": resource, "id": id},
			StatusCode: http.StatusNotFound,
		}
	}
	
	ErrUnauthorized = func(msg string) *APIError {
		return &APIError{
			Code:       "UNAUTHORIZED",
			Message:    msg,
			StatusCode: http.StatusUnauthorized,
		}
	}
	
	ErrForbidden = func(msg string, details map[string]interface{}) *APIError {
		return &APIError{
			Code:       "FORBIDDEN",
			Message:    msg,
			Details:    details,
			StatusCode: http.StatusForbidden,
		}
	}
	
	ErrConflict = func(resource, msg string) *APIError {
		return &APIError{
			Code:    "CONFLICT",
			Message: fmt.Sprintf("%s conflict: %s", resource, msg),
			Details: map[string]interface{}{"resource": resource},
			StatusCode: http.StatusConflict,
		}
	}
	
	ErrRateLimited = func(msg string, retryAfter int) *APIError {
		details := map[string]interface{}{}
		if retryAfter > 0 {
			details["retry_after_seconds"] = retryAfter
		}
		return &APIError{
			Code:       "RATE_LIMITED",
			Message:    msg,
			Details:    details,
			StatusCode: http.StatusTooManyRequests,
		}
	}
	
	ErrInternalError = func(msg string) *APIError {
		return &APIError{
			Code:       "INTERNAL_ERROR",
			Message:    msg,
			StatusCode: http.StatusInternalServerError,
		}
	}
	
	ErrNotImplemented = func(feature string) *APIError {
		return &APIError{
			Code:    "NOT_IMPLEMENTED",
			Message: fmt.Sprintf("%s not implemented", feature),
			Details: map[string]interface{}{"feature": feature},
			StatusCode: http.StatusNotImplemented,
		}
	}
	
	ErrServiceUnavailable = func(service, msg string) *APIError {
		return &APIError{
			Code:    "SERVICE_UNAVAILABLE",
			Message: fmt.Sprintf("%s unavailable: %s", service, msg),
			Details: map[string]interface{}{"service": service},
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	
	ErrProviderError = func(provider, msg string) *APIError {
		return &APIError{
			Code:    "PROVIDER_ERROR",
			Message: fmt.Sprintf("Provider %s error: %s", provider, msg),
			Details: map[string]interface{}{"provider": provider},
			StatusCode: http.StatusBadGateway,
		}
	}
	
	ErrPolicyViolation = func(rule, msg string) *APIError {
		return &APIError{
			Code:    "POLICY_VIOLATION",
			Message: fmt.Sprintf("Policy violation: %s", msg),
			Details: map[string]interface{}{"rule": rule},
			StatusCode: http.StatusForbidden,
		}
	}
)

// errorRecoveryMiddleware recovers from panics and converts them to structured errors
func errorRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				buf := make([]byte, 1024*4)
				n := runtime.Stack(buf, false)
				_ = string(buf[:n]) // Use stack trace for logging (placeholder)
				
				// Log panic to stderr for now (in production, use proper structured logging)
				fmt.Fprintf(os.Stderr, "PANIC RECOVERED: %v at %s %s (correlation_id: %s)\n", 
					err, r.Method, r.URL.Path, getCorrelationID(r))
				
				// Return a generic internal server error
				apiErr := ErrInternalError("Internal server error occurred")
				writeErrorJSON(w, apiErr.StatusCode, apiErr.Code, apiErr.Message, 
					map[string]interface{}{
						"correlation_id": getCorrelationID(r),
						"recoverable": true,
					}, r)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

// validateTenantAccess validates that the user has access to the specified tenant
func validateTenantAccess(r *http.Request, tenantID string) error {
	// Get tenant from context (set by auth middleware)
	contextTenantID := getTenantID(r)
	
	// If no tenant in URL, use context tenant
	if tenantID == "" {
		if contextTenantID == "" {
			return ErrInvalidArgument("tenant_id", "required")
		}
		return nil
	}
	
	// If tenant specified in URL, ensure it matches context (for user tokens)
	// Admin tokens can access any tenant
	if contextTenantID != "" && contextTenantID != tenantID {
		// Check if this is an admin token (has special permission)
		if !isAdminToken(r) {
			return ErrForbidden("Access denied to tenant", map[string]interface{}{
				"requested_tenant": tenantID,
				"allowed_tenant":   contextTenantID,
			})
		}
	}
	
	return nil
}

// isAdminToken checks if the request has admin token privileges
func isAdminToken(r *http.Request) bool {
	// This would check the token claims or permissions
	// For now, assume admin if admin auth middleware passed
	return r.Context().Value("is_admin") == true
}

// validateJSON validates that a request body contains valid JSON
func validateJSON(r *http.Request, target interface{}) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return ErrInvalidRequest("Content-Type must be application/json", nil)
	}
	
	if r.ContentLength == 0 {
		return ErrInvalidRequest("Request body is required", nil)
	}
	
	if r.ContentLength > 1024*1024*10 { // 10MB limit
		return ErrInvalidRequest("Request body too large", map[string]interface{}{
			"max_size_mb": 10,
		})
	}
	
	// The actual JSON decoding would be done by the caller
	// This just validates preconditions
	return nil
}

// validateUUID validates that a string is a valid UUID
func validateUUID(value, field string) error {
	if value == "" {
		return ErrInvalidArgument(field, "required")
	}
	
	// Simple UUID format validation
	if len(value) != 36 || 
		!strings.Contains(value, "-") ||
		strings.Count(value, "-") != 4 {
		return ErrInvalidArgument(field, "must be a valid UUID")
	}
	
	return nil
}

// validatePagination validates pagination parameters
func validatePagination(limit, offset int) error {
	if limit < 0 {
		return ErrInvalidArgument("limit", "must be non-negative")
	}
	
	if limit > 1000 {
		return ErrInvalidArgument("limit", "must be <= 1000")
	}
	
	if offset < 0 {
		return ErrInvalidArgument("offset", "must be non-negative")
	}
	
	return nil
}

// validateProvider validates that a provider is supported
func validateProvider(provider string) error {
	if provider == "" {
		return ErrInvalidArgument("provider", "required")
	}
	
	validProviders := map[string]bool{
		"openai":       true,
		"anthropic":    true,
		"azure_openai": true,
		"custom":       true,
	}
	
	if !validProviders[strings.ToLower(provider)] {
		return ErrInvalidArgument("provider", fmt.Sprintf("unsupported provider: %s", provider))
	}
	
	return nil
}

// handleAPIError converts various error types to appropriate HTTP responses
func handleAPIError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	
	// Handle APIError directly
	if apiErr, ok := err.(*APIError); ok {
		writeErrorJSON(w, apiErr.StatusCode, apiErr.Code, apiErr.Message, apiErr.Details, r)
		return
	}
	
	// Handle context errors
	if err == context.Canceled {
		writeErrorJSON(w, http.StatusRequestTimeout, "REQUEST_CANCELED", 
			"Request was canceled", nil, r)
		return
	}
	
	if err == context.DeadlineExceeded {
		writeErrorJSON(w, http.StatusRequestTimeout, "REQUEST_TIMEOUT", 
			"Request timed out", nil, r)
		return
	}
	
	// Handle common error patterns by message
	errMsg := strings.ToLower(err.Error())
	
	if strings.Contains(errMsg, "not found") {
		writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", 
			err.Error(), nil, r)
		return
	}
	
	if strings.Contains(errMsg, "unauthorized") {
		writeErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", 
			err.Error(), nil, r)
		return
	}
	
	if strings.Contains(errMsg, "forbidden") {
		writeErrorJSON(w, http.StatusForbidden, "FORBIDDEN", 
			err.Error(), nil, r)
		return
	}
	
	if strings.Contains(errMsg, "conflict") || strings.Contains(errMsg, "already exists") {
		writeErrorJSON(w, http.StatusConflict, "CONFLICT", 
			err.Error(), nil, r)
		return
	}
	
	// Default to internal server error
	writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
		"An internal error occurred", map[string]interface{}{
			"original_error": err.Error(),
		}, r)
}