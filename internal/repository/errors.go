package repository

import (
	"fmt"
	"time"
)

// RepositoryError represents a repository-specific error with additional context
type RepositoryError struct {
	Type        ErrorType
	Operation   string
	Repository  string
	Underlying  error
	Timestamp   time.Time
	Context     map[string]interface{}
	Retryable   bool
}

// ErrorType represents different categories of repository errors
type ErrorType string

const (
	// Connection errors
	ErrorTypeConnectionFailed    ErrorType = "connection_failed"
	ErrorTypeConnectionTimeout   ErrorType = "connection_timeout"
	ErrorTypeConnectionLost      ErrorType = "connection_lost"
	
	// Configuration errors
	ErrorTypeConfigInvalid       ErrorType = "config_invalid"
	ErrorTypeConfigMissing       ErrorType = "config_missing"
	
	// Repository operation errors
	ErrorTypeRepositoryInit      ErrorType = "repository_init"
	ErrorTypeRepositoryOperation ErrorType = "repository_operation"
	ErrorTypeRepositoryNotFound  ErrorType = "repository_not_found"
	
	// Cache errors
	ErrorTypeCacheUnavailable    ErrorType = "cache_unavailable"
	ErrorTypeCacheOperation      ErrorType = "cache_operation"
	
	// Health check errors
	ErrorTypeHealthCheck         ErrorType = "health_check"
	
	// Factory errors
	ErrorTypeFactoryInit         ErrorType = "factory_init"
	ErrorTypeFactoryValidation   ErrorType = "factory_validation"
)

// Error implements the error interface
func (e *RepositoryError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("%s error in %s.%s: %v", e.Type, e.Repository, e.Operation, e.Underlying)
	}
	return fmt.Sprintf("%s error in %s.%s", e.Type, e.Repository, e.Operation)
}

// Unwrap returns the underlying error for error unwrapping
func (e *RepositoryError) Unwrap() error {
	return e.Underlying
}

// IsRetryable returns true if the error is potentially retryable
func (e *RepositoryError) IsRetryable() bool {
	return e.Retryable
}

// WithContext adds context information to the error
func (e *RepositoryError) WithContext(key string, value interface{}) *RepositoryError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewRepositoryError creates a new repository error with the given parameters
func NewRepositoryError(errType ErrorType, repository, operation string, underlying error) *RepositoryError {
	return &RepositoryError{
		Type:       errType,
		Repository: repository,
		Operation:  operation,
		Underlying: underlying,
		Timestamp:  time.Now(),
		Context:    make(map[string]interface{}),
		Retryable:  isRetryableError(errType, underlying),
	}
}

// isRetryableError determines if an error type is potentially retryable
func isRetryableError(errType ErrorType, underlying error) bool {
	switch errType {
	case ErrorTypeConnectionTimeout, ErrorTypeConnectionLost, ErrorTypeCacheUnavailable:
		return true
	case ErrorTypeConnectionFailed:
		// Some connection failures might be retryable (network issues)
		return true
	case ErrorTypeConfigInvalid, ErrorTypeConfigMissing, ErrorTypeRepositoryNotFound:
		// Configuration and not found errors are typically not retryable
		return false
	default:
		return false
	}
}

// ConnectionError creates a connection-related error
func ConnectionError(repository, operation string, underlying error) *RepositoryError {
	return NewRepositoryError(ErrorTypeConnectionFailed, repository, operation, underlying)
}

// ConfigurationError creates a configuration-related error
func ConfigurationError(repository, operation string, underlying error) *RepositoryError {
	return NewRepositoryError(ErrorTypeConfigInvalid, repository, operation, underlying)
}

// HealthCheckError creates a health check-related error
func HealthCheckError(repository, operation string, underlying error) *RepositoryError {
	return NewRepositoryError(ErrorTypeHealthCheck, repository, operation, underlying)
}

// FactoryError creates a factory-related error
func FactoryError(repository, operation string, underlying error) *RepositoryError {
	return NewRepositoryError(ErrorTypeFactoryInit, repository, operation, underlying)
}