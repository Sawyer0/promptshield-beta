package errors

import (
	"net/http"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// System error codes
const (
	ErrCodeInternalError      types.ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable types.ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeDatabaseError      types.ErrorCode = "DATABASE_ERROR"
	ErrCodeCacheError         types.ErrorCode = "CACHE_ERROR"
	ErrCodeConfigurationError types.ErrorCode = "CONFIGURATION_ERROR"
)

// Repository error codes
const (
	ErrCodeRepositoryError     types.ErrorCode = "REPOSITORY_ERROR"
	ErrCodeDuplicateEntry      types.ErrorCode = "DUPLICATE_ENTRY"
	ErrCodeConcurrencyConflict types.ErrorCode = "CONCURRENCY_CONFLICT"
	ErrCodeTransactionFailed   types.ErrorCode = "TRANSACTION_FAILED"
)

// InternalError returns a generic internal error
func InternalError(message string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeInternalError,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Retryable:  true,
	}
}

// ServiceUnavailable returns an error for unavailable services
func ServiceUnavailable(service string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeServiceUnavailable,
		Message:    "service unavailable",
		HTTPStatus: http.StatusServiceUnavailable,
		Details:    map[string]interface{}{"service": service},
		Retryable:  true,
	}
}

// DatabaseError returns an error for database operations
func DatabaseError(operation string, err error) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeDatabaseError,
		Message:    "database operation failed",
		HTTPStatus: http.StatusInternalServerError,
		Details:    map[string]interface{}{"operation": operation},
		Cause:      err,
		Retryable:  true,
	}
}

// CacheError returns an error for cache operations
func CacheError(operation string, err error) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeCacheError,
		Message:    "cache operation failed",
		HTTPStatus: http.StatusInternalServerError,
		Details:    map[string]interface{}{"operation": operation},
		Cause:      err,
		Retryable:  true,
	}
}

// ConfigurationError returns an error for configuration issues
func ConfigurationError(component string, reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeConfigurationError,
		Message:    "configuration error",
		HTTPStatus: http.StatusInternalServerError,
		Details: map[string]interface{}{
			"component": component,
			"reason":    reason,
		},
	}
}

// RepositoryError returns an error for repository operations
func RepositoryError(entity string, operation string, err error) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeRepositoryError,
		Message:    "repository operation failed",
		HTTPStatus: http.StatusInternalServerError,
		Details: map[string]interface{}{
			"entity":    entity,
			"operation": operation,
		},
		Cause:     err,
		Retryable: true,
	}
}

// DuplicateEntry returns an error for duplicate entries
func DuplicateEntry(entity string, field string, value interface{}) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeDuplicateEntry,
		Message:    "duplicate entry",
		HTTPStatus: http.StatusConflict,
		Details: map[string]interface{}{
			"entity": entity,
			"field":  field,
			"value":  value,
		},
	}
}

// ConcurrencyConflict returns an error for concurrency conflicts
func ConcurrencyConflict(entity string, id string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeConcurrencyConflict,
		Message:    "concurrency conflict detected",
		HTTPStatus: http.StatusConflict,
		Details: map[string]interface{}{
			"entity": entity,
			"id":     id,
		},
		Retryable: true,
	}
}

// TransactionFailed returns an error for failed transactions
func TransactionFailed(reason string, err error) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeTransactionFailed,
		Message:    "transaction failed",
		HTTPStatus: http.StatusInternalServerError,
		Details:    map[string]interface{}{"reason": reason},
		Cause:      err,
		Retryable:  true,
	}
}