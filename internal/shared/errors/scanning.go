package errors

import (
	"net/http"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// Scanning and enforcement error codes
const (
	ErrCodeScanningTimeout             types.ErrorCode = "SCANNING_TIMEOUT"
	ErrCodeScanningFailed              types.ErrorCode = "SCANNING_FAILED"
	ErrCodePolicyViolationDetected     types.ErrorCode = "POLICY_VIOLATION_DETECTED"
	ErrCodeSemanticProviderUnavailable types.ErrorCode = "SEMANTIC_PROVIDER_UNAVAILABLE"
	ErrCodeRuleCompilationFailed       types.ErrorCode = "RULE_COMPILATION_FAILED"
	ErrCodeStreamLimitExceeded         types.ErrorCode = "STREAM_LIMIT_EXCEEDED"
	ErrCodeSemanticAnalysisFailed      types.ErrorCode = "SEMANTIC_ANALYSIS_FAILED"
	ErrCodeLLMProviderError            types.ErrorCode = "LLM_PROVIDER_ERROR"
	ErrCodeSemanticCacheMiss           types.ErrorCode = "SEMANTIC_CACHE_MISS"
	ErrCodeSemanticConfigError         types.ErrorCode = "SEMANTIC_CONFIG_ERROR"
)

// ScanningTimeout returns an error for scanning timeouts
func ScanningTimeout(timeoutMs int64) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeScanningTimeout,
		Message:    "scanning operation timed out",
		HTTPStatus: http.StatusRequestTimeout,
		Details:    map[string]interface{}{"timeout_ms": timeoutMs},
		Retryable:  true,
	}
}

// ScanningFailed returns an error for scanning failures
func ScanningFailed(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeScanningFailed,
		Message:    "scanning operation failed",
		HTTPStatus: http.StatusInternalServerError,
		Details:    map[string]interface{}{"reason": reason},
		Retryable:  true,
	}
}

// PolicyViolationDetected returns an error for detected policy violations
func PolicyViolationDetected(ruleID string, severity string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodePolicyViolationDetected,
		Message:    "policy violation detected",
		HTTPStatus: http.StatusForbidden,
		Details: map[string]interface{}{
			"rule_id":  ruleID,
			"severity": severity,
		},
	}
}

// SemanticProviderUnavailable returns an error for unavailable semantic providers
func SemanticProviderUnavailable(provider string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeSemanticProviderUnavailable,
		Message:    "semantic provider unavailable",
		HTTPStatus: http.StatusServiceUnavailable,
		Details:    map[string]interface{}{"provider": provider},
		Retryable:  true,
	}
}

// RuleCompilationFailed returns an error for rule compilation failures
func RuleCompilationFailed(ruleID string, reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeRuleCompilationFailed,
		Message:    "failed to compile rule",
		HTTPStatus: http.StatusInternalServerError,
		Details: map[string]interface{}{
			"rule_id": ruleID,
			"reason":  reason,
		},
	}
}

// StreamLimitExceeded returns an error for exceeded stream limits
func StreamLimitExceeded(limit int64) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeStreamLimitExceeded,
		Message:    "stream byte limit exceeded",
		HTTPStatus: http.StatusRequestEntityTooLarge,
		Details:    map[string]interface{}{"limit_bytes": limit},
	}
}

// SemanticAnalysisFailed returns an error for semantic analysis failures
func SemanticAnalysisFailed(provider string, reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeSemanticAnalysisFailed,
		Message:    "semantic analysis failed",
		HTTPStatus: http.StatusInternalServerError,
		Details: map[string]interface{}{
			"provider": provider,
			"reason":   reason,
		},
		Retryable: true,
	}
}

// LLMProviderError returns an error for LLM provider errors
func LLMProviderError(provider string, message string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeLLMProviderError,
		Message:    "LLM provider error",
		HTTPStatus: http.StatusBadGateway,
		Details: map[string]interface{}{
			"provider": provider,
			"message":  message,
		},
		Retryable: true,
	}
}

// SemanticCacheMiss returns an error for semantic cache misses
func SemanticCacheMiss(key string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeSemanticCacheMiss,
		Message:    "semantic cache miss",
		HTTPStatus: http.StatusOK, // Not really an error, just informational
		Details:    map[string]interface{}{"cache_key": key},
	}
}

// SemanticConfigError returns an error for semantic configuration errors
func SemanticConfigError(reason string) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeSemanticConfigError,
		Message:    "semantic configuration error",
		HTTPStatus: http.StatusInternalServerError,
		Details:    map[string]interface{}{"reason": reason},
	}
}