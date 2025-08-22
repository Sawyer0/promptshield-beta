package errors

import (
	"net/http"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// Validation error codes
const (
	ErrCodeValidationFailed     types.ErrorCode = "VALIDATION_FAILED"
	ErrCodeInvalidRequestFormat types.ErrorCode = "INVALID_REQUEST_FORMAT"
	ErrCodeMissingRequiredField types.ErrorCode = "MISSING_REQUIRED_FIELD"
	ErrCodeInvalidFieldValue    types.ErrorCode = "INVALID_FIELD_VALUE"
)

// ValidationFailed returns an error for validation failures
func ValidationFailed(field string, reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeValidationFailed,
		Message:    "validation failed",
		HTTPStatus: http.StatusBadRequest,
		Details: map[string]interface{}{
			"field":  field,
			"reason": reason,
		},
	}
}

// InvalidRequestFormat returns an error for invalid request format
func InvalidRequestFormat(expected string, received string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeInvalidRequestFormat,
		Message:    "invalid request format",
		HTTPStatus: http.StatusBadRequest,
		Details: map[string]interface{}{
			"expected": expected,
			"received": received,
		},
	}
}

// MissingRequiredField returns an error for missing required fields
func MissingRequiredField(field string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeMissingRequiredField,
		Message:    "missing required field",
		HTTPStatus: http.StatusBadRequest,
		Details:    map[string]interface{}{"field": field},
	}
}

// InvalidFieldValue returns an error for invalid field values
func InvalidFieldValue(field string, value interface{}, expected string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeInvalidFieldValue,
		Message:    "invalid field value",
		HTTPStatus: http.StatusBadRequest,
		Details: map[string]interface{}{
			"field":    field,
			"value":    value,
			"expected": expected,
		},
	}
}