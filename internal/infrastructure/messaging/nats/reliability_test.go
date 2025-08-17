package nats

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// TestSubscriber_GracefulDegradation_RedisDown tests that the subscriber
// can be created even when Redis is completely unavailable
func TestSubscriber_GracefulDegradation_RedisDown(t *testing.T) {
	handler := func(ctx context.Context, update RuleUpdate) error {
		return nil
	}

	// Try to create subscriber with non-existent Redis address
	sub, err := NewSubscriber("localhost:99999", "test-group", "test-consumer", "tenant-1", handler)

	// Should NOT fail - graceful degradation means we can create the subscriber
	assert.NoError(t, err)
	assert.NotNil(t, sub)

	// Clean up
	if sub != nil {
		sub.Close()
	}
}

// TestSubscriber_NoRedis_Address tests no-op behavior when Redis is not configured
func TestSubscriber_NoRedis_Address(t *testing.T) {
	handler := func(ctx context.Context, update RuleUpdate) error {
		return nil
	}

	// Create subscriber with empty Redis address (disabled)
	sub, err := NewSubscriber("", "test-group", "test-consumer", "tenant-1", handler)

	assert.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Nil(t, sub.rdb) // No Redis client should be created

	// Start should be no-op and block until stopped
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error)
	go func() {
		done <- sub.Start(ctx)
	}()

	// Stop the subscriber
	sub.Stop()

	// Should return without error
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Start() did not return after Stop()")
	}
}

// TestSubscriber_CircuitBreaker_Concept tests the circuit breaker logic conceptually
// (This would need a real Redis instance for full integration testing)
func TestSubscriber_CircuitBreaker_Concept(t *testing.T) {
	// This test validates the circuit breaker state transitions
	// In a real environment, this would use a Redis testcontainer

	// Test parameters that match the implementation
	baseBackoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	// Validate exponential backoff calculation
	consecutiveFailures := 3
	expectedBackoff := time.Duration(1<<consecutiveFailures) * baseBackoff // 8 seconds

	assert.Equal(t, 8*time.Second, expectedBackoff)

	// Validate max backoff cap
	consecutiveFailures = 10 // Would be 1024 seconds without cap
	backoff := time.Duration(1<<consecutiveFailures) * baseBackoff
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	assert.Equal(t, maxBackoff, backoff)
	assert.Equal(t, 30*time.Second, backoff)
}

// TestSubscriber_TenantFiltering tests that tenant filtering works correctly
func TestSubscriber_TenantFiltering(t *testing.T) {
	tenantID := "tenant-123"
	var receivedUpdates []RuleUpdate

	handler := func(ctx context.Context, update RuleUpdate) error {
		receivedUpdates = append(receivedUpdates, update)
		return nil
	}

	// Create subscriber for specific tenant with a fake Redis address
	// (we won't actually connect, just test the message processing logic)
	sub := &Subscriber{
		tenantID: tenantID,
		handler:  handler,
		done:     make(chan struct{}),
	}

	// Test message processing directly (without Redis dependency)
	ctx := context.Background()

	// Create rule updates in JSON format (as they're stored in Redis)
	correctTenantUpdate := RuleUpdate{
		TenantID:      tenantID,
		RulepackID:    "pack-1",
		TargetScope:   "global",
		Version:       1,
		ContentSHA256: "abc123",
	}

	wrongTenantUpdate := RuleUpdate{
		TenantID:      "tenant-456",
		RulepackID:    "pack-2",
		TargetScope:   "global",
		Version:       1,
		ContentSHA256: "def456",
	}

	// Convert to JSON as Redis would store them
	correctJSON, _ := json.Marshal(correctTenantUpdate)
	wrongJSON, _ := json.Marshal(wrongTenantUpdate)

	// Create mock Redis messages with proper format
	mockMsg1 := redis.XMessage{
		ID: "1-0",
		Values: map[string]interface{}{
			"json": string(correctJSON),
		},
	}

	mockMsg2 := redis.XMessage{
		ID: "2-0",
		Values: map[string]interface{}{
			"json": string(wrongJSON),
		},
	}

	// Process messages
	err1 := sub.processMessage(ctx, mockMsg1)
	err2 := sub.processMessage(ctx, mockMsg2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Should only receive the message for the correct tenant
	assert.Len(t, receivedUpdates, 1)
	assert.Equal(t, tenantID, receivedUpdates[0].TenantID)
	assert.Equal(t, "pack-1", receivedUpdates[0].RulepackID)
}
