package repository

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// RepositoryMetrics tracks metrics for repository operations
type RepositoryMetrics struct {
	mu sync.RWMutex
	
	// Connection metrics
	ConnectionAttempts    int64
	ConnectionSuccesses   int64
	ConnectionFailures    int64
	ActiveConnections     int64
	
	// Operation metrics
	OperationCount        map[string]int64
	OperationDuration     map[string]time.Duration
	OperationErrors       map[string]int64
	
	// Health check metrics
	HealthCheckCount      int64
	HealthCheckFailures   int64
	LastHealthCheck       time.Time
	LastHealthCheckStatus bool
	
	// Cache metrics (when Redis is available)
	CacheHits             int64
	CacheMisses           int64
	CacheErrors           int64
}

// NewRepositoryMetrics creates a new metrics instance
func NewRepositoryMetrics() *RepositoryMetrics {
	return &RepositoryMetrics{
		OperationCount:    make(map[string]int64),
		OperationDuration: make(map[string]time.Duration),
		OperationErrors:   make(map[string]int64),
	}
}

// RecordConnection records a connection attempt and its result
func (m *RepositoryMetrics) RecordConnection(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ConnectionAttempts++
	if success {
		m.ConnectionSuccesses++
		m.ActiveConnections++
	} else {
		m.ConnectionFailures++
	}
}

// RecordDisconnection records a connection being closed
func (m *RepositoryMetrics) RecordDisconnection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.ActiveConnections > 0 {
		m.ActiveConnections--
	}
}

// RecordOperation records an operation and its duration
func (m *RepositoryMetrics) RecordOperation(operation string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.OperationCount[operation]++
	m.OperationDuration[operation] += duration
	
	if err != nil {
		m.OperationErrors[operation]++
	}
}

// RecordHealthCheck records a health check result
func (m *RepositoryMetrics) RecordHealthCheck(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.HealthCheckCount++
	m.LastHealthCheck = time.Now()
	m.LastHealthCheckStatus = success
	
	if !success {
		m.HealthCheckFailures++
	}
}

// RecordCacheOperation records cache hit/miss/error
func (m *RepositoryMetrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHits++
}

func (m *RepositoryMetrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheMisses++
}

func (m *RepositoryMetrics) RecordCacheError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheErrors++
}

// GetSnapshot returns a snapshot of current metrics
func (m *RepositoryMetrics) GetSnapshot() RepositoryMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create copies of maps to avoid race conditions
	opCount := make(map[string]int64)
	opDuration := make(map[string]time.Duration)
	opErrors := make(map[string]int64)
	
	for k, v := range m.OperationCount {
		opCount[k] = v
	}
	for k, v := range m.OperationDuration {
		opDuration[k] = v
	}
	for k, v := range m.OperationErrors {
		opErrors[k] = v
	}
	
	return RepositoryMetricsSnapshot{
		ConnectionAttempts:    m.ConnectionAttempts,
		ConnectionSuccesses:   m.ConnectionSuccesses,
		ConnectionFailures:    m.ConnectionFailures,
		ActiveConnections:     m.ActiveConnections,
		OperationCount:        opCount,
		OperationDuration:     opDuration,
		OperationErrors:       opErrors,
		HealthCheckCount:      m.HealthCheckCount,
		HealthCheckFailures:   m.HealthCheckFailures,
		LastHealthCheck:       m.LastHealthCheck,
		LastHealthCheckStatus: m.LastHealthCheckStatus,
		CacheHits:             m.CacheHits,
		CacheMisses:           m.CacheMisses,
		CacheErrors:           m.CacheErrors,
	}
}

// RepositoryMetricsSnapshot represents a point-in-time snapshot of metrics
type RepositoryMetricsSnapshot struct {
	ConnectionAttempts    int64                        `json:"connection_attempts"`
	ConnectionSuccesses   int64                        `json:"connection_successes"`
	ConnectionFailures    int64                        `json:"connection_failures"`
	ActiveConnections     int64                        `json:"active_connections"`
	OperationCount        map[string]int64             `json:"operation_count"`
	OperationDuration     map[string]time.Duration     `json:"operation_duration"`
	OperationErrors       map[string]int64             `json:"operation_errors"`
	HealthCheckCount      int64                        `json:"health_check_count"`
	HealthCheckFailures   int64                        `json:"health_check_failures"`
	LastHealthCheck       time.Time                    `json:"last_health_check"`
	LastHealthCheckStatus bool                         `json:"last_health_check_status"`
	CacheHits             int64                        `json:"cache_hits"`
	CacheMisses           int64                        `json:"cache_misses"`
	CacheErrors           int64                        `json:"cache_errors"`
}

// RepositoryMonitor provides monitoring and alerting for repository operations
type RepositoryMonitor struct {
	logger  *slog.Logger
	metrics *RepositoryMetrics
	
	// Alerting thresholds
	maxErrorRate         float64
	maxResponseTime      time.Duration
	healthCheckInterval  time.Duration
	
	// State
	alerting bool
	mu       sync.RWMutex
}

// NewRepositoryMonitor creates a new repository monitor
func NewRepositoryMonitor(logger *slog.Logger) *RepositoryMonitor {
	return &RepositoryMonitor{
		logger:              logger.With("component", "repository-monitor"),
		metrics:             NewRepositoryMetrics(),
		maxErrorRate:        0.05, // 5% error rate threshold
		maxResponseTime:     time.Second * 5,
		healthCheckInterval: time.Minute * 1,
		alerting:            false,
	}
}

// StartMonitoring begins background monitoring
func (m *RepositoryMonitor) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAlerts()
		}
	}
}

// checkAlerts evaluates current metrics against thresholds
func (m *RepositoryMonitor) checkAlerts() {
	snapshot := m.metrics.GetSnapshot()
	
	// Check error rates
	for operation, count := range snapshot.OperationCount {
		if count > 0 {
			errorCount := snapshot.OperationErrors[operation]
			errorRate := float64(errorCount) / float64(count)
			
			if errorRate > m.maxErrorRate {
				m.logger.Warn("High error rate detected",
					"operation", operation,
					"error_rate", errorRate,
					"threshold", m.maxErrorRate,
					"total_operations", count,
					"errors", errorCount)
			}
		}
	}
	
	// Check response times
	for operation, totalDuration := range snapshot.OperationDuration {
		if count := snapshot.OperationCount[operation]; count > 0 {
			avgDuration := totalDuration / time.Duration(count)
			if avgDuration > m.maxResponseTime {
				m.logger.Warn("Slow operation detected",
					"operation", operation,
					"avg_duration", avgDuration,
					"threshold", m.maxResponseTime,
					"total_operations", count)
			}
		}
	}
	
	// Check health check status
	if !snapshot.LastHealthCheckStatus && !snapshot.LastHealthCheck.IsZero() {
		m.logger.Error("Repository health check failing",
			"last_check", snapshot.LastHealthCheck,
			"failure_count", snapshot.HealthCheckFailures,
			"total_checks", snapshot.HealthCheckCount)
	}
}

// RecordOperation is a convenience method to record operations with timing
func (m *RepositoryMonitor) RecordOperation(operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	m.metrics.RecordOperation(operation, duration, err)
	
	if err != nil {
		m.logger.Error("Repository operation failed",
			"operation", operation,
			"duration", duration,
			"error", err)
	} else {
		m.logger.Debug("Repository operation completed",
			"operation", operation,
			"duration", duration)
	}
	
	return err
}

// GetMetrics returns the current metrics snapshot
func (m *RepositoryMonitor) GetMetrics() RepositoryMetricsSnapshot {
	return m.metrics.GetSnapshot()
}

// LogMetricsSummary logs a summary of current metrics
func (m *RepositoryMonitor) LogMetricsSummary() {
	snapshot := m.metrics.GetSnapshot()
	
	m.logger.Info("Repository metrics summary",
		"active_connections", snapshot.ActiveConnections,
		"connection_success_rate", m.calculateSuccessRate(snapshot.ConnectionSuccesses, snapshot.ConnectionAttempts),
		"health_check_success_rate", m.calculateSuccessRate(snapshot.HealthCheckCount-snapshot.HealthCheckFailures, snapshot.HealthCheckCount),
		"cache_hit_rate", m.calculateCacheHitRate(snapshot.CacheHits, snapshot.CacheMisses),
		"total_operations", m.sumOperations(snapshot.OperationCount))
}

func (m *RepositoryMonitor) calculateSuccessRate(successes, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(successes) / float64(total)
}

func (m *RepositoryMonitor) calculateCacheHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

func (m *RepositoryMonitor) sumOperations(operations map[string]int64) int64 {
	var total int64
	for _, count := range operations {
		total += count
	}
	return total
}