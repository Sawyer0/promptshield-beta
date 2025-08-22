package errors

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Tenant error codes
const (
	ErrCodeTenantNotFound      types.ErrorCode = "TENANT_NOT_FOUND"
	ErrCodeTenantSuspended     types.ErrorCode = "TENANT_SUSPENDED"
	ErrCodeTenantAlreadyExists types.ErrorCode = "TENANT_ALREADY_EXISTS"
	ErrCodeTenantInvalid       types.ErrorCode = "TENANT_INVALID"
)

// TenantNotFound returns an error for when a tenant is not found
func TenantNotFound(tenantID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeTenantNotFound,
		Message:    "tenant not found",
		HTTPStatus: http.StatusNotFound,
		Details:    map[string]interface{}{"tenant_id": tenantID.String()},
	}
}

// TenantSuspended returns an error for when a tenant is suspended
func TenantSuspended(tenantID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeTenantSuspended,
		Message:    "tenant is suspended",
		HTTPStatus: http.StatusForbidden,
		Details:    map[string]interface{}{"tenant_id": tenantID.String()},
	}
}

// TenantAlreadyExists returns an error for when a tenant already exists
func TenantAlreadyExists(name string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeTenantAlreadyExists,
		Message:    "tenant already exists",
		HTTPStatus: http.StatusConflict,
		Details:    map[string]interface{}{"name": name},
	}
}

// TenantInvalid returns an error for invalid tenant data
func TenantInvalid(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeTenantInvalid,
		Message:    "tenant data is invalid",
		HTTPStatus: http.StatusBadRequest,
		Details:    map[string]interface{}{"reason": reason},
	}
}