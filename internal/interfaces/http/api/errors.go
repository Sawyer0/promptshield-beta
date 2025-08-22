package api

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
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

