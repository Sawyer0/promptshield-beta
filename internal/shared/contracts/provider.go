package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// LLMProvider defines the interface for LLM provider integrations
type LLMProvider interface {
	// GetProvider returns the provider type
	GetProvider() types.Provider
	
	// SendRequest sends a request to the LLM provider
	SendRequest(ctx context.Context, req *types.LLMRequest, credentials *types.ProviderKey) (*types.LLMResponse, error)
	
	// SendStreamingRequest sends a streaming request to the LLM provider
	SendStreamingRequest(ctx context.Context, req *types.LLMRequest, credentials *types.ProviderKey) (<-chan *types.LLMResponse, error)
	
	// ValidateCredentials validates provider credentials
	ValidateCredentials(ctx context.Context, credentials *types.ProviderKey) error
	
	// GetModels returns available models for the provider
	GetModels(ctx context.Context, credentials *types.ProviderKey) ([]string, error)
	
	// EstimateTokens estimates token count for the given content
	EstimateTokens(ctx context.Context, content string, model string) (int, error)
	
	// HealthCheck verifies the provider is accessible
	HealthCheck(ctx context.Context) error
}

// ProviderFactory defines the interface for creating provider instances
type ProviderFactory interface {
	// CreateProvider creates a new provider instance
	CreateProvider(providerType types.Provider, config map[string]interface{}) (LLMProvider, error)
	
	// GetSupportedProviders returns list of supported providers
	GetSupportedProviders() []types.Provider
	
	// ValidateConfig validates provider configuration
	ValidateConfig(providerType types.Provider, config map[string]interface{}) error
}

// ProviderAdapter defines the interface for converting between provider formats
type ProviderAdapter interface {
	// ConvertRequest converts a universal request to provider-specific format
	ConvertRequest(req *types.LLMRequest) (interface{}, error)
	
	// ConvertResponse converts a provider-specific response to universal format
	ConvertResponse(resp interface{}) (*types.LLMResponse, error)
	
	// ConvertError converts a provider-specific error to universal format
	ConvertError(err error) *types.ProviderError
	
	// GetContentType returns the expected content type for requests
	GetContentType() string
	
	// GetEndpoint returns the API endpoint for the provider
	GetEndpoint(model string) string
}

// HTTPClient defines the interface for HTTP client operations
type HTTPClient interface {
	// Do sends an HTTP request and returns the response
	Do(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)
	
	// DoStream sends a streaming HTTP request
	DoStream(ctx context.Context, req *HTTPRequest) (<-chan []byte, error)
	
	// SetRetryPolicy configures retry behavior
	SetRetryPolicy(policy RetryPolicy)
	
	// SetTimeout configures request timeout
	SetTimeout(timeout time.Duration)
}

// HTTPRequest represents an HTTP request
type HTTPRequest struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// RetryPolicy defines retry behavior for HTTP requests
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	Multiplier      float64       `json:"multiplier"`
	RetryableErrors []int         `json:"retryable_errors"` // HTTP status codes
}

// RateLimiter defines the interface for rate limiting provider requests
type RateLimiter interface {
	// Allow returns true if the request is allowed
	Allow(ctx context.Context, key string) (bool, error)
	
	// Wait blocks until the request can proceed
	Wait(ctx context.Context, key string) error
	
	// GetLimit returns the current rate limit configuration
	GetLimit(key string) (int, time.Duration, error)
	
	// SetLimit configures rate limiting for a key
	SetLimit(key string, requests int, window time.Duration) error
}

// CircuitBreaker defines the interface for circuit breaker pattern
type CircuitBreaker interface {
	// Execute runs a function with circuit breaker protection
	Execute(ctx context.Context, operation func() error) error
	
	// GetState returns the current circuit breaker state
	GetState() CircuitBreakerState
	
	// GetMetrics returns circuit breaker metrics
	GetMetrics() CircuitBreakerMetrics
	
	// Reset manually resets the circuit breaker
	Reset()
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerStateClosed   CircuitBreakerState = "closed"
	CircuitBreakerStateOpen     CircuitBreakerState = "open"
	CircuitBreakerStateHalfOpen CircuitBreakerState = "half_open"
)

// CircuitBreakerMetrics represents circuit breaker metrics
type CircuitBreakerMetrics struct {
	TotalRequests   int64                 `json:"total_requests"`
	SuccessRequests int64                 `json:"success_requests"`
	FailureRequests int64                 `json:"failure_requests"`
	State           CircuitBreakerState   `json:"state"`
	LastFailure     *time.Time            `json:"last_failure,omitempty"`
	NextRetryAt     *time.Time            `json:"next_retry_at,omitempty"`
}

// TokenCounter defines the interface for token counting
type TokenCounter interface {
	// CountTokens counts tokens in the given text for the specified model
	CountTokens(text string, model string) (int, error)
	
	// CountRequestTokens counts tokens in a request
	CountRequestTokens(req *types.LLMRequest) (int, error)
	
	// CountResponseTokens counts tokens in a response
	CountResponseTokens(resp *types.LLMResponse) (int, error)
	
	// GetTokenLimit returns the token limit for a model
	GetTokenLimit(model string) (int, error)
}

// ProviderMonitor defines the interface for monitoring provider health and performance
type ProviderMonitor interface {
	// RecordRequest records metrics for a provider request
	RecordRequest(ctx context.Context, provider types.Provider, model string, latency time.Duration, success bool)
	
	// RecordError records an error for a provider
	RecordError(ctx context.Context, provider types.Provider, errorType string, err error)
	
	// GetMetrics returns provider metrics
	GetMetrics(ctx context.Context, provider types.Provider, window time.Duration) (map[string]interface{}, error)
	
	// GetHealthStatus returns the health status of providers
	GetHealthStatus(ctx context.Context) (map[types.Provider]bool, error)
}

// KeyManager defines the interface for managing provider API keys
type KeyManager interface {
	// EncryptKey encrypts a provider API key
	EncryptKey(plaintext string) (string, error)
	
	// DecryptKey decrypts a provider API key
	DecryptKey(encrypted string) (string, error)
	
	// RotateKey generates a new encryption key and re-encrypts all keys
	RotateKey(ctx context.Context) error
	
	// ValidateKey validates that a key can be decrypted
	ValidateKey(encrypted string) error
}