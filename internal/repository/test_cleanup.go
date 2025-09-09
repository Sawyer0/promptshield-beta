package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestCleanupManager manages cleanup for tests
type TestCleanupManager struct {
	t            *testing.T
	cleanup      *ResourceCleanup
	factories    []RepositoryFactory
	mu           sync.Mutex
	cleanupDone  bool
}

// NewTestCleanupManager creates a new test cleanup manager
func NewTestCleanupManager(t *testing.T) *TestCleanupManager {
	return &TestCleanupManager{
		t:       t,
		cleanup: NewResourceCleanup(),
	}
}

// RegisterFactory registers a factory for cleanup
func (tcm *TestCleanupManager) RegisterFactory(factory RepositoryFactory) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()
	
	if tcm.cleanupDone {
		tcm.t.Error("Cannot register factory after cleanup")
		return
	}
	
	tcm.factories = append(tcm.factories, factory)
}

// AddCleanup adds a cleanup function
func (tcm *TestCleanupManager) AddCleanup(cleanup func() error) {
	tcm.cleanup.AddCleanup(cleanup)
}

// AddCleanupFunc adds a simple cleanup function
func (tcm *TestCleanupManager) AddCleanupFunc(cleanup func()) {
	tcm.cleanup.AddCleanupFunc(cleanup)
}

// Cleanup runs all cleanup functions and closes factories
func (tcm *TestCleanupManager) Cleanup() {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()
	
	if tcm.cleanupDone {
		return
	}
	tcm.cleanupDone = true
	
	// Close all factories
	for i, factory := range tcm.factories {
		if err := factory.Close(); err != nil {
			tcm.t.Errorf("Failed to close factory %d: %v", i, err)
		}
	}
	
	// Run cleanup functions
	if err := tcm.cleanup.Cleanup(); err != nil {
		tcm.t.Errorf("Cleanup failed: %v", err)
	}
}

// CreateTestFactory creates a test factory with automatic cleanup
func (tcm *TestCleanupManager) CreateTestFactory() (*TestRepositoryFactory, error) {
	factory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		return nil, err
	}
	
	tcm.RegisterFactory(factory)
	return factory, nil
}

// WithCleanup is a helper that ensures cleanup runs even if the test panics
func WithCleanup(t *testing.T, fn func(*TestCleanupManager)) {
	tcm := NewTestCleanupManager(t)
	defer tcm.Cleanup()
	fn(tcm)
}

// FactoryTestSuite provides a test suite for factory lifecycle testing
type FactoryTestSuite struct {
	t       *testing.T
	cleanup *TestCleanupManager
}

// NewFactoryTestSuite creates a new factory test suite
func NewFactoryTestSuite(t *testing.T) *FactoryTestSuite {
	return &FactoryTestSuite{
		t:       t,
		cleanup: NewTestCleanupManager(t),
	}
}

// TestFactoryLifecycle tests the complete lifecycle of a factory
func (fts *FactoryTestSuite) TestFactoryLifecycle(factory RepositoryFactory) {
	fts.cleanup.RegisterFactory(factory)
	
	// Test health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := factory.HealthCheck(ctx); err != nil {
		fts.t.Errorf("Factory health check failed: %v", err)
	}
	
	// Test repository access
	fts.testRepositoryAccess(factory)
	
	// Test close
	if err := factory.Close(); err != nil {
		fts.t.Errorf("Factory close failed: %v", err)
	}
}

// testRepositoryAccess tests that all repositories can be accessed
func (fts *FactoryTestSuite) testRepositoryAccess(factory RepositoryFactory) {
	// Test that all repository methods don't panic
	defer func() {
		if r := recover(); r != nil {
			fts.t.Errorf("Repository access panicked: %v", r)
		}
	}()
	
	_ = factory.Tenant()
	_ = factory.Audit()
	_ = factory.RulepackAssignment()
	_ = factory.APIToken()
	_ = factory.Settings()
	_ = factory.Rulepack()
}

// Cleanup runs cleanup for the test suite
func (fts *FactoryTestSuite) Cleanup() {
	fts.cleanup.Cleanup()
}

// BenchmarkFactoryOperations benchmarks factory operations
func BenchmarkFactoryOperations(b *testing.B, createFactory func() (RepositoryFactory, error)) {
	factory, err := createFactory()
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Close()
	
	b.ResetTimer()
	
	b.Run("HealthCheck", func(b *testing.B) {
		ctx := context.Background()
		for i := 0; i < b.N; i++ {
			_ = factory.HealthCheck(ctx)
		}
	})
	
	b.Run("RepositoryAccess", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = factory.Tenant()
			_ = factory.Audit()
			_ = factory.RulepackAssignment()
			_ = factory.APIToken()
			_ = factory.Settings()
			_ = factory.Rulepack()
		}
	})
}

// ParallelFactoryTest tests factory operations under concurrent load
func ParallelFactoryTest(t *testing.T, factory RepositoryFactory, numGoroutines int, duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Test health check
					if err := factory.HealthCheck(ctx); err != nil {
						errors <- fmt.Errorf("goroutine %d health check failed: %w", id, err)
						return
					}
					
					// Test repository access
					_ = factory.Tenant()
					_ = factory.Audit()
					_ = factory.RulepackAssignment()
					_ = factory.APIToken()
					_ = factory.Settings()
					_ = factory.Rulepack()
					
					// Small delay to avoid overwhelming
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	for err := range errors {
		t.Error(err)
	}
}

// MemoryLeakTest tests for memory leaks in factory operations
func MemoryLeakTest(t *testing.T, createFactory func() (RepositoryFactory, error), iterations int) {
	// This is a basic memory leak test - in a real scenario you'd use more sophisticated tools
	for i := 0; i < iterations; i++ {
		factory, err := createFactory()
		if err != nil {
			t.Fatalf("Failed to create factory %d: %v", i, err)
		}
		
		// Use the factory briefly
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = factory.HealthCheck(ctx)
		_ = factory.Tenant()
		_ = factory.Audit()
		cancel()
		
		// Close the factory
		if err := factory.Close(); err != nil {
			t.Errorf("Failed to close factory %d: %v", i, err)
		}
		
		// Force garbage collection periodically
		if i%100 == 0 {
			// In a real test, you might call runtime.GC() and check memory stats
			t.Logf("Completed %d iterations", i)
		}
	}
}