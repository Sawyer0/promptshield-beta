package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/security/crypto"
	"github.com/google/uuid"
)

// TestProviderKey creates a domain.ProviderKey for testing with proper encryption
func createTestProviderKey(provider, endpoint, apiKey string) *domain.ProviderKey {
	// Encrypt the API key for testing
	encryptedKey, err := crypto.EncryptString(apiKey)
	if err != nil {
		// Fallback to plain text if encryption fails
		encryptedKey = apiKey
	}
	
	return &domain.ProviderKey{
		ID:           uuid.New(),
		Provider:     domain.Provider(provider),
		EncryptedKey: encryptedKey,
		Endpoint:     endpoint,
	}
}

func TestProviderClient_ErrorHandling(t *testing.T) {
	// Set test encryption key to allow decryption to work
	os.Setenv("PS_ENCRYPTION_KEY", "test-encryption-key-32-characters!")
	defer os.Unsetenv("PS_ENCRYPTION_KEY")
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		provider       string
		expectedError  string
		expectedRetryable bool
		expectedRateLimit bool
		expectedQuotaExhausted bool
	}{
		{
			name:         "OpenAI Rate Limit",
			statusCode:   429,
			responseBody: `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_exceeded", "code": "rate_limit_exceeded"}}`,
			provider:     "openai",
			expectedError: "rate_limit_exceeded",
			expectedRetryable: true,
			expectedRateLimit: true,
		},
		{
			name:         "OpenAI Quota Exhausted",
			statusCode:   429,
			responseBody: `{"error": {"message": "You exceeded your current quota", "type": "insufficient_quota", "code": "insufficient_quota"}}`,
			provider:     "openai",
			expectedError: "insufficient_quota",
			expectedRetryable: false,
			expectedQuotaExhausted: true,
		},
		{
			name:         "Anthropic Auth Error",
			statusCode:   401,
			responseBody: `{"type": "authentication_error", "message": "Invalid API key"}`,
			provider:     "anthropic",
			expectedError: "authentication_error",
			expectedRetryable: false,
		},
		{
			name:         "Azure Service Unavailable",
			statusCode:   503,
			responseBody: `{"error": {"code": "ServiceUnavailable", "message": "Service temporarily unavailable"}}`,
			provider:     "azure",
			expectedError: "ServiceUnavailable",
			expectedRetryable: true,
		},
		{
			name:         "Generic Internal Error",
			statusCode:   500,
			responseBody: `{"error": "Internal server error"}`,
			provider:     "openai",
			expectedError: "HTTP_500",
			expectedRetryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("x-request-id", "test-request-123")
				if tt.expectedRateLimit {
					w.Header().Set("retry-after", "60")
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create provider client
			client := NewProviderClient()
			
			// Create mock provider key with plain text for testing
			key := createTestProviderKey(tt.provider, server.URL, "test-api-key")

			// Execute request
			ctx := context.Background()
			_, err := client.ProxyRequest(ctx, tt.provider, key, "chat/completions", []byte(`{"model": "test"}`))

			// Verify error
			if err == nil {
				t.Fatal("Expected error but got none")
			}

			providerErr, ok := err.(*ProviderError)
			if !ok {
				t.Fatalf("Expected ProviderError, got %T", err)
			}

			if providerErr.ErrorCode != tt.expectedError {
				t.Errorf("Expected error code %s, got %s", tt.expectedError, providerErr.ErrorCode)
			}

			if providerErr.Retryable != tt.expectedRetryable {
				t.Errorf("Expected retryable %v, got %v", tt.expectedRetryable, providerErr.Retryable)
			}

			if providerErr.RateLimited != tt.expectedRateLimit {
				t.Errorf("Expected rate limited %v, got %v", tt.expectedRateLimit, providerErr.RateLimited)
			}

			if providerErr.QuotaExhausted != tt.expectedQuotaExhausted {
				t.Errorf("Expected quota exhausted %v, got %v", tt.expectedQuotaExhausted, providerErr.QuotaExhausted)
			}

			if providerErr.RequestID != "test-request-123" {
				t.Errorf("Expected request ID 'test-request-123', got %s", providerErr.RequestID)
			}
		})
	}
}

func TestProviderClient_RetryLogic(t *testing.T) {
	// Set test encryption key to allow decryption to work
	os.Setenv("PS_ENCRYPTION_KEY", "test-encryption-key-32-characters!")
	defer os.Unsetenv("PS_ENCRYPTION_KEY")
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(500)
			w.Write([]byte(`{"error": "Internal server error"}`))
		} else {
			w.WriteHeader(200)
			w.Write([]byte(`{"id": "test", "model": "gpt-3.5-turbo", "choices": []}`))
		}
	}))
	defer server.Close()

	client := NewProviderClient()
	
	key := createTestProviderKey("openai", server.URL, "test-api-key")

	ctx := context.Background()
	resp, err := client.ProxyRequest(ctx, "openai", key, "chat/completions", []byte(`{"model": "gpt-3.5-turbo"}`))

	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response but got nil")
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestProviderClient_ContextTimeout(t *testing.T) {
	// Set test encryption key to allow decryption to work
	os.Setenv("PS_ENCRYPTION_KEY", "test-encryption-key-32-characters!")
	defer os.Unsetenv("PS_ENCRYPTION_KEY")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Delay response
		w.WriteHeader(200)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewProviderClient()
	
	key := createTestProviderKey("openai", server.URL, "test-api-key")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.ProxyRequest(ctx, "openai", key, "chat/completions", []byte(`{"model": "test"}`))

	if err == nil {
		t.Fatal("Expected timeout error but got none")
	}

	if ctx.Err() == nil {
		t.Fatal("Expected context to be cancelled")
	}
}

func TestProviderFormatConversion(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		request  *ProxyRequest
		expected string
	}{
		{
			name:     "OpenAI Format",
			provider: "openai",
			request: &ProxyRequest{
				Model: "gpt-3.5-turbo",
				Messages: []ChatMessage{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens:   100,
				Temperature: 0.7,
			},
			expected: `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}],"max_tokens":100,"temperature":0.7,"stream":false}`,
		},
		{
			name:     "Anthropic Format",
			provider: "anthropic",
			request: &ProxyRequest{
				Model: "claude-3-sonnet-20240229",
				Messages: []ChatMessage{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens:   100,
				Temperature: 0.7,
			},
			expected: `{"max_tokens":100,"messages":[{"content":"Hello","role":"user"}],"model":"claude-3-sonnet-20240229","stream":false,"temperature":0.7}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToProviderFormat(tt.provider, tt.request)
			if err != nil {
				t.Fatalf("convertToProviderFormat failed: %v", err)
			}

			// Parse both JSON strings to compare structure
			var resultMap, expectedMap map[string]interface{}
			
			if err := json.Unmarshal(result, &resultMap); err != nil {
				t.Fatalf("Failed to parse result JSON: %v", err)
			}
			
			if err := json.Unmarshal([]byte(tt.expected), &expectedMap); err != nil {
				t.Fatalf("Failed to parse expected JSON: %v", err)
			}

			// Compare key fields
			if resultMap["model"] != expectedMap["model"] {
				t.Errorf("Model mismatch: got %v, want %v", resultMap["model"], expectedMap["model"])
			}
			
			if resultMap["max_tokens"] != expectedMap["max_tokens"] {
				t.Errorf("MaxTokens mismatch: got %v, want %v", resultMap["max_tokens"], expectedMap["max_tokens"])
			}
		})
	}
}