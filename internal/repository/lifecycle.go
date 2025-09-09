package repository

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// LifecycleManager manages the lifecycle of repository factories and their resources
type LifecycleManager struct {
	mu        sync.RWMutex
	factories map[string]RepositoryFactory
	logger    *slog.Logger
	
	// Shutdown configuration
	shutdownTimeout time.Duration
	gracefulPeriod  time.Duration
	
	// State tracking
	isShuttingDown bool
	shutdownChan   chan struct{}
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		factories:       make(map[string]RepositoryFactory),
		logger:          slog.With("component", "lifecycle-manager"),
		shutdownTimeout: 30 * time.Second,
		gracefulPeriod:  5 * time.Second,
		shutdownChan:    make(chan struct{}),
	}
}

// RegisterFactory registers a factory for lifecycle management
func (lm *LifecycleManager) RegisterFactory(name string, factory RepositoryFactory) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if lm.isShuttingDown {
		lm.logger.Warn("Cannot register factory during shutdown", "name", name)
		return
	}
	
	lm.factories[name] = factory
	lm.logger.Info("Factory registered for lifecycle management", "name", name)
}

// UnregisterFactory removes a factory from lifecycle management
func (lm *LifecycleManager) UnregisterFactory(name string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	delete(lm.factories, name)
	lm.logger.Info("Factory unregistered from lifecycle management", "name", name)
}

// GetFactory retrieves a registered factory
func (lm *LifecycleManager) GetFactory(name string) (RepositoryFactory, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	factory, exists := lm.factories[name]
	return factory, exists
}

// HealthCheck performs health checks on all registered factories
func (lm *LifecycleManager) HealthCheck(ctx context.Context) error {
	lm.mu.RLock()
	factories := make(map[string]RepositoryFactory)
	for name, factory := range lm.factories {
		factories[name] = factory
	}
	lm.mu.RUnlock()
	
	var errors []error
	for name, factory := range factories {
		if err := factory.HealthCheck(ctx); err != nil {
			lm.logger.Error("Factory health check failed", "name", name, "error", err)
			errors = append(errors, fmt.Errorf("factory %s: %w", name, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("health check failures: %v", errors)
	}
	
	return nil
}

// Shutdown gracefully shuts down all registered factories
func (lm *LifecycleManager) Shutdown(ctx context.Context) error {
	lm.mu.Lock()
	if lm.isShuttingDown {
		lm.mu.Unlock()
		return fmt.Errorf("shutdown already in progress")
	}
	lm.isShuttingDown = true
	close(lm.shutdownChan)
	
	// Create a copy of factories to avoid holding the lock during shutdown
	factories := make(map[string]RepositoryFactory)
	for name, factory := range lm.factories {
		factories[name] = factory
	}
	lm.mu.Unlock()
	
	lm.logger.Info("Starting graceful shutdown", "factory_count", len(factories))
	
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, lm.shutdownTimeout)
	defer cancel()
	
	// Wait for graceful period before starting shutdown
	if lm.gracefulPeriod > 0 {
		lm.logger.Info("Waiting for graceful period", "duration", lm.gracefulPeriod)
		select {
		case <-time.After(lm.gracefulPeriod):
		case <-shutdownCtx.Done():
			lm.logger.Warn("Shutdown context cancelled during graceful period")
		}
	}
	
	// Shutdown all factories concurrently
	var wg sync.WaitGroup
	errorChan := make(chan error, len(factories))
	
	for name, factory := range factories {
		wg.Add(1)
		go func(name string, factory RepositoryFactory) {
			defer wg.Done()
			
			lm.logger.Info("Shutting down factory", "name", name)
			if err := factory.Close(); err != nil {
				lm.logger.Error("Factory shutdown failed", "name", name, "error", err)
				errorChan <- fmt.Errorf("factory %s: %w", name, err)
			} else {
				lm.logger.Info("Factory shutdown completed", "name", name)
			}
		}(name, factory)
	}
	
	// Wait for all shutdowns to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		lm.logger.Info("All factories shut down successfully")
	case <-shutdownCtx.Done():
		lm.logger.Error("Shutdown timeout exceeded", "timeout", lm.shutdownTimeout)
		return fmt.Errorf("shutdown timeout exceeded")
	}
	
	// Collect any errors
	close(errorChan)
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}
	
	lm.logger.Info("Lifecycle manager shutdown completed")
	return nil
}

// IsShuttingDown returns true if shutdown is in progress
func (lm *LifecycleManager) IsShuttingDown() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.isShuttingDown
}

// ShutdownChan returns a channel that is closed when shutdown begins
func (lm *LifecycleManager) ShutdownChan() <-chan struct{} {
	return lm.shutdownChan
}

// SetShutdownTimeout configures the shutdown timeout
func (lm *LifecycleManager) SetShutdownTimeout(timeout time.Duration) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.shutdownTimeout = timeout
}

// SetGracefulPeriod configures the graceful shutdown period
func (lm *LifecycleManager) SetGracefulPeriod(period time.Duration) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.gracefulPeriod = period
}

// GetStats returns statistics about managed factories
func (lm *LifecycleManager) GetStats() LifecycleStats {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	return LifecycleStats{
		RegisteredFactories: len(lm.factories),
		IsShuttingDown:      lm.isShuttingDown,
		ShutdownTimeout:     lm.shutdownTimeout,
		GracefulPeriod:      lm.gracefulPeriod,
	}
}

// LifecycleStats provides statistics about the lifecycle manager
type LifecycleStats struct {
	RegisteredFactories int           `json:"registered_factories"`
	IsShuttingDown      bool          `json:"is_shutting_down"`
	ShutdownTimeout     time.Duration `json:"shutdown_timeout"`
	GracefulPeriod      time.Duration `json:"graceful_period"`
}

// ResourceCleanup provides utilities for cleaning up resources in tests and applications
type ResourceCleanup struct {
	mu           sync.Mutex
	cleanupFuncs []func() error
	logger       *slog.Logger
}

// NewResourceCleanup creates a new resource cleanup utility
func NewResourceCleanup() *ResourceCleanup {
	return &ResourceCleanup{
		logger: slog.With("component", "resource-cleanup"),
	}
}

// AddCleanup adds a cleanup function to be called during cleanup
func (rc *ResourceCleanup) AddCleanup(cleanup func() error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cleanupFuncs = append(rc.cleanupFuncs, cleanup)
}

// AddCleanupFunc adds a simple cleanup function (no error return)
func (rc *ResourceCleanup) AddCleanupFunc(cleanup func()) {
	rc.AddCleanup(func() error {
		cleanup()
		return nil
	})
}

// Cleanup runs all registered cleanup functions
func (rc *ResourceCleanup) Cleanup() error {
	rc.mu.Lock()
	funcs := make([]func() error, len(rc.cleanupFuncs))
	copy(funcs, rc.cleanupFuncs)
	rc.mu.Unlock()
	
	var errors []error
	for i := len(funcs) - 1; i >= 0; i-- { // Run in reverse order
		if err := funcs[i](); err != nil {
			rc.logger.Error("Cleanup function failed", "error", err)
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}
	
	return nil
}

// Reset clears all cleanup functions
func (rc *ResourceCleanup) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cleanupFuncs = nil
}

// Count returns the number of registered cleanup functions
func (rc *ResourceCleanup) Count() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return len(rc.cleanupFuncs)
}