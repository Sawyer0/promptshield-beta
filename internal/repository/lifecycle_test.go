package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/contracts"
	"github.com/promptshield/promptshield/internal/domain"
	sharedcontracts "github.com/promptshield/promptshield/internal/shared/contracts"
)

func TestLifecycleManager(t *testing.T) {
	lm := NewLifecycleManager()
	
	// Test initial state
	if lm.IsShuttingDown() {
		t.Error("Expected lifecycle manager to not be shutting down initially")
	}
	
	stats := lm.GetStats()
	if stats.RegisteredFactories != 0 {
		t.Errorf("Expected 0 registered factories, got %d", stats.RegisteredFactories)
	}
}

func TestLifecycleManagerRegisterUnregister(t *testing.T) {
	lm := NewLifecycleManager()
	
	// Create test factory
	factory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	defer factory.Close()
	
	// Register factory
	lm.RegisterFactory("test", factory)
	
	stats := lm.GetStats()
	if stats.RegisteredFactories != 1 {
		t.Errorf("Expected 1 registered factory, got %d", stats.RegisteredFactories)
	}
	
	// Get factory
	retrievedFactory, exists := lm.GetFactory("test")
	if !exists {
		t.Error("Expected factory to exist")
	}
	if retrievedFactory != factory {
		t.Error("Retrieved factory doesn't match registered factory")
	}
	
	// Unregister factory
	lm.UnregisterFactory("test")
	
	stats = lm.GetStats()
	if stats.RegisteredFactories != 0 {
		t.Errorf("Expected 0 registered factories after unregister, got %d", stats.RegisteredFactories)
	}
	
	// Try to get unregistered factory
	_, exists = lm.GetFactory("test")
	if exists {
		t.Error("Expected factory to not exist after unregister")
	}
}

func TestLifecycleManagerHealthCheck(t *testing.T) {
	lm := NewLifecycleManager()
	
	// Create test factories
	factory1, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory 1: %v", err)
	}
	defer factory1.Close()
	
	factory2, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory 2: %v", err)
	}
	defer factory2.Close()
	
	// Register factories
	lm.RegisterFactory("test1", factory1)
	lm.RegisterFactory("test2", factory2)
	
	// Test health check
	ctx := context.Background()
	if err := lm.HealthCheck(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestLifecycleManagerShutdown(t *testing.T) {
	lm := NewLifecycleManager()
	lm.SetShutdownTimeout(5 * time.Second)
	lm.SetGracefulPeriod(100 * time.Millisecond)
	
	// Create test factories
	factory1, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory 1: %v", err)
	}
	
	factory2, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory 2: %v", err)
	}
	
	// Register factories
	lm.RegisterFactory("test1", factory1)
	lm.RegisterFactory("test2", factory2)
	
	// Test shutdown
	ctx := context.Background()
	if err := lm.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	
	// Verify shutdown state
	if !lm.IsShuttingDown() {
		t.Error("Expected lifecycle manager to be shutting down")
	}
	
	// Test that shutdown channel is closed
	select {
	case <-lm.ShutdownChan():
		// Expected
	default:
		t.Error("Expected shutdown channel to be closed")
	}
	
	// Test that second shutdown returns error
	if err := lm.Shutdown(ctx); err == nil {
		t.Error("Expected second shutdown to return error")
	}
}

func TestLifecycleManagerShutdownTimeout(t *testing.T) {
	lm := NewLifecycleManager()
	lm.SetShutdownTimeout(100 * time.Millisecond)
	lm.SetGracefulPeriod(0) // No graceful period for this test
	
	// Create a mock factory that takes too long to close
	slowFactory := &slowCloseFactory{delay: 200 * time.Millisecond}
	lm.RegisterFactory("slow", slowFactory)
	
	// Test shutdown with timeout
	ctx := context.Background()
	start := time.Now()
	err := lm.Shutdown(ctx)
	duration := time.Since(start)
	
	// Should timeout
	if err == nil {
		t.Error("Expected shutdown to timeout")
	}
	
	// Should complete within reasonable time of the timeout
	if duration > 300*time.Millisecond {
		t.Errorf("Shutdown took too long: %v", duration)
	}
}

func TestResourceCleanup(t *testing.T) {
	rc := NewResourceCleanup()
	
	// Test initial state
	if rc.Count() != 0 {
		t.Errorf("Expected 0 cleanup functions, got %d", rc.Count())
	}
	
	// Add cleanup functions
	var calls []int
	rc.AddCleanup(func() error {
		calls = append(calls, 1)
		return nil
	})
	
	rc.AddCleanupFunc(func() {
		calls = append(calls, 2)
	})
	
	rc.AddCleanup(func() error {
		calls = append(calls, 3)
		return nil
	})
	
	if rc.Count() != 3 {
		t.Errorf("Expected 3 cleanup functions, got %d", rc.Count())
	}
	
	// Run cleanup
	if err := rc.Cleanup(); err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
	
	// Verify cleanup functions were called in reverse order (LIFO)
	expected := []int{3, 2, 1}
	if len(calls) != len(expected) {
		t.Errorf("Expected %d calls, got %d", len(expected), len(calls))
	}
	
	for i, call := range calls {
		if call != expected[i] {
			t.Errorf("Expected call %d to be %d, got %d", i, expected[i], call)
		}
	}
}

func TestResourceCleanupWithErrors(t *testing.T) {
	rc := NewResourceCleanup()
	
	// Add cleanup functions with errors
	rc.AddCleanup(func() error {
		return errors.New("error 1")
	})
	
	rc.AddCleanup(func() error {
		return nil // Success
	})
	
	rc.AddCleanup(func() error {
		return errors.New("error 3")
	})
	
	// Run cleanup
	err := rc.Cleanup()
	if err == nil {
		t.Error("Expected cleanup to return error")
	}
	
	// Should contain both errors
	errStr := err.Error()
	if !contains(errStr, "error 1") {
		t.Error("Expected error to contain 'error 1'")
	}
	if !contains(errStr, "error 3") {
		t.Error("Expected error to contain 'error 3'")
	}
}

func TestResourceCleanupReset(t *testing.T) {
	rc := NewResourceCleanup()
	
	// Add cleanup functions
	rc.AddCleanupFunc(func() {})
	rc.AddCleanupFunc(func() {})
	
	if rc.Count() != 2 {
		t.Errorf("Expected 2 cleanup functions, got %d", rc.Count())
	}
	
	// Reset
	rc.Reset()
	
	if rc.Count() != 0 {
		t.Errorf("Expected 0 cleanup functions after reset, got %d", rc.Count())
	}
	
	// Cleanup should not fail
	if err := rc.Cleanup(); err != nil {
		t.Errorf("Cleanup after reset failed: %v", err)
	}
}

func TestLifecycleManagerConcurrency(t *testing.T) {
	lm := NewLifecycleManager()
	
	// Test concurrent registration and health checks
	var wg sync.WaitGroup
	numGoroutines := 10
	
	// Start goroutines that register factories and run health checks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Create and register factory
			factory, err := NewTestRepositoryFactory(nil, nil)
			if err != nil {
				t.Errorf("Failed to create factory %d: %v", id, err)
				return
			}
			defer factory.Close()
			
			factoryName := fmt.Sprintf("test%d", id)
			lm.RegisterFactory(factoryName, factory)
			
			// Run health check
			ctx := context.Background()
			if err := lm.HealthCheck(ctx); err != nil {
				t.Errorf("Health check failed for goroutine %d: %v", id, err)
			}
			
			// Unregister factory
			lm.UnregisterFactory(factoryName)
		}(i)
	}
	
	wg.Wait()
	
	// Final state should be clean
	stats := lm.GetStats()
	if stats.RegisteredFactories != 0 {
		t.Errorf("Expected 0 registered factories at end, got %d", stats.RegisteredFactories)
	}
}

// Mock factory for testing slow close
type slowCloseFactory struct {
	delay time.Duration
}

func (f *slowCloseFactory) Tenant() domain.TenantRepository                { return nil }
func (f *slowCloseFactory) Audit() domain.AuditRepository                 { return nil }
func (f *slowCloseFactory) RulepackAssignment() domain.RulepackAssignmentRepository    { return nil }
func (f *slowCloseFactory) Policy() sharedcontracts.PolicyRepository        { return nil }
func (f *slowCloseFactory) APIToken() domain.APITokenRepository              { return nil }
func (f *slowCloseFactory) Settings() domain.SettingsRepository              { return nil }
func (f *slowCloseFactory) Rulepack() contracts.RulepackRepository              { return nil }
func (f *slowCloseFactory) HealthCheck(ctx context.Context) error { return nil }

func (f *slowCloseFactory) Close() error {
	time.Sleep(f.delay)
	return nil
}

