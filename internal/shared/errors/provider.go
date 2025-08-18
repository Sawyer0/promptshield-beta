package errors

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Provider error codes
const (
	ErrCodeProviderKeyNotFound        types.ErrorCode = "PROVIDER_KEY_NOT_FOUND"
	ErrCodeProviderKeyRevoked         types.ErrorCode = "PROVIDER_KEY_REVOKED"
	ErrCodeProviderKeyExpired         types.ErrorCode = "PROVIDER_KEY_EXPIRED"
	ErrCodeProviderKeyInvalid         types.ErrorCode = "PROVIDER_KEY_INVALID"
	ErrCodeProviderKeyDecryption      types.ErrorCode = "PROVIDER_KEY_DECRYPTION_FAILED"
	ErrCodeProviderRateLimited        types.ErrorCode = "PROVIDER_RATE_LIMITED"
	ErrCodeProviderQuotaExhausted     types.ErrorCode = "PROVIDER_QUOTA_EXHAUSTED"
	ErrCodeProviderInvalidRequest     types.ErrorCode = "PROVIDER_INVALID_REQUEST"
	ErrCodeProviderUnauthorized       types.ErrorCode = "PROVIDER_UNAUTHORIZED"
	ErrCodeProviderServiceUnavailable types.ErrorCode = "PROVIDER_SERVICE_UNAVAILABLE"
	ErrCodeProviderInternalError      types.ErrorCode = "PROVIDER_INTERNAL_ERROR"
)

// ProviderKeyNotFound returns an error for when a provider key is not found
func ProviderKeyNotFound(keyID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderKeyNotFound,
		Message:    "provider key not found",
		HTTPStatus: http.StatusNotFound,
		Details:    map[string]interface{}{"key_id": keyID.String()},
	}
}

// ProviderKeyRevoked returns an error for revoked provider keys
func ProviderKeyRevoked(keyID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderKeyRevoked,
		Message:    "provider key is revoked",
		HTTPStatus: http.StatusForbidden,
		Details:    map[string]interface{}{"key_id": keyID.String()},
	}
}

// ProviderKeyExpired returns an error for expired provider keys
func ProviderKeyExpired(keyID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderKeyExpired,
		Message:    "provider key has expired",
		HTTPStatus: http.StatusForbidden,
		Details:    map[string]interface{}{"key_id": keyID.String()},
	}
}

// ProviderKeyInvalid returns an error for invalid provider keys
func ProviderKeyInvalid(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderKeyInvalid,
		Message:    "provider key is invalid",
		HTTPStatus: http.StatusBadRequest,
		Details:    map[string]interface{}{"reason": reason},
	}
}

// ProviderKeyDecryptionFailed returns an error for provider key decryption failure
func ProviderKeyDecryptionFailed(keyID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderKeyDecryption,
		Message:    "failed to decrypt provider key",
		HTTPStatus: http.StatusInternalServerError,
		Details:    map[string]interface{}{"key_id": keyID.String()},
	}
}

// ProviderRateLimited returns an error for rate-limited provider requests
func ProviderRateLimited(provider string, retryAfterSeconds int) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderRateLimited,
		Message:    "provider rate limited",
		HTTPStatus: http.StatusTooManyRequests,
		Details: map[string]interface{}{
			"provider":           provider,
			"retry_after_seconds": retryAfterSeconds,
		},
		Retryable: true,
	}
}

// ProviderQuotaExhausted returns an error for exhausted provider quota
func ProviderQuotaExhausted(provider string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderQuotaExhausted,
		Message:    "provider quota exhausted",
		HTTPStatus: http.StatusPaymentRequired,
		Details:    map[string]interface{}{"provider": provider},
	}
}

// ProviderInvalidRequest returns an error for invalid provider requests
func ProviderInvalidRequest(provider string, reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderInvalidRequest,
		Message:    "provider rejected request",
		HTTPStatus: http.StatusBadRequest,
		Details: map[string]interface{}{
			"provider": provider,
			"reason":   reason,
		},
	}
}

// ProviderUnauthorized returns an error for unauthorized provider access
func ProviderUnauthorized(provider string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderUnauthorized,
		Message:    "provider authentication failed",
		HTTPStatus: http.StatusUnauthorized,
		Details:    map[string]interface{}{"provider": provider},
	}
}

// ProviderServiceUnavailable returns an error for unavailable provider service
func ProviderServiceUnavailable(provider string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderServiceUnavailable,
		Message:    "provider service unavailable",
		HTTPStatus: http.StatusServiceUnavailable,
		Details:    map[string]interface{}{"provider": provider},
		Retryable:  true,
	}
}

// ProviderInternalError returns an error for provider internal errors
func ProviderInternalError(provider string, message string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProviderInternalError,
		Message:    "provider internal error",
		HTTPStatus: http.StatusBadGateway,
		Details: map[string]interface{}{
			"provider": provider,
			"message":  message,
		},
		Retryable: true,
	}
}