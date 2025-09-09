package repository

import (
	"errors"
	"testing"
	"time"
)

var testError = errors.New("test error")

func TestRepositoryError(t *testing.T) {
	underlying := errors.New("connection refused")
	
	err := NewRepositoryError(ErrorTypeConnectionFailed, "postgresql", "connect", underlying)
	
	// Test basic properties
	if err.Type != ErrorTypeConnectionFailed {
		t.Errorf("Expected error type %s, got %s", ErrorTypeConnectionFailed, err.Type)
	}
	if err.Repository != "postgresql" {
		t.Errorf("Expected repository 'postgresql', got '%s'", err.Repository)
	}
	if err.Operation != "connect" {
		t.Errorf("Expected operation 'connect', got '%s'", err.Operation)
	}
	if err.Underlying != underlying {
		t.Errorf("Expected underlying error to be preserved")
	}
	
	// Test error message
	expectedMsg := "connection_failed error in postgresql.connect: connection refused"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
	
	// Test unwrapping
	if errors.Unwrap(err) != underlying {
		t.Errorf("Expected unwrapped error to be underlying error")
	}
	
	// Test retryable
	if !err.IsRetryable() {
		t.Errorf("Expected connection failed error to be retryable")
	}
	
	// Test context
	err.WithContext("attempt", 1)
	err.WithContext("timeout", "5s")
	
	if err.Context["attempt"] != 1 {
		t.Errorf("Expected context 'attempt' to be 1, got %v", err.Context["attempt"])
	}
	if err.Context["timeout"] != "5s" {
		t.Errorf("Expected context 'timeout' to be '5s', got %v", err.Context["timeout"])
	}
	
	// Test timestamp
	if err.Timestamp.IsZero() {
		t.Errorf("Expected timestamp to be set")
	}
	if time.Since(err.Timestamp) > time.Second {
		t.Errorf("Expected timestamp to be recent")
	}
}

func TestRepositoryErrorWithoutUnderlying(t *testing.T) {
	err := NewRepositoryError(ErrorTypeConfigMissing, "factory", "validation", nil)
	
	expectedMsg := "config_missing error in factory.validation"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
	
	if errors.Unwrap(err) != nil {
		t.Errorf("Expected unwrapped error to be nil")
	}
}

func TestErrorTypeRetryability(t *testing.T) {
	testCases := []struct {
		errorType ErrorType
		retryable bool
	}{
		{ErrorTypeConnectionFailed, true},
		{ErrorTypeConnectionTimeout, true},
		{ErrorTypeConnectionLost, true},
		{ErrorTypeCacheUnavailable, true},
		{ErrorTypeConfigInvalid, false},
		{ErrorTypeConfigMissing, false},
		{ErrorTypeRepositoryNotFound, false},
		{ErrorTypeRepositoryInit, false},
		{ErrorTypeRepositoryOperation, false},
		{ErrorTypeCacheOperation, false},
		{ErrorTypeHealthCheck, false},
		{ErrorTypeFactoryInit, false},
		{ErrorTypeFactoryValidation, false},
	}
	
	for _, tc := range testCases {
		err := NewRepositoryError(tc.errorType, "test", "test", nil)
		if err.IsRetryable() != tc.retryable {
			t.Errorf("Expected error type %s to have retryable=%v, got %v", 
				tc.errorType, tc.retryable, err.IsRetryable())
		}
	}
}

func TestErrorConstructors(t *testing.T) {
	underlying := testError
	
	// Test ConnectionError
	connErr := ConnectionError("postgresql", "connect", underlying)
	if connErr.Type != ErrorTypeConnectionFailed {
		t.Errorf("Expected ConnectionError to have type %s", ErrorTypeConnectionFailed)
	}
	
	// Test ConfigurationError
	configErr := ConfigurationError("factory", "validate", underlying)
	if configErr.Type != ErrorTypeConfigInvalid {
		t.Errorf("Expected ConfigurationError to have type %s", ErrorTypeConfigInvalid)
	}
	
	// Test HealthCheckError
	healthErr := HealthCheckError("redis", "ping", underlying)
	if healthErr.Type != ErrorTypeHealthCheck {
		t.Errorf("Expected HealthCheckError to have type %s", ErrorTypeHealthCheck)
	}
	
	// Test FactoryError
	factoryErr := FactoryError("production", "init", underlying)
	if factoryErr.Type != ErrorTypeFactoryInit {
		t.Errorf("Expected FactoryError to have type %s", ErrorTypeFactoryInit)
	}
}

func TestErrorChaining(t *testing.T) {
	originalErr := testError
	repoErr := ConnectionError("postgresql", "connect", originalErr)
	
	// Test that we can use errors.Is
	if !errors.Is(repoErr, originalErr) {
		t.Errorf("Expected errors.Is to find original error in chain")
	}
	
	// Test that we can use errors.As
	var targetErr *RepositoryError
	if !errors.As(repoErr, &targetErr) {
		t.Errorf("Expected errors.As to find RepositoryError in chain")
	}
	
	if targetErr.Type != ErrorTypeConnectionFailed {
		t.Errorf("Expected found error to have correct type")
	}
}