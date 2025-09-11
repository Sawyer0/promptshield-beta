package repository

import (
	"context"
	"testing"
	"time"
)

func TestProductionFactoryLifecycle(t *testing.T) {
	// Skip if no database available
	config := &RepositoryConfig{
		DatabaseURL: "", // No database for this test
		Environment: "production",
	}
	
	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer cm.Close()
	
	// This should fail because production requires PostgreSQL
	_, err = NewProductionRepositoryFactory(config, cm)
	if err == nil {
		t.Error("Expected production factory creation to fail without database")
	}
}

func TestDevelopmentFactoryLifecycle(t *testing.T) {
	WithCleanup(t, func(tcm *TestCleanupManager) {
		config := &RepositoryConfig{
			DatabaseURL: "", // No database for this test
			Environment: "development",
		}
		
		cm, err := NewConnectionManager(context.Background(), config)
		if err != nil {
			t.Fatalf("Failed to create connection manager: %v", err)
		}
		tcm.AddCleanup(func() error { return cm.Close() })
		
		// This should fail because development also requires PostgreSQL
		_, err = NewDevelopmentRepositoryFactory(config, cm)
		if err == nil {
			t.Error("Expected development factory creation to fail without database")
		}
	})
}

func TestTestFactoryLifecycle(t *testing.T) {
	WithCleanup(t, func(tcm *TestCleanupManager) {
		factory, err := tcm.CreateTestFactory()
		if err != nil {
			t.Fatalf("Failed to create test factory: %v", err)
		}
		
		// Test that factory works
		if factory.Tenant() == nil {
			t.Error("Expected tenant repository to be available")
		}
		
		if factory.Audit() == nil {
			t.Error("Expected audit repository to be available")
		}
		
		if factory.RulepackAssignment() == nil {
			t.Error("Expected assignment repository to be available")
		}
		
		if factory.APIToken() == nil {
			t.Error("Expected API token repository to be available")
		}
		
		if factory.Settings() == nil {
			t.Error("Expected settings repository to be available")
		}
		
		if factory.Rulepack() == nil {
			t.Error("Expected rulepack repository to be available")
		}
		
		// Test health check
		ctx := context.Background()
		if err := factory.HealthCheck(ctx); err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})
}

func TestTestFactoryCleanup(t *testing.T) {
	factory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	
	// Add cleanup functions
	var cleanupCalls []int
	factory.AddCleanup(func() error {
		cleanupCalls = append(cleanupCalls, 1)
		return nil
	})
	
	factory.AddCleanup(func() error {
		cleanupCalls = append(cleanupCalls, 2)
		return nil
	})
	
	// Close factory
	if err := factory.Close(); err != nil {
		t.Errorf("Factory close failed: %v", err)
	}
	
	// Verify cleanup functions were called in reverse order
	expected := []int{2, 1}
	if len(cleanupCalls) != len(expected) {
		t.Errorf("Expected %d cleanup calls, got %d", len(expected), len(cleanupCalls))
	}
	
	for i, call := range cleanupCalls {
		if call != expected[i] {
			t.Errorf("Expected cleanup call %d to be %d, got %d", i, expected[i], call)
		}
	}
	
	// Verify repositories are cleared
	if factory.Tenant() != nil {
		t.Error("Expected tenant repository to be nil after close")
	}
	
	if factory.Audit() != nil {
		t.Error("Expected audit repository to be nil after close")
	}
}

func TestConnectionManagerLifecycle(t *testing.T) {
	WithCleanup(t, func(tcm *TestCleanupManager) {
		config := &RepositoryConfig{
			DatabaseURL: "", // No database
			RedisAddr:   "", // No Redis
			Environment: "test",
		}
		
		cm, err := NewConnectionManager(context.Background(), config)
		if err != nil {
			t.Fatalf("Failed to create connection manager: %v", err)
		}
		tcm.AddCleanup(func() error { return cm.Close() })
		
		// Test health check
		ctx := context.Background()
		if err := cm.HealthCheck(ctx); err != nil {
			t.Errorf("Health check failed: %v", err)
		}
		
		// Test that close works
		if err := cm.Close(); err != nil {
			t.Errorf("Connection manager close failed: %v", err)
		}
	})
}

func TestFactoryTestSuite(t *testing.T) {
	suite := NewFactoryTestSuite(t)
	defer suite.Cleanup()
	
	// Create test factory
	factory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	
	// Test lifecycle
	suite.TestFactoryLifecycle(factory)
}

func TestParallelFactoryOperations(t *testing.T) {
	factory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	defer factory.Close()
	
	// Test parallel operations
	ParallelFactoryTest(t, factory, 5, 100*time.Millisecond)
}

func TestMemoryLeakPrevention(t *testing.T) {
	// Test that creating and closing many factories doesn't leak memory
	MemoryLeakTest(t, func() (RepositoryFactory, error) {
		return NewTestRepositoryFactory(nil, nil)
	}, 100)
}

func BenchmarkTestFactoryOperations(b *testing.B) {
	BenchmarkFactoryOperations(b, func() (RepositoryFactory, error) {
		return NewTestRepositoryFactory(nil, nil)
	})
}

func TestFactoryCloseIdempotency(t *testing.T) {
	factory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	
	// Close multiple times should not cause issues
	if err := factory.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}
	
	if err := factory.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}
	
	if err := factory.Close(); err != nil {
		t.Errorf("Third close failed: %v", err)
	}
}

func TestConnectionManagerCloseIdempotency(t *testing.T) {
	config := &RepositoryConfig{
		DatabaseURL: "",
		RedisAddr:   "",
		Environment: "test",
	}
	
	cm, err := NewConnectionManager(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	
	// Close multiple times should not cause issues
	if err := cm.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}
	
	if err := cm.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}
	
	if err := cm.Close(); err != nil {
		t.Errorf("Third close failed: %v", err)
	}
}