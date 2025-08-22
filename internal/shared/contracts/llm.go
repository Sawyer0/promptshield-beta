package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

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
