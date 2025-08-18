package errors

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Policy error codes
const (
	ErrCodePolicyNotFound            types.ErrorCode = "POLICY_NOT_FOUND"
	ErrCodePolicyValidationFailed    types.ErrorCode = "POLICY_VALIDATION_FAILED"
	ErrCodePolicyAssignmentNotFound  types.ErrorCode = "POLICY_ASSIGNMENT_NOT_FOUND"
	ErrCodePolicyConflict            types.ErrorCode = "POLICY_CONFLICT"
	ErrCodePolicyInvalidFormat       types.ErrorCode = "POLICY_INVALID_FORMAT"
)

// PolicyNotFound returns an error for when a policy is not found
func PolicyNotFound(policyID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodePolicyNotFound,
		Message:    "policy not found",
		HTTPStatus: http.StatusNotFound,
		Details:    map[string]interface{}{"policy_id": policyID.String()},
	}
}

// PolicyValidationFailed returns an error for policy validation failure
func PolicyValidationFailed(details map[string]interface{}) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodePolicyValidationFailed,
		Message:    "policy validation failed",
		HTTPStatus: http.StatusBadRequest,
		Details:    details,
	}
}

// PolicyAssignmentNotFound returns an error for when a policy assignment is not found
func PolicyAssignmentNotFound(assignmentID uuid.UUID) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodePolicyAssignmentNotFound,
		Message:    "policy assignment not found",
		HTTPStatus: http.StatusNotFound,
		Details:    map[string]interface{}{"assignment_id": assignmentID.String()},
	}
}

// PolicyConflict returns an error for policy conflicts
func PolicyConflict(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodePolicyConflict,
		Message:    "policy conflict detected",
		HTTPStatus: http.StatusConflict,
		Details:    map[string]interface{}{"reason": reason},
	}
}

// PolicyInvalidFormat returns an error for invalid policy format
func PolicyInvalidFormat(format, expected string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodePolicyInvalidFormat,
		Message:    "policy format is invalid",
		HTTPStatus: http.StatusBadRequest,
		Details: map[string]interface{}{
			"format":   format,
			"expected": expected,
		},
	}
}