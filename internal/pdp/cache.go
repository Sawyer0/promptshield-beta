package pdp

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type cacheKey struct{
	Subject string
	Action string
	Resource string
	AttrsHash string
	PolicyEpoch string
}

type cacheEntry struct{
	Resp Response
	ExpiresAt time.Time
}

type TTLCache struct{
	mu sync.RWMutex
	data map[cacheKey]cacheEntry
	ttl time.Duration
	max int
}

func NewTTLCache(ttl time.Duration, max int) *TTLCache{
	if ttl <= 0 { ttl = 1500 * time.Millisecond }
	if max <= 0 { max = 10000 }
	return &TTLCache{ data: make(map[cacheKey]cacheEntry), ttl: ttl, max: max }
}

func hashAttrs(m map[string]any) string {
	if len(m) == 0 { return "" }
	b, _ := json.Marshal(m)
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h[:])
}

func keyFrom(req Request, epoch string) cacheKey{
	resID := req.Resource.ID
	if resID == "" && len(req.Resource.Tags) > 0 { resID = fmt.Sprintf("tags:%v", req.Resource.Tags) }
	return cacheKey{
		Subject: req.Subject.UserID+"|"+req.Subject.TenantID,
		Action: req.Action,
		Resource: req.Resource.Type+"|"+resID,
		AttrsHash: hashAttrs(req.Resource.Attributes),
		PolicyEpoch: epoch,
	}
}

func (c *TTLCache) Get(k cacheKey) (Response, bool){
	c.mu.RLock(); defer c.mu.RUnlock()
	if e, ok := c.data[k]; ok {
		if time.Now().Before(e.ExpiresAt) { return e.Resp, true }
	}
	return Response{}, false
}

func (c *TTLCache) Set(k cacheKey, v Response){
	c.mu.Lock(); defer c.mu.Unlock()
	if len(c.data) >= c.max {
		// naive eviction: first key deletion
		for kk := range c.data { delete(c.data, kk); break }
	}
	c.data[k] = cacheEntry{Resp: v, ExpiresAt: time.Now().Add(c.ttl)}
}

