package types

import "time"

// LLMRequest represents a normalized request to any LLM provider
// Consolidates ProxyRequest from proxy.go and ChatRequest from llm_clients.go
type LLMRequest struct {
	Model       string                 `json:"model"`
	Messages    []ChatMessage          `json:"messages,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`     // For completion endpoints
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Extra       map[string]interface{} `json:"-"` // Provider-specific fields
}

// ChatMessage represents a chat message
// Consolidates ChatMessage definitions from proxy.go and llm_clients.go
type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// LLMResponse represents a normalized response from any LLM provider
// Consolidates ProxyResponse from proxy.go and ChatResponse from llm_clients.go
type LLMResponse struct {
	ID      string      `json:"id"`
	Object  string      `json:"object,omitempty"`  // From llm_clients.go
	Created int64       `json:"created,omitempty"` // From llm_clients.go
	Model   string      `json:"model"`
	Choices []Choice    `json:"choices"`
	Usage   *UsageStats `json:"usage,omitempty"`
	Meta    *ProxyMeta  `json:"_meta,omitempty"` // PromptShield-specific metadata
}

// Choice represents a completion choice
// Consolidates Choice from proxy.go and ChatChoice from llm_clients.go
type Choice struct {
	Index        int          `json:"index"`
	Message      *ChatMessage `json:"message,omitempty"` // For chat completions
	Text         string       `json:"text,omitempty"`    // For text completions
	FinishReason string       `json:"finish_reason,omitempty"`
}

// UsageStats represents token usage information
// Consolidates Usage from proxy.go and UsageStats from llm_clients.go
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProxyMeta contains metadata about the proxied request
// From proxy.go - PromptShield-specific metadata
type ProxyMeta struct {
	Provider      string `json:"provider"`
	RequestID     string `json:"request_id"`
	PolicyApplied string `json:"policy_applied,omitempty"`
	CacheStatus   string `json:"cache_status,omitempty"`
	ProcessingMs  int64  `json:"processing_ms"`
	TokensUsed    int    `json:"tokens_used"`
}

// EmbeddingRequest represents embedding requests
// Extracted from universalEmbeddingsHandler in proxy.go
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// CompletionRequest represents text completion requests
// For completion endpoints (non-chat)
type CompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// Provider represents supported LLM providers
type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderAzure  Provider = "azure"
)

// ProviderError represents structured errors from LLM providers
// From provider_client.go
type ProviderError struct {
	Provider       string         `json:"provider"`
	ErrorCode      string         `json:"error_code"`
	Message        string         `json:"message"`
	StatusCode     int            `json:"status_code"`
	RequestID      string         `json:"request_id,omitempty"`
	RetryAfter     *time.Duration `json:"retry_after,omitempty"`
	RateLimited    bool           `json:"rate_limited"`
	QuotaExhausted bool           `json:"quota_exhausted"`
	Retryable      bool           `json:"retryable"`
}

func (e *ProviderError) Error() string {
	return e.Message
}