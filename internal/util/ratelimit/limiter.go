package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/shared/errors"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Limiter represents a rate limiter
type Limiter struct {
	config     *types.RateLimitConfig
	tokens     int
	lastRefill time.Time
	
	mu         sync.Mutex
	closed     bool
	metrics    *Metrics
}

// Metrics tracks rate limiter metrics
type Metrics struct {
	mu             sync.RWMutex
	Allowed        int64
	Denied         int64
	TotalRequests  int64
	CurrentTokens  int
	LastRefillTime time.Time
}

// New creates a new rate limiter
func New(config *types.RateLimitConfig) *Limiter {
	if config == nil {
		config = DefaultConfig()
	}

	return &Limiter{
		config:     config,
		tokens:     config.BurstSize,
		lastRefill: time.Now(),
		metrics:    &Metrics{CurrentTokens: config.BurstSize},
	}
}

// DefaultConfig returns default rate limiter configuration
func DefaultConfig() *types.RateLimitConfig {
	return &types.RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
		BackoffDelay:      100 * time.Millisecond,
	}
}

// Allow checks if a request is allowed
func (l *Limiter) Allow() bool {
	return l.AllowN(1)
}

// AllowN checks if n requests are allowed
func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return false
	}

	// Refill tokens
	l.refill()

	// Update metrics
	l.metrics.mu.Lock()
	l.metrics.TotalRequests += int64(n)
	l.metrics.mu.Unlock()

	// Check if we have enough tokens
	if l.tokens >= n {
		l.tokens -= n
		l.metrics.mu.Lock()
		l.metrics.Allowed += int64(n)
		l.metrics.CurrentTokens = l.tokens
		l.metrics.mu.Unlock()
		return true
	}

	// Not enough tokens
	l.metrics.mu.Lock()
	l.metrics.Denied += int64(n)
	l.metrics.mu.Unlock()
	return false
}

// Wait blocks until the request is allowed
func (l *Limiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// WaitN blocks until n requests are allowed
func (l *Limiter) WaitN(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}

	// Check if immediately available
	if l.AllowN(n) {
		return nil
	}

	// Calculate wait time
	waitTime := l.timeToWait(n)
	
	// Wait with backoff
	timer := time.NewTimer(waitTime)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if l.AllowN(n) {
				return nil
			}
			// Recalculate wait time
			waitTime = l.timeToWait(n)
			timer.Reset(waitTime)
			
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Reserve reserves n tokens
func (l *Limiter) Reserve(n int) *Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return &Reservation{
			ok:     false,
			limiter: l,
		}
	}

	l.refill()

	// Calculate when tokens will be available
	tokensNeeded := n - l.tokens
	if tokensNeeded <= 0 {
		// Tokens available now
		l.tokens -= n
		return &Reservation{
			ok:       true,
			limiter:  l,
			tokens:   n,
			timeToAct: time.Now(),
		}
	}

	// Calculate future availability
	secondsToWait := float64(tokensNeeded) / float64(l.config.RequestsPerSecond)
	timeToAct := time.Now().Add(time.Duration(secondsToWait * float64(time.Second)))

	return &Reservation{
		ok:       true,
		limiter:  l,
		tokens:   n,
		timeToAct: timeToAct,
	}
}

// refill adds tokens based on elapsed time
func (l *Limiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefill)
	
	// Calculate tokens to add
	tokensToAdd := int(elapsed.Seconds() * float64(l.config.RequestsPerSecond))
	
	if tokensToAdd > 0 {
		l.tokens += tokensToAdd
		if l.tokens > l.config.BurstSize {
			l.tokens = l.config.BurstSize
		}
		l.lastRefill = now
		
		l.metrics.mu.Lock()
		l.metrics.CurrentTokens = l.tokens
		l.metrics.LastRefillTime = now
		l.metrics.mu.Unlock()
	}
}

// timeToWait calculates how long to wait for n tokens
func (l *Limiter) timeToWait(n int) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= n {
		return 0
	}

	tokensNeeded := n - l.tokens
	secondsToWait := float64(tokensNeeded) / float64(l.config.RequestsPerSecond)
	
	waitTime := time.Duration(secondsToWait * float64(time.Second))
	if waitTime < l.config.BackoffDelay {
		waitTime = l.config.BackoffDelay
	}
	
	return waitTime
}

// Reset resets the rate limiter
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.tokens = l.config.BurstSize
	l.lastRefill = time.Now()
	
	l.metrics = &Metrics{
		CurrentTokens: l.config.BurstSize,
		LastRefillTime: l.lastRefill,
	}
}

// Close closes the rate limiter
func (l *Limiter) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true
	return nil
}

// GetMetrics returns current metrics
func (l *Limiter) GetMetrics() Metrics {
	l.metrics.mu.RLock()
	defer l.metrics.mu.RUnlock()
	return *l.metrics
}

// Tokens returns the current number of tokens
func (l *Limiter) Tokens() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.refill()
	return l.tokens
}

// Reservation represents a token reservation
type Reservation struct {
	ok        bool
	limiter   *Limiter
	tokens    int
	timeToAct time.Time
	canceled  bool
	mu        sync.Mutex
}

// OK returns whether the reservation is valid
func (r *Reservation) OK() bool {
	return r.ok
}

// Cancel cancels the reservation
func (r *Reservation) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.canceled || !r.ok {
		return
	}

	// Return tokens if not yet used
	if time.Now().Before(r.timeToAct) {
		r.limiter.mu.Lock()
		r.limiter.tokens += r.tokens
		if r.limiter.tokens > r.limiter.config.BurstSize {
			r.limiter.tokens = r.limiter.config.BurstSize
		}
		r.limiter.mu.Unlock()
	}

	r.canceled = true
}

// Delay returns how long to wait before acting on the reservation
func (r *Reservation) Delay() time.Duration {
	if !r.ok {
		return 0
	}
	
	delay := time.Until(r.timeToAct)
	if delay < 0 {
		return 0
	}
	return delay
}

// DelayFrom returns how long to wait from a specific time
func (r *Reservation) DelayFrom(t time.Time) time.Duration {
	if !r.ok {
		return 0
	}
	
	delay := r.timeToAct.Sub(t)
	if delay < 0 {
		return 0
	}
	return delay
}

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	capacity      int
	tokens        float64
	refillRate    float64
	lastRefill    time.Time
	mu            sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     float64(capacity),
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN checks if n requests are allowed
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

// refill adds tokens based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
	
	tb.lastRefill = now
}