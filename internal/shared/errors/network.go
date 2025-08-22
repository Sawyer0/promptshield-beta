package errors

import (
	"net/http"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// Network and I/O error codes
const (
	ErrCodeNetworkError     types.ErrorCode = "NETWORK_ERROR"
	ErrCodeTimeoutError     types.ErrorCode = "TIMEOUT_ERROR"
	ErrCodeConnectionFailed types.ErrorCode = "CONNECTION_FAILED"
	ErrCodeRequestTooLarge  types.ErrorCode = "REQUEST_TOO_LARGE"
)

// NetworkError returns an error for network issues
func NetworkError(operation string, err error) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeNetworkError,
		Message:    "network error",
		HTTPStatus: http.StatusServiceUnavailable,
		Details:    map[string]interface{}{"operation": operation},
		Cause:      err,
		Retryable:  true,
	}
}

// TimeoutError returns an error for timeout issues
func TimeoutError(operation string, timeoutMs int64) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeTimeoutError,
		Message:    "operation timed out",
		HTTPStatus: http.StatusRequestTimeout,
		Details: map[string]interface{}{
			"operation":  operation,
			"timeout_ms": timeoutMs,
		},
		Retryable: true,
	}
}

// ConnectionFailed returns an error for connection failures
func ConnectionFailed(target string, err error) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeConnectionFailed,
		Message:    "connection failed",
		HTTPStatus: http.StatusServiceUnavailable,
		Details:    map[string]interface{}{"target": target},
		Cause:      err,
		Retryable:  true,
	}
}

// RequestTooLarge returns an error for oversized requests
func RequestTooLarge(sizeBytes int64, maxBytes int64) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeRequestTooLarge,
		Message:    "request too large",
		HTTPStatus: http.StatusRequestEntityTooLarge,
		Details: map[string]interface{}{
			"size_bytes": sizeBytes,
			"max_bytes":  maxBytes,
		},
	}
}