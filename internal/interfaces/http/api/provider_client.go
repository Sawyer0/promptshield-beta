package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/backoff"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/observability/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ProviderError represents a structured error from LLM providers
type ProviderError struct {
	Provider     string            `json:"provider"`
	StatusCode   int               `json:"status_code"`
	ErrorCode    string            `json:"error_code"`
	Message      string            `json:"message"`
	Details      map[string]any    `json:"details,omitempty"`
	Retryable    bool              `json:"retryable"`
	RetryAfter   *time.Duration    `json:"retry_after,omitempty"`
	RequestID    string            `json:"request_id,omitempty"`
	RateLimited  bool              `json:"rate_limited"`
	QuotaExhausted bool            `json:"quota_exhausted"`
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s error [%d]: %s (%s)", e.Provider, e.StatusCode, e.Message, e.ErrorCode)
}

// ProviderClient handles HTTP requests to LLM providers with robust error handling
type ProviderClient struct {
	httpClient *http.Client
	retryConfig RetryConfig
}

type RetryConfig struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	RetryableErrors []int // HTTP status codes that should be retried
}

func NewProviderClient() *ProviderClient {
	return &ProviderClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       60 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		retryConfig: RetryConfig{
			MaxRetries:  3,
			BaseDelay:   100 * time.Millisecond,
			MaxDelay:    5 * time.Second,
			RetryableErrors: []int{
				http.StatusInternalServerError,
				http.StatusBadGateway,
				http.StatusServiceUnavailable,
				http.StatusGatewayTimeout,
				http.StatusTooManyRequests,
			},
		},
	}
}

// ProxyRequest sends a request to the specified provider with retry logic and error handling
func (pc *ProviderClient) ProxyRequest(ctx context.Context, provider string, key *domain.ProviderKey, endpoint string, body []byte) ([]byte, error) {
	// Create tracing span for the entire provider request
	tracer := otel.Tracer("promptshield/provider")
	ctx, span := tracer.Start(ctx, fmt.Sprintf("llm.%s.%s", provider, endpoint),
		trace.WithAttributes(
			attribute.String("llm.provider", provider),
			attribute.String("llm.endpoint", endpoint),
		),
	)
	defer span.End()

	url, headers, err := pc.buildProviderRequest(provider, key, endpoint)
	if err != nil {
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.type", "config_error"))
		return nil, &ProviderError{
			Provider:   provider,
			StatusCode: 0,
			ErrorCode:  "CONFIG_ERROR",
			Message:    fmt.Sprintf("Failed to build provider request: %v", err),
			Retryable:  false,
		}
	}

	// Extract model from request body for accurate metrics and tracing
	model := pc.extractModelFromRequest(body)
	span.SetAttributes(attribute.String("llm.model", model))

	var lastErr error
	for attempt := 0; attempt <= pc.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Record retry metrics
			metrics.LLMRetries.WithLabelValues(provider, fmt.Sprintf("%d", attempt)).Inc()
			
			delay := backoff.FullJitter(pc.retryConfig.BaseDelay, pc.retryConfig.MaxDelay, attempt-1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		startTime := time.Now()
		resp, err := pc.executeRequestWithTracing(ctx, url, headers, body, provider, model, attempt)
		duration := time.Since(startTime)
		
		// Record latency metrics with actual model name
		metrics.LLMLatency.WithLabelValues(provider, model).Observe(duration.Seconds())
		span.SetAttributes(attribute.Float64("llm.duration_seconds", duration.Seconds()))

		if err != nil {
			lastErr = err
			
			// Add error attributes to span
			span.SetAttributes(
				attribute.Bool("error", true),
				attribute.String("error.message", err.Error()),
				attribute.Int("retry.attempt", attempt),
			)
			
			// Check for timeout errors
			if ctx.Err() == context.DeadlineExceeded {
				metrics.LLMTimeouts.WithLabelValues(provider, model).Inc()
				span.SetAttributes(attribute.String("error.type", "timeout"))
			}
			
			if providerErr, ok := err.(*ProviderError); ok {
				// Record error metrics
				retryable := "false"
				if providerErr.Retryable {
					retryable = "true"
				}
				metrics.LLMErrors.WithLabelValues(provider, providerErr.ErrorCode, retryable).Inc()
				
				// Add provider-specific error attributes
				span.SetAttributes(
					attribute.String("error.code", providerErr.ErrorCode),
					attribute.Bool("error.retryable", providerErr.Retryable),
					attribute.Int("http.status_code", providerErr.StatusCode),
				)
				
				if !providerErr.Retryable || attempt == pc.retryConfig.MaxRetries {
					return nil, providerErr
				}
				continue
			}
			// Network or other non-provider errors
			if attempt == pc.retryConfig.MaxRetries {
				return nil, &ProviderError{
					Provider:   provider,
					StatusCode: 0,
					ErrorCode:  "NETWORK_ERROR",
					Message:    fmt.Sprintf("Request failed after %d attempts: %v", attempt+1, err),
					Retryable:  false,
				}
			}
			continue
		}

		// Record successful request with retry count
		if attempt > 0 {
			metrics.LLMRetrySuccess.WithLabelValues(provider, model, fmt.Sprintf("%d", attempt)).Inc()
			span.SetAttributes(attribute.Int("retry.final_attempt", attempt))
		}
		
		// Add success attributes
		span.SetAttributes(
			attribute.Bool("success", true),
			attribute.Int("response.size_bytes", len(resp)),
		)

		return resp, nil
	}

	return nil, lastErr
}

func (pc *ProviderClient) buildProviderRequest(provider string, key *domain.ProviderKey, endpoint string) (string, http.Header, error) {
	apiKey, err := key.DecryptKey()
	if err != nil {
		return "", nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", "PromptShield/0.2.0")

	var baseURL string
	switch provider {
	case "openai":
		baseURL = "https://api.openai.com/v1"
		if key.Endpoint != "" {
			baseURL = strings.TrimSuffix(key.Endpoint, "/")
		}
		headers.Set("Authorization", "Bearer "+apiKey)
		
	case "anthropic":
		baseURL = "https://api.anthropic.com/v1"
		if key.Endpoint != "" {
			baseURL = strings.TrimSuffix(key.Endpoint, "/")
		}
		headers.Set("x-api-key", apiKey)
		headers.Set("anthropic-version", "2023-06-01")
		
	case "azure":
		if key.Endpoint == "" {
			return "", nil, fmt.Errorf("azure endpoint is required")
		}
		baseURL = strings.TrimSuffix(key.Endpoint, "/")
		headers.Set("api-key", apiKey)
		if key.Deployment != "" {
			// Azure OpenAI specific deployment handling
			endpoint = fmt.Sprintf("deployments/%s/%s", key.Deployment, endpoint)
		}
		
	default:
		return "", nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	url := fmt.Sprintf("%s/%s", baseURL, strings.TrimPrefix(endpoint, "/"))
	return url, headers, nil
}

func (pc *ProviderClient) executeRequestWithTracing(ctx context.Context, url string, headers http.Header, body []byte, provider, model string, attempt int) ([]byte, error) {
	// Create span for individual HTTP request
	tracer := otel.Tracer("promptshield/provider")
	ctx, span := tracer.Start(ctx, fmt.Sprintf("http.%s", provider),
		trace.WithAttributes(
			attribute.String("http.url", url),
			attribute.String("http.method", "POST"),
			attribute.String("llm.provider", provider),
			attribute.String("llm.model", model),
			attribute.Int("retry.attempt", attempt),
		),
	)
	defer span.End()

	return pc.executeRequest(ctx, url, headers, body)
}

func (pc *ProviderClient) executeRequest(ctx context.Context, url string, headers http.Header, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = headers
	
	// Inject tracing context into outgoing request headers
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		// Check for specific network errors
		if netErr, ok := err.(net.Error); ok {
			return nil, &ProviderError{
				Provider:   pc.extractProviderFromURL(url),
				StatusCode: 0,
				ErrorCode:  "NETWORK_ERROR",
				Message:    fmt.Sprintf("Network error: %v", netErr),
				Retryable:  netErr.Timeout(),
			}
		}
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{
			Provider:   pc.extractProviderFromURL(url),
			StatusCode: resp.StatusCode,
			ErrorCode:  "READ_ERROR",
			Message:    fmt.Sprintf("Failed to read response body: %v", err),
			Retryable:  false,
		}
	}

	if resp.StatusCode >= 400 {
		return nil, pc.parseProviderError(url, resp.StatusCode, responseBody, resp.Header)
	}

	return responseBody, nil
}

func (pc *ProviderClient) parseProviderError(url string, statusCode int, body []byte, headers http.Header) *ProviderError {
	provider := pc.extractProviderFromURL(url)
	
	providerErr := &ProviderError{
		Provider:     provider,
		StatusCode:   statusCode,
		Retryable:    pc.isRetryableStatus(statusCode),
		RequestID:    headers.Get("x-request-id"),
		RateLimited:  statusCode == http.StatusTooManyRequests,
		QuotaExhausted: false,
	}

	// Parse retry-after header
	if retryAfter := headers.Get("retry-after"); retryAfter != "" {
		if duration, err := time.ParseDuration(retryAfter + "s"); err == nil {
			providerErr.RetryAfter = &duration
		}
	}

	// Try to parse provider-specific error format
	var errorResponse map[string]any
	if err := json.Unmarshal(body, &errorResponse); err == nil {
		switch provider {
		case "openai":
			if errorObj, ok := errorResponse["error"].(map[string]any); ok {
				if code, ok := errorObj["code"].(string); ok {
					providerErr.ErrorCode = code
				}
				if message, ok := errorObj["message"].(string); ok {
					providerErr.Message = message
				}
				if errorType, ok := errorObj["type"].(string); ok {
					providerErr.Details = map[string]any{"type": errorType}
					// Check for quota exhaustion
					if errorType == "insufficient_quota" || strings.Contains(strings.ToLower(providerErr.Message), "quota") {
						providerErr.QuotaExhausted = true
						providerErr.Retryable = false
					}
				}
			}
			
		case "anthropic":
			if errorType, ok := errorResponse["type"].(string); ok {
				providerErr.ErrorCode = errorType
			}
			if message, ok := errorResponse["message"].(string); ok {
				providerErr.Message = message
			}
			
		case "azure":
			if errorObj, ok := errorResponse["error"].(map[string]any); ok {
				if code, ok := errorObj["code"].(string); ok {
					providerErr.ErrorCode = code
				}
				if message, ok := errorObj["message"].(string); ok {
					providerErr.Message = message
				}
			}
		}
	}

	// Set default values if not parsed
	if providerErr.ErrorCode == "" {
		providerErr.ErrorCode = fmt.Sprintf("HTTP_%d", statusCode)
	}
	if providerErr.Message == "" {
		providerErr.Message = string(body)
		if len(providerErr.Message) > 200 {
			providerErr.Message = providerErr.Message[:200] + "..."
		}
	}

	return providerErr
}

func (pc *ProviderClient) extractProviderFromURL(url string) string {
	if strings.Contains(url, "api.openai.com") {
		return "openai"
	}
	if strings.Contains(url, "api.anthropic.com") {
		return "anthropic"
	}
	if strings.Contains(url, "openai.azure.com") {
		return "azure"
	}
	return "unknown"
}

func (pc *ProviderClient) isRetryableStatus(statusCode int) bool {
	for _, retryableCode := range pc.retryConfig.RetryableErrors {
		if statusCode == retryableCode {
			return true
		}
	}
	return false
}

// extractModelFromRequest parses the request body to extract the model name for metrics
func (pc *ProviderClient) extractModelFromRequest(body []byte) string {
	var request map[string]interface{}
	if err := json.Unmarshal(body, &request); err != nil {
		return "unknown"
	}
	
	if model, ok := request["model"].(string); ok && model != "" {
		return model
	}
	
	return "unknown"
}