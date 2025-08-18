package errors

import (
	"errors"
	"net/http"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// Utility error codes
const (
	ErrCodePoolClosed         types.ErrorCode = "POOL_CLOSED"
	ErrCodeNoWorkersAvailable types.ErrorCode = "NO_WORKERS_AVAILABLE"
	ErrCodeBatchFull          types.ErrorCode = "BATCH_FULL"
	ErrCodeProcessorClosed    types.ErrorCode = "PROCESSOR_CLOSED"
	ErrCodeCircuitOpen        types.ErrorCode = "CIRCUIT_OPEN"
	ErrCodeCircuitTimeout     types.ErrorCode = "CIRCUIT_TIMEOUT"
	ErrCodeRateLimitExceeded  types.ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeLimiterClosed      types.ErrorCode = "LIMITER_CLOSED"
)

// Worker pool errors
var (
	ErrPoolClosed      = errors.New("worker pool is closed")
	ErrNoWorkers       = errors.New("no workers available")
	ErrPoolTimeout     = errors.New("worker pool timeout")
)

// Batch processor errors
var (
	ErrBatchFull       = errors.New("batch is full")
	ErrProcessorClosed = errors.New("batch processor is closed")
)

// Circuit breaker errors
var (
	ErrCircuitOpen    = errors.New("circuit breaker is open")
	ErrCircuitTimeout = errors.New("circuit breaker timeout")
)

// Rate limiter errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrLimiterClosed     = errors.New("rate limiter is closed")
)

// PoolClosed creates a domain error for closed pool
func PoolClosed() *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodePoolClosed,
		Message:    "worker pool is closed",
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// NoWorkersAvailable creates a domain error for no available workers
func NoWorkersAvailable() *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeNoWorkersAvailable,
		Message:    "no workers available",
		HTTPStatus: http.StatusServiceUnavailable,
		Retryable:  true,
	}
}

// BatchFull creates a domain error for full batch
func BatchFull() *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeBatchFull,
		Message:    "batch is full",
		HTTPStatus: http.StatusServiceUnavailable,
		Retryable:  true,
	}
}

// ProcessorClosed creates a domain error for closed processor
func ProcessorClosed() *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeProcessorClosed,
		Message:    "batch processor is closed",
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// CircuitOpen creates a domain error for open circuit
func CircuitOpen() *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeCircuitOpen,
		Message:    "circuit breaker is open",
		HTTPStatus: http.StatusServiceUnavailable,
		Retryable:  true,
	}
}

// CircuitTimeout creates a domain error for circuit timeout
func CircuitTimeout() *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeCircuitTimeout,
		Message:    "circuit breaker timeout",
		HTTPStatus: http.StatusRequestTimeout,
		Retryable:  true,
	}
}

// RateLimitExceeded creates a domain error for rate limit exceeded
func RateLimitExceeded(requestsPerSecond int) *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeRateLimitExceeded,
		Message:    "rate limit exceeded",
		HTTPStatus: http.StatusTooManyRequests,
		Details: map[string]interface{}{
			"limit": requestsPerSecond,
		},
		Retryable: true,
	}
}

// LimiterClosed creates a domain error for closed limiter
func LimiterClosed() *types.DomainError {
	return &types.DomainError{
		Code:       ErrCodeLimiterClosed,
		Message:    "rate limiter is closed",
		HTTPStatus: http.StatusServiceUnavailable,
	}
}