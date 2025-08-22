package types

import (
	"fmt"
)

// ErrorCode represents structured error codes for the PromptShield system
type ErrorCode string

// DomainError represents a structured domain error with context
type DomainError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	HTTPStatus int                    `json:"-"`
	Retryable  bool                   `json:"retryable"`
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

// WithCause adds a cause to the domain error
func (e *DomainError) WithCause(cause error) *DomainError {
	e.Cause = cause
	return e
}

// WithDetail adds a detail field to the error
func (e *DomainError) WithDetail(key string, value interface{}) *DomainError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}