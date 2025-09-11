package repository

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestRepositoryMetrics(t *testing.T) {
	metrics := NewRepositoryMetrics()
	
	// Test connection recording
	metrics.RecordConnection(true)
	metrics.RecordConnection(false)
	
	snapshot := metrics.GetSnapshot()
	if snapshot.ConnectionAttempts != 2 {
		t.Errorf("Expected 2 connection attempts, got %d", snapshot.ConnectionAttempts)
	}
	if snapshot.ConnectionSuccesses != 1 {
		t.Errorf("Expected 1 connection success, got %d", snapshot.ConnectionSuccesses)
	}
	if snapshot.ConnectionFailures != 1 {
		t.Errorf("Expected 1 connection failure, got %d", snapshot.ConnectionFailures)
	}
	if snapshot.ActiveConnections != 1 {
		t.Errorf("Expected 1 active connection, got %d", snapshot.ActiveConnections)
	}
	
	// Test disconnection
	metrics.RecordDisconnection()
	snapshot = metrics.GetSnapshot()
	if snapshot.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections after disconnect, got %d", snapshot.ActiveConnections)
	}
	
	// Test operation recording
	metrics.RecordOperation("test_op", time.Millisecond*100, nil)
	metrics.RecordOperation("test_op", time.Millisecond*200, nil)
	metrics.RecordOperation("test_op", time.Millisecond*50, errors.New("test error"))
	
	snapshot = metrics.GetSnapshot()
	if snapshot.OperationCount["test_op"] != 3 {
		t.Errorf("Expected 3 test_op operations, got %d", snapshot.OperationCount["test_op"])
	}
	if snapshot.OperationErrors["test_op"] != 1 {
		t.Errorf("Expected 1 test_op error, got %d", snapshot.OperationErrors["test_op"])
	}
	
	expectedDuration := time.Millisecond * 350
	if snapshot.OperationDuration["test_op"] != expectedDuration {
		t.Errorf("Expected %v total duration, got %v", expectedDuration, snapshot.OperationDuration["test_op"])
	}
	
	// Test health check recording
	metrics.RecordHealthCheck(true)
	metrics.RecordHealthCheck(false)
	
	snapshot = metrics.GetSnapshot()
	if snapshot.HealthCheckCount != 2 {
		t.Errorf("Expected 2 health checks, got %d", snapshot.HealthCheckCount)
	}
	if snapshot.HealthCheckFailures != 1 {
		t.Errorf("Expected 1 health check failure, got %d", snapshot.HealthCheckFailures)
	}
	if snapshot.LastHealthCheckStatus != false {
		t.Errorf("Expected last health check to be false")
	}
	
	// Test cache recording
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheMiss()
	metrics.RecordCacheError()
	
	snapshot = metrics.GetSnapshot()
	if snapshot.CacheHits != 2 {
		t.Errorf("Expected 2 cache hits, got %d", snapshot.CacheHits)
	}
	if snapshot.CacheMisses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", snapshot.CacheMisses)
	}
	if snapshot.CacheErrors != 1 {
		t.Errorf("Expected 1 cache error, got %d", snapshot.CacheErrors)
	}
}

func TestRepositoryMonitor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	monitor := NewRepositoryMonitor(logger)
	
	// Test operation recording with timing
	err := monitor.RecordOperation("test_operation", func() error {
		time.Sleep(time.Millisecond * 10)
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error from successful operation, got %v", err)
	}
	
	// Test operation recording with error
	testErr := errors.New("test error")
	err = monitor.RecordOperation("test_operation_error", func() error {
		return testErr
	})
	
	if err != testErr {
		t.Errorf("Expected error to be returned, got %v", err)
	}
	
	// Verify metrics were recorded
	metrics := monitor.GetMetrics()
	if metrics.OperationCount["test_operation"] != 1 {
		t.Errorf("Expected 1 successful operation, got %d", metrics.OperationCount["test_operation"])
	}
	if metrics.OperationCount["test_operation_error"] != 1 {
		t.Errorf("Expected 1 error operation, got %d", metrics.OperationCount["test_operation_error"])
	}
	if metrics.OperationErrors["test_operation_error"] != 1 {
		t.Errorf("Expected 1 operation error, got %d", metrics.OperationErrors["test_operation_error"])
	}
}

func TestRepositoryMonitorAlerting(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	monitor := NewRepositoryMonitor(logger)
	
	// Set low thresholds for testing
	monitor.maxErrorRate = 0.1 // 10%
	monitor.maxResponseTime = time.Millisecond * 5
	
	// Generate operations that exceed thresholds
	for i := 0; i < 10; i++ {
		var err error
		if i < 2 { // 20% error rate
			err = errors.New("test error")
		}
		
		monitor.RecordOperation("high_error_op", func() error {
			time.Sleep(time.Millisecond * 10) // Slow operation
			return err
		})
	}
	
	// Check alerts (this would normally log warnings)
	monitor.checkAlerts()
	
	// Verify metrics
	metrics := monitor.GetMetrics()
	if metrics.OperationCount["high_error_op"] != 10 {
		t.Errorf("Expected 10 operations, got %d", metrics.OperationCount["high_error_op"])
	}
	if metrics.OperationErrors["high_error_op"] != 2 {
		t.Errorf("Expected 2 errors, got %d", metrics.OperationErrors["high_error_op"])
	}
}

func TestRepositoryMonitorStartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	monitor := NewRepositoryMonitor(logger)
	
	// Set short interval for testing
	monitor.healthCheckInterval = time.Millisecond * 100
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*250)
	defer cancel()
	
	// Start monitoring in background
	done := make(chan bool)
	go func() {
		monitor.StartMonitoring(ctx)
		done <- true
	}()
	
	// Wait for context to expire
	<-ctx.Done()
	
	// Wait for monitoring to stop
	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Monitoring did not stop within timeout")
	}
}