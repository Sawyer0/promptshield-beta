package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// Retryable represents a function that can be retried
type Retryable func(ctx context.Context) error

// IsRetryable determines if an error should trigger a retry
type IsRetryable func(error) bool

// Retry executes a function with retry logic based on the provided policy
func Retry(ctx context.Context, policy *types.RetryPolicy, fn Retryable, isRetryable IsRetryable) error {
	if policy == nil {
		return fn(ctx)
	}

	var lastErr error
	delay := policy.InitialDelay

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry canceled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		if !isRetryable(lastErr) {
			return lastErr
		}

		if attempt < policy.MaxRetries {
			delay = calculateNextDelay(delay, policy)
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// WithJitter adds jitter to a duration
func WithJitter(d time.Duration, jitterFactor float64) time.Duration {
	if jitterFactor <= 0 {
		return d
	}
	jitter := time.Duration(rand.Float64() * float64(d) * jitterFactor)
	return d + jitter
}

// calculateNextDelay calculates the next retry delay
func calculateNextDelay(currentDelay time.Duration, policy *types.RetryPolicy) time.Duration {
	nextDelay := time.Duration(float64(currentDelay) * policy.BackoffFactor)
	if nextDelay > policy.MaxDelay {
		nextDelay = policy.MaxDelay
	}
	return WithJitter(nextDelay, 0.1) // Add 10% jitter
}

// ExponentialBackoff creates an exponential backoff retry policy
func ExponentialBackoff(maxRetries int, initialDelay, maxDelay time.Duration) *types.RetryPolicy {
	return &types.RetryPolicy{
		MaxRetries:    maxRetries,
		InitialDelay:  initialDelay,
		MaxDelay:      maxDelay,
		BackoffFactor: 2.0,
	}
}

// LinearBackoff creates a linear backoff retry policy
func LinearBackoff(maxRetries int, delay time.Duration) *types.RetryPolicy {
	return &types.RetryPolicy{
		MaxRetries:    maxRetries,
		InitialDelay:  delay,
		MaxDelay:      delay * time.Duration(maxRetries),
		BackoffFactor: 1.0,
	}
}

// FixedDelay creates a fixed delay retry policy
func FixedDelay(maxRetries int, delay time.Duration) *types.RetryPolicy {
	return &types.RetryPolicy{
		MaxRetries:    maxRetries,
		InitialDelay:  delay,
		MaxDelay:      delay,
		BackoffFactor: 1.0,
	}
}

// RetryWithBackoff is a simplified retry function with exponential backoff
func RetryWithBackoff(ctx context.Context, maxRetries int, fn Retryable) error {
	policy := ExponentialBackoff(maxRetries, 100*time.Millisecond, 30*time.Second)
	return Retry(ctx, policy, fn, func(err error) bool { return true })
}

// AlwaysRetry is a retry predicate that always returns true
func AlwaysRetry(err error) bool {
	return err != nil
}

// NeverRetry is a retry predicate that always returns false  
func NeverRetry(err error) bool {
	return false
}

// MaxAttemptsReached checks if max retry attempts have been reached
func MaxAttemptsReached(attempt, maxRetries int) bool {
	return attempt >= maxRetries
}

// CalculateBackoff calculates exponential backoff duration
func CalculateBackoff(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}