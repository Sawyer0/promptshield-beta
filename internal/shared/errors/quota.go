package errors

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Quota error codes
const (
	ErrCodeQuotaExceeded types.ErrorCode = "QUOTA_EXCEEDED"
	ErrCodeQuotaNotFound types.ErrorCode = "QUOTA_NOT_FOUND"
	ErrCodeQuotaInvalid  types.ErrorCode = "QUOTA_INVALID"
)

// QuotaExceeded returns an error for exceeded quotas
func QuotaExceeded(quotaType string, current, limit int64) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeQuotaExceeded,
		Message:    "quota exceeded",
		HTTPStatus: http.StatusTooManyRequests,
		Details: map[string]interface{}{
			"quota_type": quotaType,
			"current":    current,
			"limit":      limit,
		},
		Retryable: true,
	}
}

// QuotaNotFound returns an error for when a quota is not found
func QuotaNotFound(tenantID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeQuotaNotFound,
		Message:    "quota not found",
		HTTPStatus: http.StatusNotFound,
		Details:    map[string]interface{}{"tenant_id": tenantID.String()},
	}
}

// QuotaInvalid returns an error for invalid quota configuration
func QuotaInvalid(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeQuotaInvalid,
		Message:    "quota configuration is invalid",
		HTTPStatus: http.StatusBadRequest,
		Details:    map[string]interface{}{"reason": reason},
	}
}