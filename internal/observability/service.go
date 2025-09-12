package observability

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"crypto/tls"
	"crypto/x509"
	"os"
	"strings"
	grpcCredentials "google.golang.org/grpc/credentials"
)

// ObservabilityService provides comprehensive monitoring, tracing, and alerting
// Note: Metrics are now handled by Prometheus client_golang in internal/observability/metrics/
type ObservabilityService struct {
	db             *sql.DB
	tracer         trace.Tracer
	alertChan      chan Alert
	metricStore    *MetricStore
	dashboardCache *DashboardCache
	mu             sync.RWMutex
}

// Alert represents a system alert
type Alert struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// MetricStore stores time-series metrics
type MetricStore struct {
	mu      sync.RWMutex
	metrics map[string][]MetricPoint
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

// DashboardCache caches dashboard data
type DashboardCache struct {
	mu         sync.RWMutex
	data       map[string]interface{}
	lastUpdate time.Time
	ttl        time.Duration
}

// NewObservabilityService creates a new observability service
// Note: Metrics are now handled by Prometheus client_golang in internal/observability/metrics/
func NewObservabilityService(ctx context.Context, db *sql.DB, otlpEndpoint string) (*ObservabilityService, error) {
	// Initialize OpenTelemetry for tracing only
	tp, _, err := initOpenTelemetry(ctx, otlpEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}

	// Create tracer only (metrics handled by Prometheus)
	tracer := tp.Tracer("promptshield")

	svc := &ObservabilityService{
		db:             db,
		tracer:         tracer,
		alertChan:      make(chan Alert, 1000),
		metricStore:    &MetricStore{metrics: make(map[string][]MetricPoint)},
		dashboardCache: &DashboardCache{data: make(map[string]interface{}), ttl: 30 * time.Second},
	}

	// Start background workers
	go svc.alertProcessor(ctx)
	go svc.metricsAggregator(ctx)

	return svc, nil
}

// RecordRequest records a request with its details
// Note: Metrics are now handled by Prometheus client_golang in internal/observability/metrics/
func (s *ObservabilityService) RecordRequest(ctx context.Context, tenantID, endpoint string, duration time.Duration, status int) {
	// Store in database for historical analysis
	go s.storeRequestMetric(tenantID, endpoint, duration, status)
}

// RecordViolation records a security violation
// Note: Metrics are now handled by Prometheus client_golang in internal/observability/metrics/
func (s *ObservabilityService) RecordViolation(ctx context.Context, tenantID, ruleID, severity string) {
	// Check if alert needed
	if severity == "CRITICAL" || severity == "HIGH" {
		s.triggerAlert(Alert{
			ID:       fmt.Sprintf("vio-%d", time.Now().Unix()),
			TenantID: tenantID,
			Type:     "security_violation",
			Severity: severity,
			Title:    fmt.Sprintf("Security violation detected: %s", ruleID),
			Metadata: map[string]interface{}{
				"rule_id":  ruleID,
				"severity": severity,
			},
			Timestamp: time.Now(),
		})
	}

	// Store in database
	go s.storeViolation(tenantID, ruleID, severity)
}

// RecordAPICall records an API call for usage tracking
// Note: Metrics are now handled by Prometheus client_golang in internal/observability/metrics/
func (s *ObservabilityService) RecordAPICall(ctx context.Context, tenantID string, dataBytes int64) {
	// Update usage metrics in database
	go s.updateUsageMetrics(tenantID, dataBytes)
}

// RecordError records an error
// Note: Metrics are now handled by Prometheus client_golang in internal/observability/metrics/
func (s *ObservabilityService) RecordError(ctx context.Context, tenantID, errorType string, err error) {
	// Error logging is handled by the calling code
}

// StartSpan starts a new tracing span
func (s *ObservabilityService) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return s.tracer.Start(ctx, name, opts...)
}

// GetDashboardData returns cached dashboard data
func (s *ObservabilityService) GetDashboardData(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	s.dashboardCache.mu.RLock()
	if time.Since(s.dashboardCache.lastUpdate) < s.dashboardCache.ttl {
		data := s.dashboardCache.data
		s.dashboardCache.mu.RUnlock()
		return data, nil
	}
	s.dashboardCache.mu.RUnlock()

	// Refresh cache
	data, err := s.loadDashboardData(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	s.dashboardCache.mu.Lock()
	s.dashboardCache.data = data
	s.dashboardCache.lastUpdate = time.Now()
	s.dashboardCache.mu.Unlock()

	return data, nil
}

// GetPerformanceMetrics returns current performance metrics
func (s *ObservabilityService) GetPerformanceMetrics() map[string]interface{} {
	// Performance metrics are no longer tracked in this service,
	// but returning a placeholder to avoid breaking existing calls.
	return map[string]interface{}{
		"p50_ms":     0,
		"p95_ms":     0,
		"p99_ms":     0,
		"throughput": 0,
		"error_rate": 0,
	}
}

// Private methods

func (s *ObservabilityService) alertProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert := <-s.alertChan:
			s.processAlert(alert)
		}
	}
}

func (s *ObservabilityService) processAlert(alert Alert) {
	// Store alert in database
	query := `
		INSERT INTO alerts (id, tenant_id, alert_type, severity, title, description, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	metadata, _ := json.Marshal(alert.Metadata)
	_, err := s.db.Exec(query, alert.ID, alert.TenantID, alert.Type, alert.Severity,
		alert.Title, alert.Description, metadata, alert.Timestamp)

	if err != nil {
		slog.Error("Failed to store alert", "error", err, "alert_id", alert.ID)
	}

	// Trigger webhooks
	go s.triggerWebhooks(alert)
}

func (s *ObservabilityService) triggerWebhooks(alert Alert) {
	// Query webhooks for tenant
	query := `
		SELECT url, secret FROM webhooks 
		WHERE tenant_id = $1 AND is_active = true 
		AND $2 = ANY(events)
	`

	rows, err := s.db.Query(query, alert.TenantID, "alert.triggered")
	if err != nil {
		return
	}
	defer rows.Close()

	// Send webhook for each configured endpoint
	for rows.Next() {
		var url, secret string
		if err := rows.Scan(&url, &secret); err != nil {
			continue
		}
		// TODO: Send webhook with HMAC signature
	}
}

func (s *ObservabilityService) metricsAggregator(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.aggregateMetrics()
		}
	}
}

func (s *ObservabilityService) aggregateMetrics() {
	// Aggregate metrics from various sources
	query := `
		INSERT INTO usage_metrics (tenant_id, metric_type, value, period_start, period_end)
		SELECT 
			tenant_id,
			'api_calls' as metric_type,
			COUNT(*) as value,
			date_trunc('hour', NOW() - INTERVAL '1 hour') as period_start,
			date_trunc('hour', NOW()) as period_end
		FROM audits
		WHERE created_at >= NOW() - INTERVAL '1 hour'
		GROUP BY tenant_id
	`

	_, _ = s.db.Exec(query)
}

func (s *ObservabilityService) storeRequestMetric(tenantID, endpoint string, duration time.Duration, status int) {
	s.metricStore.mu.Lock()
	defer s.metricStore.mu.Unlock()

	key := fmt.Sprintf("request_%s_%s", tenantID, endpoint)
	s.metricStore.metrics[key] = append(s.metricStore.metrics[key], MetricPoint{
		Timestamp: time.Now(),
		Value:     duration.Seconds() * 1000,
		Labels: map[string]string{
			"tenant_id": tenantID,
			"endpoint":  endpoint,
			"status":    fmt.Sprintf("%d", status),
		},
	})

	// Keep only last hour of data
	if len(s.metricStore.metrics[key]) > 3600 {
		s.metricStore.metrics[key] = s.metricStore.metrics[key][1:]
	}
}

func (s *ObservabilityService) storeViolation(tenantID, ruleID, severity string) {
	query := `
		INSERT INTO violations (tenant_id, rule_id, severity, action_taken, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, _ = s.db.Exec(query, tenantID, ruleID, severity, "blocked", time.Now())
}

func (s *ObservabilityService) updateUsageMetrics(tenantID string, dataBytes int64) {
	query := `
		INSERT INTO usage_metrics (tenant_id, metric_type, value, period_start, period_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, metric_type, period_start)
		DO UPDATE SET value = usage_metrics.value + EXCLUDED.value
	`

	now := time.Now()
	periodStart := now.Truncate(time.Hour)
	periodEnd := periodStart.Add(time.Hour)

	_, _ = s.db.Exec(query, tenantID, "data_processed_bytes", dataBytes, periodStart, periodEnd, now)
}

func (s *ObservabilityService) loadDashboardData(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// Get request statistics
	var totalRequests, blockedRequests int64
	err := s.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN action = 'block' THEN 1 END) as blocked
		FROM audits
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'
	`, tenantID).Scan(&totalRequests, &blockedRequests)

	if err != nil {
		return nil, err
	}

	data["total_requests_24h"] = totalRequests
	data["blocked_requests_24h"] = blockedRequests
	data["block_rate"] = float64(blockedRequests) / float64(totalRequests) * 100

	// Get violation trends
	rows, err := s.db.Query(`
		SELECT 
			date_trunc('hour', created_at) as hour,
			COUNT(*) as count,
			severity
		FROM violations
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'
		GROUP BY hour, severity
		ORDER BY hour
	`, tenantID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	violations := []map[string]interface{}{}
	for rows.Next() {
		var hour time.Time
		var count int
		var severity string
		if err := rows.Scan(&hour, &count, &severity); err != nil {
			continue
		}
		violations = append(violations, map[string]interface{}{
			"hour":     hour,
			"count":    count,
			"severity": severity,
		})
	}
	data["violation_trends"] = violations

	// Get performance metrics
	data["performance"] = s.GetPerformanceMetrics()

	// Get top rules triggered
	rows, err = s.db.Query(`
		SELECT rule_id, COUNT(*) as count
		FROM violations
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'
		GROUP BY rule_id
		ORDER BY count DESC
		LIMIT 10
	`, tenantID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topRules := []map[string]interface{}{}
	for rows.Next() {
		var ruleID string
		var count int
		if err := rows.Scan(&ruleID, &count); err != nil {
			continue
		}
		topRules = append(topRules, map[string]interface{}{
			"rule_id": ruleID,
			"count":   count,
		})
	}
	data["top_rules"] = topRules

	return data, nil
}

func (s *ObservabilityService) triggerAlert(alert Alert) {
	select {
	case s.alertChan <- alert:
	default:
		// Channel full, drop alert
	}
}

// Helper functions

func initOpenTelemetry(ctx context.Context, endpoint string) (*sdktrace.TracerProvider, interface{}, error) {
    // Create OTLP trace exporter
    var traceExporter sdktrace.SpanExporter
    var err error

    if endpoint != "" {
        // Optional mTLS controls via env (same keys as enforcer HTTP)
        insecure := strings.EqualFold(strings.TrimSpace(os.Getenv("PS_OTEL_INSECURE")), "true") || strings.TrimSpace(os.Getenv("PS_OTEL_INSECURE")) == "1"
        var opts []otlptracegrpc.Option
        opts = append(opts, otlptracegrpc.WithEndpoint(endpoint))
        if insecure {
            opts = append(opts, otlptracegrpc.WithInsecure())
        } else {
            tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
            if sni := strings.TrimSpace(os.Getenv("PS_OTEL_SERVER_NAME")); sni != "" {
                tlsCfg.ServerName = sni
            }
            if caFile := strings.TrimSpace(os.Getenv("PS_OTEL_CA_FILE")); caFile != "" {
                if pem, err := os.ReadFile(caFile); err == nil {
                    pool := x509.NewCertPool()
                    if pool.AppendCertsFromPEM(pem) { tlsCfg.RootCAs = pool }
                }
            }
            certFile := strings.TrimSpace(os.Getenv("PS_OTEL_CLIENT_CERT_FILE"))
            keyFile := strings.TrimSpace(os.Getenv("PS_OTEL_CLIENT_KEY_FILE"))
            if certFile != "" && keyFile != "" {
                if crt, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
                    tlsCfg.Certificates = []tls.Certificate{crt}
                }
            }
            creds := grpcCredentials.NewTLS(tlsCfg)
            opts = append(opts, otlptracegrpc.WithTLSCredentials(creds))
        }
        traceExporter, err = otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
        if err != nil {
            return nil, nil, err
        }
    } else {
        // Use no-op exporter if no endpoint provided
		traceExporter = &noopExporter{}
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Return nil for meter provider since metrics are now handled by Prometheus
	return tp, nil, nil
}

// noopExporter is a no-op span exporter
type noopExporter struct{}

func (n *noopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (n *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}

func calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	index := int(float64(len(values)) * percentile / 100)
	if index >= len(values) {
		index = len(values) - 1
	}

	return values[index]
}
