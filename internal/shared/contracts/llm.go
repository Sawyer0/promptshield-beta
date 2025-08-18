package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// LLMProvider defines the interface for LLM provider implementations
type LLMProvider interface {
	// AnalyzeContent analyzes content using the LLM
	AnalyzeContent(ctx context.Context, request *types.LLMRequest) (*types.LLMResponse, error)
	
	// AnalyzeContentStream analyzes content using streaming
	AnalyzeContentStream(ctx context.Context, request *types.LLMRequest) (<-chan *types.LLMResponse, error)
	
	// GetProviderInfo returns information about the provider
	GetProviderInfo() *types.ProviderInfo
	
	// IsAvailable checks if the provider is available
	IsAvailable(ctx context.Context) bool
	
	// GetModels returns available models for this provider
	GetModels(ctx context.Context) ([]string, error)
	
	// ValidateModel validates if a model is supported
	ValidateModel(ctx context.Context, model string) error
	
	// EstimateCost estimates the cost for a request
	EstimateCost(ctx context.Context, request *types.LLMRequest) (*types.CostEstimate, error)
	
	// GetUsage returns usage statistics for the provider
	GetUsage(ctx context.Context, timeRange types.TimeRange) (*types.ProviderUsage, error)
}

// LLMRouter defines the interface for routing requests to appropriate providers
type LLMRouter interface {
	// RouteRequest routes a request to the best available provider
	RouteRequest(ctx context.Context, request *types.LLMRequest) (types.Provider, error)
	
	// GetProvider returns a specific provider instance
	GetProvider(ctx context.Context, provider types.Provider) (LLMProvider, error)
	
	// RegisterProvider registers a new provider
	RegisterProvider(provider types.Provider, instance LLMProvider) error
	
	// UnregisterProvider unregisters a provider
	UnregisterProvider(provider types.Provider) error
	
	// ListProviders returns all registered providers
	ListProviders() []types.Provider
	
	// SetRoutingStrategy sets the routing strategy
	SetRoutingStrategy(strategy types.RoutingStrategy) error
	
	// GetRoutingStrategy returns the current routing strategy
	GetRoutingStrategy() types.RoutingStrategy
	
	// HealthCheck performs health check on all providers
	HealthCheck(ctx context.Context) (map[types.Provider]bool, error)
}

// LLMCache defines the interface for caching LLM responses
type LLMCache interface {
	// Get retrieves a cached response
	Get(ctx context.Context, key string) (*types.LLMResponse, error)
	
	// Set stores a response in cache
	Set(ctx context.Context, key string, response *types.LLMResponse, ttl time.Duration) error
	
	// Delete removes a cached response
	Delete(ctx context.Context, key string) error
	
	// Clear clears all cached responses
	Clear(ctx context.Context) error
	
	// GetStats returns cache statistics
	GetStats(ctx context.Context) (*types.CacheStats, error)
	
	// GenerateKey generates a cache key for a request
	GenerateKey(request *types.LLMRequest) string
	
	// SetTTL sets default TTL for cached responses
	SetTTL(ttl time.Duration) error
	
	// GetTTL returns the default TTL
	GetTTL() time.Duration
}

// LLMRateLimiter defines the interface for rate limiting LLM requests
type LLMRateLimiter interface {
	// CheckLimit checks if a request is within rate limits
	CheckLimit(ctx context.Context, provider types.Provider, userID string) (*types.RateLimitResult, error)
	
	// ConsumeLimit consumes from the rate limit
	ConsumeLimit(ctx context.Context, provider types.Provider, userID string, tokens int) error
	
	// GetLimits returns current rate limits for a provider
	GetLimits(ctx context.Context, provider types.Provider) (*types.RateLimit, error)
	
	// SetLimits sets rate limits for a provider
	SetLimits(ctx context.Context, provider types.Provider, limits *types.RateLimit) error
	
	// ResetLimits resets rate limits for a user
	ResetLimits(ctx context.Context, provider types.Provider, userID string) error
	
	// GetUsage returns current rate limit usage
	GetUsage(ctx context.Context, provider types.Provider, userID string) (*types.RateLimitUsage, error)
}

// LLMMonitor defines the interface for monitoring LLM operations
type LLMMonitor interface {
	// RecordRequest records an LLM request for monitoring
	RecordRequest(ctx context.Context, provider types.Provider, request *types.LLMRequest) error
	
	// RecordResponse records an LLM response for monitoring
	RecordResponse(ctx context.Context, provider types.Provider, response *types.LLMResponse) error
	
	// RecordError records an LLM error for monitoring
	RecordError(ctx context.Context, provider types.Provider, err error) error
	
	// GetMetrics returns LLM operation metrics
	GetMetrics(ctx context.Context, timeRange types.TimeRange) (*types.LLMMetrics, error)
	
	// GetProviderMetrics returns metrics for a specific provider
	GetProviderMetrics(ctx context.Context, provider types.Provider, timeRange types.TimeRange) (*types.ProviderMetrics, error)
	
	// GetLatencyStats returns latency statistics
	GetLatencyStats(ctx context.Context, provider types.Provider, timeRange types.TimeRange) (*types.LatencyStats, error)
	
	// GetErrorRate returns error rate for a provider
	GetErrorRate(ctx context.Context, provider types.Provider, timeRange types.TimeRange) (float64, error)
	
	// GetThroughput returns throughput metrics
	GetThroughput(ctx context.Context, provider types.Provider, timeRange types.TimeRange) (*types.ThroughputMetrics, error)
}

// LLMFallbackManager defines the interface for managing LLM fallbacks
type LLMFallbackManager interface {
	// GetFallbackProvider returns a fallback provider for a failed provider
	GetFallbackProvider(ctx context.Context, failedProvider types.Provider) (types.Provider, error)
	
	// SetFallbackChain sets the fallback chain for a provider
	SetFallbackChain(provider types.Provider, fallbacks []types.Provider) error
	
	// GetFallbackChain returns the fallback chain for a provider
	GetFallbackChain(provider types.Provider) ([]types.Provider, error)
	
	// ExecuteWithFallback executes a request with automatic fallback
	ExecuteWithFallback(ctx context.Context, request *types.LLMRequest) (*types.LLMResponse, error)
	
	// RecordFailure records a provider failure
	RecordFailure(ctx context.Context, provider types.Provider, err error) error
	
	// GetFailureStats returns failure statistics
	GetFailureStats(ctx context.Context, provider types.Provider) (*types.FailureStats, error)
	
	// IsProviderHealthy checks if a provider is healthy
	IsProviderHealthy(ctx context.Context, provider types.Provider) bool
}

// LLMLoadBalancer defines the interface for load balancing LLM requests
type LLMLoadBalancer interface {
	// SelectProvider selects a provider based on load balancing strategy
	SelectProvider(ctx context.Context, request *types.LLMRequest) (types.Provider, error)
	
	// RecordLatency records latency for load balancing decisions
	RecordLatency(provider types.Provider, latency time.Duration) error
	
	// UpdateWeights updates provider weights for load balancing
	UpdateWeights(weights map[types.Provider]int) error
	
	// GetWeights returns current provider weights
	GetWeights() map[types.Provider]int
	
	// SetStrategy sets the load balancing strategy
	SetStrategy(strategy types.LoadBalancingStrategy) error
	
	// GetStrategy returns the current load balancing strategy
	GetStrategy() types.LoadBalancingStrategy
	
	// GetProviderLoad returns current load for each provider
	GetProviderLoad(ctx context.Context) (map[types.Provider]float64, error)
}

// LLMConfigManager defines the interface for managing LLM configurations
type LLMConfigManager interface {
	// GetProviderConfig returns configuration for a provider
	GetProviderConfig(ctx context.Context, provider types.Provider) (*types.ProviderConfig, error)
	
	// SetProviderConfig sets configuration for a provider
	SetProviderConfig(ctx context.Context, provider types.Provider, config *types.ProviderConfig) error
	
	// UpdateProviderConfig updates configuration for a provider
	UpdateProviderConfig(ctx context.Context, provider types.Provider, updates map[string]interface{}) error
	
	// DeleteProviderConfig deletes configuration for a provider
	DeleteProviderConfig(ctx context.Context, provider types.Provider) error
	
	// ListProviderConfigs lists all provider configurations
	ListProviderConfigs(ctx context.Context) (map[types.Provider]*types.ProviderConfig, error)
	
	// ValidateConfig validates a provider configuration
	ValidateConfig(ctx context.Context, config *types.ProviderConfig) error
	
	// ReloadConfig reloads configuration from source
	ReloadConfig(ctx context.Context) error
	
	// WatchConfig watches for configuration changes
	WatchConfig(ctx context.Context) (<-chan *types.ConfigChangeEvent, error)
}

// LLMAuditLogger defines the interface for auditing LLM operations
type LLMAuditLogger interface {
	// LogRequest logs an LLM request for audit purposes
	LogRequest(ctx context.Context, request *types.LLMRequest) error
	
	// LogResponse logs an LLM response for audit purposes
	LogResponse(ctx context.Context, response *types.LLMResponse) error
	
	// LogError logs an LLM error for audit purposes
	LogError(ctx context.Context, provider types.Provider, err error) error
	
	// GetAuditTrail returns audit trail for LLM operations
	GetAuditTrail(ctx context.Context, filter *types.AuditFilter) ([]*types.LLMAuditEvent, error)
	
	// ExportAuditLog exports audit log in specified format
	ExportAuditLog(ctx context.Context, timeRange types.TimeRange, format string) ([]byte, error)
	
	// SetRedactionRules sets rules for redacting sensitive data in audit logs
	SetRedactionRules(rules []*types.RedactionRule) error
	
	// GetRedactionRules returns current redaction rules
	GetRedactionRules() []*types.RedactionRule
}

// LLMContentFilter defines the interface for filtering LLM content
type LLMContentFilter interface {
	// FilterRequest filters request content before sending to LLM
	FilterRequest(ctx context.Context, request *types.LLMRequest) (*types.LLMRequest, error)
	
	// FilterResponse filters response content from LLM
	FilterResponse(ctx context.Context, response *types.LLMResponse) (*types.LLMResponse, error)
	
	// DetectSensitiveContent detects sensitive content in text
	DetectSensitiveContent(ctx context.Context, content string) ([]*types.SensitiveContentDetection, error)
	
	// RedactContent redacts sensitive content from text
	RedactContent(ctx context.Context, content string, detections []*types.SensitiveContentDetection) (string, error)
	
	// SetFilterRules sets content filtering rules
	SetFilterRules(rules []*types.ContentFilterRule) error
	
	// GetFilterRules returns current content filtering rules
	GetFilterRules() []*types.ContentFilterRule
	
	// ValidateContent validates content against filtering rules
	ValidateContent(ctx context.Context, content string) (*types.ContentValidation, error)
}