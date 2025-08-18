package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// DefaultClientConfig returns a default HTTP client configuration
func DefaultClientConfig() *types.HTTPClientConfig {
	return &types.HTTPClientConfig{
		Timeout:             30 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		UserAgent:           "promptshield/1.0",
		RetryConfig: &types.RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
	}
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(config *types.HTTPClientConfig) *http.Client {
	if config == nil {
		config = DefaultClientConfig()
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableKeepAlives:   config.DisableKeepAlives,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
}

// SecureClient creates an HTTP client with secure defaults
func SecureClient() *http.Client {
	config := DefaultClientConfig()
	
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableKeepAlives:   config.DisableKeepAlives,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
}

// WithHeader adds a header to the request
func WithHeader(key, value string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// WithUserAgent sets the User-Agent header
func WithUserAgent(userAgent string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("User-Agent", userAgent)
	}
}

// WithAuth sets the Authorization header
func WithAuth(token string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// WithAPIKey sets the API key header
func WithAPIKey(key string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("X-API-Key", key)
	}
}

// WithContentType sets the Content-Type header
func WithContentType(contentType string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("Content-Type", contentType)
	}
}

// WithAccept sets the Accept header
func WithAccept(accept string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("Accept", accept)
	}
}

// WithCorrelationID sets the correlation ID header
func WithCorrelationID(id string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("X-PS-Correlation-ID", id)
	}
}

// WithTenantID sets the tenant ID header
func WithTenantID(tenantID string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("X-PS-Tenant-ID", tenantID)
	}
}

// WithTimeout sets a context timeout for the request
func WithTimeout(timeout time.Duration) func(*http.Request) {
	return func(req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		_ = cancel // Will be canceled when request completes
		*req = *req.WithContext(ctx)
	}
}

// NewRequest creates a new HTTP request with options
func NewRequest(ctx context.Context, method, url string, body interface{}, options ...func(*http.Request)) (*http.Request, error) {
	var reqBody []byte
	var err error

	if body != nil {
		switch v := body.(type) {
		case []byte:
			reqBody = v
		case string:
			reqBody = []byte(v)
		default:
			reqBody, err = json.Marshal(v)
			if err != nil {
				return nil, err
			}
		}
	}

	var reqBodyReader io.Reader
	if reqBody != nil {
		reqBodyReader = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBodyReader)
	if err != nil {
		return nil, err
	}

	// Apply default headers
	req.Header.Set("User-Agent", "promptshield/1.0")
	req.Header.Set("Accept", "application/json")
	
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply custom options
	for _, option := range options {
		option(req)
	}

	return req, nil
}

// IsRetryableStatusCode checks if an HTTP status code indicates a retryable error
func IsRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// IsTemporaryError checks if an error is temporary and retryable
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network errors
	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary() || netErr.Timeout()
	}

	// Check for context timeout
	if err == context.DeadlineExceeded {
		return true
	}

	return false
}