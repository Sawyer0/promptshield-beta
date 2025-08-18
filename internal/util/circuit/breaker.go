package circuit

import (
	"context"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/shared/errors"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Breaker represents a circuit breaker
type Breaker struct {
	config *types.CircuitBreakerConfig
	state  types.CircuitBreakerState
	stats  *types.CircuitBreakerStats

	mu              sync.RWMutex
	lastStateChange time.Time
	failures        []time.Time
}

// New creates a new circuit breaker
func New(config *types.CircuitBreakerConfig) *Breaker {
	if config == nil {
		config = DefaultConfig()
	}

	return &Breaker{
		config:          config,
		state:           types.CircuitBreakerStateClosed,
		lastStateChange: time.Now(),
		stats: &types.CircuitBreakerStats{
			State:           types.CircuitBreakerStateClosed,
			LastStateChange: time.Now(),
		},
		failures: make([]time.Time, 0, config.FailureThreshold),
	}
}

// DefaultConfig returns default circuit breaker configuration
func DefaultConfig() *types.CircuitBreakerConfig {
	return &types.CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		Timeout:          10 * time.Second,
		MaxRequests:      100,
		Interval:         60 * time.Second,
	}
}

// Execute runs a function through the circuit breaker
func (b *Breaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if !b.canExecute() {
		return errors.ErrCircuitOpen
	}

	// Create timeout context if configured
	var cancel context.CancelFunc
	if b.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, b.config.Timeout)
		defer cancel()
	}

	// Execute the function
	err := fn(ctx)

	// Update state based on result
	b.recordResult(err)

	return err
}

// canExecute checks if a request can be executed
func (b *Breaker) canExecute() bool {
	b.mu.RLock()
	state := b.state
	lastChange := b.lastStateChange
	b.mu.RUnlock()

	switch state {
	case types.CircuitBreakerStateClosed:
		return true
	case types.CircuitBreakerStateOpen:
		// Check if recovery timeout has passed
		if time.Since(lastChange) > b.config.RecoveryTimeout {
			b.transition(types.CircuitBreakerStateHalfOpen)
			return true
		}
		return false
	case types.CircuitBreakerStateHalfOpen:
		// Allow limited requests in half-open state
		b.mu.RLock()
		requests := b.stats.Requests
		b.mu.RUnlock()
		return requests < int64(b.config.MaxRequests)
	default:
		return false
	}
}

// recordResult records the result of an execution
func (b *Breaker) recordResult(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats.Requests++

	if err != nil {
		b.stats.Failures++
		b.stats.ConsecutiveFailures++
		b.recordFailure()
	} else {
		b.stats.Successes++
		b.stats.ConsecutiveFailures = 0

		// Transition to closed if in half-open state
		if b.state == types.CircuitBreakerStateHalfOpen {
			b.transitionInternal(types.CircuitBreakerStateClosed)
		}
	}
}

// recordFailure records a failure and checks if circuit should open
func (b *Breaker) recordFailure() {
	now := time.Now()
	b.failures = append(b.failures, now)

	// Remove old failures outside the interval window
	cutoff := now.Add(-b.config.Interval)
	validFailures := make([]time.Time, 0, len(b.failures))
	for _, t := range b.failures {
		if t.After(cutoff) {
			validFailures = append(validFailures, t)
		}
	}
	b.failures = validFailures

	// Check if threshold is exceeded
	if len(b.failures) >= b.config.FailureThreshold {
		if b.state != types.CircuitBreakerStateOpen {
			b.transitionInternal(types.CircuitBreakerStateOpen)
		}
	}
}

// transition changes the circuit breaker state
func (b *Breaker) transition(newState types.CircuitBreakerState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionInternal(newState)
}

// transitionInternal changes state without locking (must be called with lock held)
func (b *Breaker) transitionInternal(newState types.CircuitBreakerState) {
	if b.state != newState {
		b.state = newState
		b.lastStateChange = time.Now()
		b.stats.State = newState
		b.stats.LastStateChange = b.lastStateChange

		// Reset consecutive failures when closing
		if newState == types.CircuitBreakerStateClosed {
			b.stats.ConsecutiveFailures = 0
			b.failures = b.failures[:0]
		}
	}
}

// State returns the current state of the circuit breaker
func (b *Breaker) State() types.CircuitBreakerState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Stats returns the current statistics
func (b *Breaker) Stats() types.CircuitBreakerStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return *b.stats
}

// Reset resets the circuit breaker to closed state
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.state = types.CircuitBreakerStateClosed
	b.lastStateChange = time.Now()
	b.failures = b.failures[:0]
	b.stats = &types.CircuitBreakerStats{
		State:           types.CircuitBreakerStateClosed,
		LastStateChange: b.lastStateChange,
	}
}

// IsOpen returns true if the circuit is open
func (b *Breaker) IsOpen() bool {
	return b.State() == types.CircuitBreakerStateOpen
}

// IsClosed returns true if the circuit is closed
func (b *Breaker) IsClosed() bool {
	return b.State() == types.CircuitBreakerStateClosed
}

// IsHalfOpen returns true if the circuit is half-open
func (b *Breaker) IsHalfOpen() bool {
	return b.State() == types.CircuitBreakerStateHalfOpen
}