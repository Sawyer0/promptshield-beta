package api

import (
	"sync"

	"github.com/promptshield/promptshield/internal/usage"
	"golang.org/x/time/rate"
)

// InMemoryQuotaStore implements usage.QuotaStore for development and testing
type InMemoryQuotaStore struct {
	mu      sync.RWMutex
	limiters map[string]*rate.Limiter
}

func NewInMemoryQuotaStore() usage.QuotaStore {
	return &InMemoryQuotaStore{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (s *InMemoryQuotaStore) Allow(tenantID string) bool {
	s.mu.RLock()
	limiter, exists := s.limiters[tenantID]
	s.mu.RUnlock()
	
	if !exists {
		// No limit configured, allow by default
		return true
	}
	
	return limiter.Allow()
}

func (s *InMemoryQuotaStore) Set(tenantID string, limit rate.Limit, burst int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.limiters[tenantID] = rate.NewLimiter(limit, burst)
}

func (s *InMemoryQuotaStore) Get(tenantID string) (rate.Limit, int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	limiter, exists := s.limiters[tenantID]
	if !exists {
		return 0, 0, false
	}
	
	return limiter.Limit(), limiter.Burst(), true
}

func (s *InMemoryQuotaStore) Delete(tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.limiters, tenantID)
}