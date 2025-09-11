package api

import (
    "fmt"
    "net/http"
    "runtime"
    "time"
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

// Error code constants for consistency
const (
	// Authentication and Authorization errors
	ErrorCodeUnauthorized        = "UNAUTHORIZED"
	ErrorCodeForbidden          = "FORBIDDEN"
	ErrorCodeJWTMissing         = "JWT_MISSING"
	ErrorCodeJWTInvalid         = "JWT_INVALID"
	ErrorCodeJWTExpired         = "JWT_EXPIRED"
	ErrorCodeJWTNotYetValid     = "JWT_NOT_YET_VALID"
	ErrorCodeJWTInvalidIssuer   = "JWT_INVALID_ISSUER"
	ErrorCodeJWTInvalidAudience = "JWT_INVALID_AUDIENCE"
	ErrorCodeJWTUnsupportedAlg  = "JWT_UNSUPPORTED_ALG"
	ErrorCodeJWTSignatureInvalid = "JWT_SIGNATURE_INVALID"
	ErrorCodeJWTConfigurationError = "JWT_CONFIGURATION_ERROR"
	
	// Tenant errors
	ErrorCodeTenantMissing         = "TENANT_MISSING"
	ErrorCodeTenantInvalidFormat   = "TENANT_INVALID_FORMAT"
	ErrorCodeTenantNotFound        = "TENANT_NOT_FOUND"
	ErrorCodeTenantInactive        = "TENANT_INACTIVE"
	ErrorCodeTenantAccessDenied    = "TENANT_ACCESS_DENIED"
	ErrorCodeTenantContextFailed   = "TENANT_CONTEXT_ERROR"
	ErrorCodeTenantValidationError = "TENANT_VALIDATION_ERROR"
	
	// Request validation errors
	ErrorCodeInvalidRequest   = "INVALID_REQUEST"
	ErrorCodeInvalidArgument  = "INVALID_ARGUMENT"
	ErrorCodeMissingParameter = "MISSING_PARAMETER"
	ErrorCodeInvalidFormat    = "INVALID_FORMAT"
	
	// Resource errors
	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeAlreadyExists = "ALREADY_EXISTS"
	ErrorCodeConflict      = "CONFLICT"
	
	// Rate limiting and quota errors
	ErrorCodeRateLimited     = "RATE_LIMITED"
	ErrorCodeQuotaExceeded   = "QUOTA_EXCEEDED"
	ErrorCodeResourceExhausted = "RESOURCE_EXHAUSTED"
	
	// System errors
	ErrorCodeInternalError      = "INTERNAL_ERROR"
	ErrorCodeNotImplemented     = "NOT_IMPLEMENTED"
	ErrorCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrorCodeProviderError      = "PROVIDER_ERROR"
	ErrorCodeConfigurationError = "CONFIGURATION_ERROR"
	
	// Policy and compliance errors
	ErrorCodePolicyViolation = "POLICY_VIOLATION"
	ErrorCodeComplianceError = "COMPLIANCE_ERROR"
)

// Common error constructors with correlation ID support
var (
	ErrInvalidRequest = func(msg string, details map[string]interface{}) *APIError {
		return &APIError{
			Code:       ErrorCodeInvalidRequest,
			Message:    msg,
			Details:    details,
			StatusCode: http.StatusBadRequest,
		}
	}
	
	ErrInvalidArgument = func(field, msg string) *APIError {
		return &APIError{
			Code:    ErrorCodeInvalidArgument,
			Message: fmt.Sprintf("Invalid %s: %s", field, msg),
			Details: map[string]interface{}{"field": field},
			StatusCode: http.StatusBadRequest,
		}
	}
	
	ErrMissingParameter = func(param string) *APIError {
		return &APIError{
			Code:    ErrorCodeMissingParameter,
			Message: fmt.Sprintf("Missing required parameter: %s", param),
			Details: map[string]interface{}{"parameter": param},
			StatusCode: http.StatusBadRequest,
		}
	}
	
	ErrNotFound = func(resource, id string) *APIError {
		return &APIError{
			Code:    ErrorCodeNotFound,
			Message: fmt.Sprintf("%s not found", resource),
			Details: map[string]interface{}{"resource": resource, "id": id},
			StatusCode: http.StatusNotFound,
		}
	}
	
	ErrAlreadyExists = func(resource, id string) *APIError {
		return &APIError{
			Code:    ErrorCodeAlreadyExists,
			Message: fmt.Sprintf("%s already exists", resource),
			Details: map[string]interface{}{"resource": resource, "id": id},
			StatusCode: http.StatusConflict,
		}
	}
	
	ErrUnauthorized = func(msg string, details map[string]interface{}) *APIError {
		return &APIError{
			Code:       ErrorCodeUnauthorized,
			Message:    msg,
			Details:    details,
			StatusCode: http.StatusUnauthorized,
		}
	}
	
	ErrForbidden = func(msg string, details map[string]interface{}) *APIError {
		return &APIError{
			Code:       ErrorCodeForbidden,
			Message:    msg,
			Details:    details,
			StatusCode: http.StatusForbidden,
		}
	}
	
	ErrConflict = func(resource, msg string) *APIError {
		return &APIError{
			Code:    ErrorCodeConflict,
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
			Code:       ErrorCodeRateLimited,
			Message:    msg,
			Details:    details,
			StatusCode: http.StatusTooManyRequests,
		}
	}
	
	ErrQuotaExceeded = func(resource, limit string) *APIError {
		return &APIError{
			Code:    ErrorCodeQuotaExceeded,
			Message: fmt.Sprintf("Quota exceeded for %s", resource),
			Details: map[string]interface{}{"resource": resource, "limit": limit},
			StatusCode: http.StatusTooManyRequests,
		}
	}
	
	ErrInternalError = func(msg string) *APIError {
		return &APIError{
			Code:       ErrorCodeInternalError,
			Message:    msg,
			StatusCode: http.StatusInternalServerError,
		}
	}
	
	ErrConfigurationError = func(component, msg string) *APIError {
		return &APIError{
			Code:    ErrorCodeConfigurationError,
			Message: fmt.Sprintf("Configuration error in %s: %s", component, msg),
			Details: map[string]interface{}{"component": component},
			StatusCode: http.StatusInternalServerError,
		}
	}
	
	ErrNotImplemented = func(feature string) *APIError {
		return &APIError{
			Code:    ErrorCodeNotImplemented,
			Message: fmt.Sprintf("%s not implemented", feature),
			Details: map[string]interface{}{"feature": feature},
			StatusCode: http.StatusNotImplemented,
		}
	}
	
	ErrServiceUnavailable = func(service, msg string) *APIError {
		return &APIError{
			Code:    ErrorCodeServiceUnavailable,
			Message: fmt.Sprintf("%s unavailable: %s", service, msg),
			Details: map[string]interface{}{"service": service},
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	
	ErrProviderError = func(provider, msg string) *APIError {
		return &APIError{
			Code:    ErrorCodeProviderError,
			Message: fmt.Sprintf("Provider %s error: %s", provider, msg),
			Details: map[string]interface{}{"provider": provider},
			StatusCode: http.StatusBadGateway,
		}
	}
	
	ErrPolicyViolation = func(rule, msg string) *APIError {
		return &APIError{
			Code:    ErrorCodePolicyViolation,
			Message: fmt.Sprintf("Policy violation: %s", msg),
			Details: map[string]interface{}{"rule": rule},
			StatusCode: http.StatusForbidden,
		}
	}
)

// WriteAPIError writes an APIError with proper correlation ID and logging
func WriteAPIError(w http.ResponseWriter, r *http.Request, apiErr *APIError) {
	correlationID := getCorrelationID(r)
	
	// Ensure details map exists and add correlation ID
	if apiErr.Details == nil {
		apiErr.Details = make(map[string]interface{})
	}
	apiErr.Details["correlation_id"] = correlationID
	apiErr.Details["timestamp"] = fmt.Sprintf("%d", time.Now().Unix())
	apiErr.Details["path"] = r.URL.Path
	apiErr.Details["method"] = r.Method
	
	// Log the error with structured logging
	logger := getLogger(r)
	logger.Error("API error",
		"error_code", apiErr.Code,
		"message", apiErr.Message,
		"status_code", apiErr.StatusCode,
		"correlation_id", correlationID,
		"path", r.URL.Path,
		"method", r.Method,
		"details", apiErr.Details,
	)
	
	writeErrorJSON(w, apiErr.StatusCode, apiErr.Code, apiErr.Message, apiErr.Details, r)
}

// NewAPIErrorWithCorrelation creates an APIError with correlation ID pre-populated
func NewAPIErrorWithCorrelation(r *http.Request, code, message string, statusCode int, details map[string]interface{}) *APIError {
	if details == nil {
		details = make(map[string]interface{})
	}
	
	correlationID := getCorrelationID(r)
	details["correlation_id"] = correlationID
	details["timestamp"] = fmt.Sprintf("%d", time.Now().Unix())
	details["path"] = r.URL.Path
	details["method"] = r.Method
	
	return &APIError{
		Code:       code,
		Message:    message,
		Details:    details,
		StatusCode: statusCode,
	}
}

// errorRecoveryMiddleware recovers from panics and converts them to structured errors
func errorRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				buf := make([]byte, 1024*4)
				n := runtime.Stack(buf, false)
				stackTrace := string(buf[:n])
				
				correlationID := getCorrelationID(r)
				
				// Log panic with structured logging
				logger := getLogger(r)
				logger.Error("Panic recovered",
					"panic", err,
					"correlation_id", correlationID,
					"path", r.URL.Path,
					"method", r.Method,
					"stack_trace", stackTrace,
				)
				
				// Return a generic internal server error
				apiErr := ErrInternalError("Internal server error occurred")
				apiErr.Details = map[string]interface{}{
					"correlation_id": correlationID,
					"recoverable":    true,
					"timestamp":      fmt.Sprintf("%d", time.Now().Unix()),
					"path":          r.URL.Path,
					"method":        r.Method,
				}
				
				WriteAPIError(w, r, apiErr)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}
