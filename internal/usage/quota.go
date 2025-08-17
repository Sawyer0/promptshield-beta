package usage

import (
    "sync"
    "time"

    "golang.org/x/time/rate"
)

// QuotaStore defines a minimal per-tenant rate limiter.
type QuotaStore interface {
    Allow(tenant string) bool
    Set(tenant string, r rate.Limit, burst int)
}

type inMemoryQuota struct {
    mu   sync.RWMutex
    rps  rate.Limit
    burst int
    m    map[string]*rate.Limiter
}

// NewInMemoryQuota creates a shared limiter store. Default rps/burst apply to new tenants.
func NewInMemoryQuota(defaultRPS float64, defaultBurst int) QuotaStore {
    if defaultBurst <= 0 { defaultBurst = 1 }
    return &inMemoryQuota{rps: rate.Limit(defaultRPS), burst: defaultBurst, m: make(map[string]*rate.Limiter)}
}

func (q *inMemoryQuota) get(tenant string) *rate.Limiter {
    q.mu.RLock()
    l := q.m[tenant]
    q.mu.RUnlock()
    if l != nil { return l }
    q.mu.Lock()
    defer q.mu.Unlock()
    if l = q.m[tenant]; l == nil {
        if q.rps <= 0 { q.rps = rate.Inf }
        l = rate.NewLimiter(q.rps, q.burst)
        // warm tokens
        l.AllowN(time.Now(), q.burst)
        q.m[tenant] = l
    }
    return l
}

func (q *inMemoryQuota) Allow(tenant string) bool {
    return q.get(tenant).Allow()
}

func (q *inMemoryQuota) Set(tenant string, r rate.Limit, burst int) {
    if burst <= 0 { burst = 1 }
    q.mu.Lock()
    q.m[tenant] = rate.NewLimiter(r, burst)
    q.mu.Unlock()
}


