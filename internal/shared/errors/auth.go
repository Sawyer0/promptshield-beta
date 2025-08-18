package errors

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Authentication and authorization error codes
const (
	ErrCodeUnauthorized              types.ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden                 types.ErrorCode = "FORBIDDEN"
	ErrCodeInvalidCredentials        types.ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired              types.ErrorCode = "TOKEN_EXPIRED"
	ErrCodeInsufficientPermissions   types.ErrorCode = "INSUFFICIENT_PERMISSIONS"
	ErrCodeAPITokenNotFound          types.ErrorCode = "API_TOKEN_NOT_FOUND"
	ErrCodeAPITokenRevoked           types.ErrorCode = "API_TOKEN_REVOKED"
	ErrCodeAPITokenExpired           types.ErrorCode = "API_TOKEN_EXPIRED"
	ErrCodeAPITokenInvalid           types.ErrorCode = "API_TOKEN_INVALID"
	ErrCodeAPITokenInsufficientScope types.ErrorCode = "API_TOKEN_INSUFFICIENT_SCOPE"
)

// Unauthorized returns an error for unauthorized access
func Unauthorized(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeUnauthorized,
		Message:    "unauthorized",
		HTTPStatus: http.StatusUnauthorized,
		Details:    map[string]interface{}{"reason": reason},
	}
}

// Forbidden returns an error for forbidden access
func Forbidden(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeForbidden,
		Message:    "forbidden",
		HTTPStatus: http.StatusForbidden,
		Details:    map[string]interface{}{"reason": reason},
	}
}

// InvalidCredentials returns an error for invalid credentials
func InvalidCredentials() *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeInvalidCredentials,
		Message:    "invalid credentials",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// TokenExpired returns an error for expired tokens
func TokenExpired(tokenType string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeTokenExpired,
		Message:    "token has expired",
		HTTPStatus: http.StatusUnauthorized,
		Details:    map[string]interface{}{"token_type": tokenType},
	}
}

// InsufficientPermissions returns an error for insufficient permissions
func InsufficientPermissions(required string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeInsufficientPermissions,
		Message:    "insufficient permissions",
		HTTPStatus: http.StatusForbidden,
		Details:    map[string]interface{}{"required": required},
	}
}

// APITokenNotFound returns an error for when an API token is not found
func APITokenNotFound(tokenID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeAPITokenNotFound,
		Message:    "API token not found",
		HTTPStatus: http.StatusNotFound,
		Details:    map[string]interface{}{"token_id": tokenID.String()},
	}
}

// APITokenRevoked returns an error for revoked API tokens
func APITokenRevoked(tokenID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeAPITokenRevoked,
		Message:    "API token is revoked",
		HTTPStatus: http.StatusUnauthorized,
		Details:    map[string]interface{}{"token_id": tokenID.String()},
	}
}

// APITokenExpired returns an error for expired API tokens
func APITokenExpired(tokenID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeAPITokenExpired,
		Message:    "API token has expired",
		HTTPStatus: http.StatusUnauthorized,
		Details:    map[string]interface{}{"token_id": tokenID.String()},
	}
}

// APITokenInvalid returns an error for invalid API tokens
func APITokenInvalid(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeAPITokenInvalid,
		Message:    "API token is invalid",
		HTTPStatus: http.StatusUnauthorized,
		Details:    map[string]interface{}{"reason": reason},
	}
}

// APITokenInsufficientScope returns an error for insufficient API token scope
func APITokenInsufficientScope(required []string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeAPITokenInsufficientScope,
		Message:    "API token has insufficient scope",
		HTTPStatus: http.StatusForbidden,
		Details:    map[string]interface{}{"required_scopes": required},
	}
}