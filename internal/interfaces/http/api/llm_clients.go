package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClient defines the interface for LLM provider HTTP clients
type LLMClient interface {
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	GetProviderName() string
	ValidateKey(ctx context.Context, key string) error
}

// ChatRequest represents a standardized chat completion request
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// ChatResponse represents a standardized chat completion response
type ChatResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChatChoice  `json:"choices"`
	Usage   *UsageStats   `json:"usage,omitempty"`
}

// ChatChoice represents a completion choice
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// UsageStats represents token usage statistics
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIClient implements LLM client for OpenAI API
type OpenAIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		BaseURL: "https://api.openai.com/v1",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *OpenAIClient) GetProviderName() string {
	return "openai"
}

func (c *OpenAIClient) ValidateKey(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/models", nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate key: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

func (c *OpenAIClient) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	
	// Note: API key would be set by the caller from tenant's provider key store
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &chatResp, nil
}

// AnthropicClient implements LLM client for Anthropic API
type AnthropicClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewAnthropicClient() *AnthropicClient {
	return &AnthropicClient{
		BaseURL: "https://api.anthropic.com/v1",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *AnthropicClient) GetProviderName() string {
	return "anthropic"
}

func (c *AnthropicClient) ValidateKey(ctx context.Context, key string) error {
	// Simple validation request to Anthropic
	testReq := map[string]interface{}{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 1,
		"messages": []map[string]string{
			{"role": "user", "content": "test"},
		},
	}
	
	jsonData, _ := json.Marshal(testReq)
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate key: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}
	
	return nil
}

func (c *AnthropicClient) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert to Anthropic format
	anthropicReq := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"messages":   req.Messages,
	}
	
	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}
	
	jsonData, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, err
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse Anthropic response and convert to standard format
	var anthropicResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// Convert to standard ChatResponse format
	chatResp := &ChatResponse{
		ID:      fmt.Sprintf("%v", anthropicResp["id"]),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatChoice{},
	}
	
	if content, ok := anthropicResp["content"].([]interface{}); ok && len(content) > 0 {
		if textBlock, ok := content[0].(map[string]interface{}); ok {
			if text, ok := textBlock["text"].(string); ok {
				chatResp.Choices = append(chatResp.Choices, ChatChoice{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: text,
					},
					FinishReason: "stop",
				})
			}
		}
	}
	
	if usage, ok := anthropicResp["usage"].(map[string]interface{}); ok {
		chatResp.Usage = &UsageStats{
			PromptTokens:     int(usage["input_tokens"].(float64)),
			CompletionTokens: int(usage["output_tokens"].(float64)),
		}
		chatResp.Usage.TotalTokens = chatResp.Usage.PromptTokens + chatResp.Usage.CompletionTokens
	}
	
	return chatResp, nil
}

// LLMClientFactory creates LLM clients based on provider
func NewLLMClient(provider string) (LLMClient, error) {
	switch provider {
	case "openai":
		return NewOpenAIClient(), nil
	case "anthropic":
		return NewAnthropicClient(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// ProxyLLMRequest handles proxying requests to the appropriate LLM provider
func ProxyLLMRequest(ctx context.Context, client LLMClient, apiKey string, req *ChatRequest) (*ChatResponse, error) {
	// This would be called from the proxy endpoints with the tenant's API key
	// The actual key injection would happen in the HTTP request headers
	
	if client == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}
	
	// Validate the request
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}
	
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	
	// Apply default limits
	if req.MaxTokens <= 0 {
		req.MaxTokens = 1000
	}
	if req.MaxTokens > 4000 {
		req.MaxTokens = 4000
	}
	
	// Make the LLM request
	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	
	return resp, nil
}