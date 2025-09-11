package pdp

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/observability/metrics"
)

// cachedClient decorates a pdp.Client with TTL cache and metrics

type cachedClient struct{
	inner Client
	cache *TTLCache
	policyEpoch string
}

func NewCached(inner Client) Client{
	ttl := 1500 * time.Millisecond
	if v := strings.TrimSpace(os.Getenv("PS_PDP_CACHE_TTL_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 { ttl = time.Duration(n) * time.Millisecond }
	}
	max := 10000
	if v := strings.TrimSpace(os.Getenv("PS_PDP_CACHE_MAX_ENTRIES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 { max = n }
	}
	return &cachedClient{ inner: inner, cache: NewTTLCache(ttl, max), policyEpoch: strings.TrimSpace(os.Getenv("PS_PDP_POLICY_EPOCH")) }
}

func (c *cachedClient) Evaluate(ctx context.Context, req Request) (Response, error) {
	k := keyFrom(req, c.policyEpoch)
	if resp, ok := c.cache.Get(k); ok {
		if metrics.Enabled() { metrics.CacheOperations.WithLabelValues("hit", "pdp").Inc() }
		return resp, nil
	}
	if metrics.Enabled() { metrics.CacheOperations.WithLabelValues("miss", "pdp").Inc() }
	resp, err := c.inner.Evaluate(ctx, req)
	if err == nil && resp.Cacheable {
		c.cache.Set(k, resp)
	}
	return resp, err
}

