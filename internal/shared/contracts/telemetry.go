package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// TelemetryCollector defines the interface for telemetry data collection
type TelemetryCollector interface {
	// Initialize initializes the telemetry collector
	Initialize(ctx context.Context, config *types.TelemetryConfig) error
	
	// CollectMetrics collects system and application metrics
	CollectMetrics(ctx context.Context) (*types.PerformanceMetrics, error)
	
	// RecordEvent records a telemetry event
	RecordEvent(ctx context.Context, event *types.TelemetryEvent) error
	
	// RecordLatency records latency measurements
	RecordLatency(ctx context.Context, operation string, duration time.Duration, tags map[string]string) error
	
	// RecordThroughput records throughput measurements
	RecordThroughput(ctx context.Context, operation string, count int64, tags map[string]string) error
	
	// RecordErrorRate records error rate measurements
	RecordErrorRate(ctx context.Context, operation string, errors int64, total int64, tags map[string]string) error
	
	// GetMetrics returns collected metrics for a time range
	GetMetrics(ctx context.Context, timeRange types.TimeRange) (*types.MetricsSnapshot, error)
	
	// GetConfig returns the current telemetry configuration
	GetConfig() *types.TelemetryConfig
	
	// Flush flushes all pending telemetry data
	Flush(ctx context.Context) error
	
	// Close closes the telemetry collector
	Close() error
}

// PerformanceMonitor defines the interface for performance monitoring
type PerformanceMonitor interface {
	// StartMonitoring starts performance monitoring
	StartMonitoring(ctx context.Context) error
	
	// StopMonitoring stops performance monitoring
	StopMonitoring(ctx context.Context) error
	
	// GetPerformanceMetrics returns current performance metrics
	GetPerformanceMetrics(ctx context.Context) (*types.PerformanceMetrics, error)
	
	// GetSystemMetrics returns system-level metrics
	GetSystemMetrics(ctx context.Context) (*types.SystemMetrics, error)
	
	// GetApplicationMetrics returns application-specific metrics
	GetApplicationMetrics(ctx context.Context) (*types.ApplicationMetrics, error)
	
	// RegisterMetric registers a custom metric
	RegisterMetric(name string, metric types.CustomMetric) error
	
	// UnregisterMetric unregisters a custom metric
	UnregisterMetric(name string) error
	
	// SetThreshold sets a performance threshold
	SetThreshold(metric string, threshold types.Threshold) error
	
	// GetThresholds returns all configured thresholds
	GetThresholds() map[string]types.Threshold
	
	// CheckThresholds checks if any thresholds are exceeded
	CheckThresholds(ctx context.Context) ([]*types.ThresholdViolation, error)
}

// ProfilingService defines the interface for application profiling
type ProfilingService interface {
	// StartProfiling starts profiling for a specific operation
	StartProfiling(ctx context.Context, operationName string) (types.ProfileToken, error)
	
	// StopProfiling stops profiling and returns profile data
	StopProfiling(ctx context.Context, token types.ProfileToken) (*types.ProfileData, error)
	
	// GetProfile returns profile data for an operation
	GetProfile(ctx context.Context, operationName string, timeRange types.TimeRange) (*types.ProfileData, error)
	
	// GetCPUProfile returns CPU profiling data
	GetCPUProfile(ctx context.Context, duration time.Duration) ([]byte, error)
	
	// GetMemoryProfile returns memory profiling data
	GetMemoryProfile(ctx context.Context) ([]byte, error)
	
	// GetGoroutineProfile returns goroutine profiling data
	GetGoroutineProfile(ctx context.Context) ([]byte, error)
	
	// GetBlockProfile returns blocking profiling data
	GetBlockProfile(ctx context.Context) ([]byte, error)
	
	// GetMutexProfile returns mutex profiling data
	GetMutexProfile(ctx context.Context) ([]byte, error)
	
	// AnalyzeProfile analyzes profile data and returns insights
	AnalyzeProfile(ctx context.Context, profileData []byte, profileType string) (*types.ProfileAnalysis, error)
}

// AlertService defines the interface for telemetry-based alerting
type AlertService interface {
	// CreateAlert creates a new alert rule
	CreateAlert(ctx context.Context, alert *types.Alert) error
	
	// UpdateAlert updates an existing alert rule
	UpdateAlert(ctx context.Context, alertID string, alert *types.Alert) error
	
	// DeleteAlert deletes an alert rule
	DeleteAlert(ctx context.Context, alertID string) error
	
	// GetAlert retrieves an alert rule by ID
	GetAlert(ctx context.Context, alertID string) (*types.Alert, error)
	
	// ListAlerts lists all alert rules
	ListAlerts(ctx context.Context) ([]*types.Alert, error)
	
	// EvaluateAlerts evaluates all alert rules against current metrics
	EvaluateAlerts(ctx context.Context) ([]*types.AlertTrigger, error)
	
	// TriggerAlert manually triggers an alert
	TriggerAlert(ctx context.Context, alertID string, message string) error
	
	// ResolveAlert resolves an active alert
	ResolveAlert(ctx context.Context, alertID string) error
	
	// GetActiveAlerts returns all currently active alerts
	GetActiveAlerts(ctx context.Context) ([]*types.Alert, error)
	
	// GetAlertHistory returns alert history
	GetAlertHistory(ctx context.Context, filter types.AlertFilter) ([]*types.AlertHistory, error)
}

// TracingService defines the interface for distributed tracing
type TracingService interface {
	// StartTrace starts a new trace
	StartTrace(ctx context.Context, operationName string) (context.Context, types.TraceID, error)
	
	// StartSpan starts a new span within a trace
	StartSpan(ctx context.Context, operationName string, parentSpanID types.SpanID) (context.Context, types.SpanID, error)
	
	// FinishSpan finishes a span
	FinishSpan(ctx context.Context, spanID types.SpanID, tags map[string]interface{}) error
	
	// AddEvent adds an event to a span
	AddEvent(ctx context.Context, spanID types.SpanID, event *types.TraceEvent) error
	
	// SetTag sets a tag on a span
	SetTag(ctx context.Context, spanID types.SpanID, key string, value interface{}) error
	
	// GetTrace retrieves a complete trace by ID
	GetTrace(ctx context.Context, traceID types.TraceID) (*types.Trace, error)
	
	// GetSpan retrieves a specific span
	GetSpan(ctx context.Context, spanID types.SpanID) (*types.Span, error)
	
	// SearchTraces searches traces by criteria
	SearchTraces(ctx context.Context, query *types.TraceQuery) ([]*types.Trace, error)
	
	// GetTraceStatistics returns statistics about traces
	GetTraceStatistics(ctx context.Context, timeRange types.TimeRange) (*types.TraceStatistics, error)
}

// DataExporter defines the interface for exporting telemetry data
type DataExporter interface {
	// Export exports telemetry data to external systems
	Export(ctx context.Context, data interface{}, format types.ExportFormat) error
	
	// ExportMetrics exports metrics data
	ExportMetrics(ctx context.Context, metrics *types.MetricsSnapshot, destination string) error
	
	// ExportTraces exports trace data
	ExportTraces(ctx context.Context, traces []*types.Trace, destination string) error
	
	// ExportLogs exports log data
	ExportLogs(ctx context.Context, logs []*types.LogEntry, destination string) error
	
	// ScheduleExport schedules periodic data export
	ScheduleExport(ctx context.Context, config *types.ExportSchedule) error
	
	// CancelScheduledExport cancels a scheduled export
	CancelScheduledExport(ctx context.Context, scheduleID string) error
	
	// GetExportStatus returns the status of an export operation
	GetExportStatus(ctx context.Context, exportID string) (*types.ExportStatus, error)
	
	// ListExportDestinations returns available export destinations
	ListExportDestinations(ctx context.Context) ([]*types.ExportDestination, error)
}

// DashboardService defines the interface for telemetry dashboards
type DashboardService interface {
	// CreateDashboard creates a new dashboard
	CreateDashboard(ctx context.Context, dashboard *types.Dashboard) error
	
	// UpdateDashboard updates an existing dashboard
	UpdateDashboard(ctx context.Context, dashboardID string, dashboard *types.Dashboard) error
	
	// DeleteDashboard deletes a dashboard
	DeleteDashboard(ctx context.Context, dashboardID string) error
	
	// GetDashboard retrieves a dashboard by ID
	GetDashboard(ctx context.Context, dashboardID string) (*types.Dashboard, error)
	
	// ListDashboards lists all dashboards
	ListDashboards(ctx context.Context) ([]*types.Dashboard, error)
	
	// GetDashboardData returns data for dashboard widgets
	GetDashboardData(ctx context.Context, dashboardID string, timeRange types.TimeRange) (*types.DashboardData, error)
	
	// RefreshDashboard refreshes dashboard data
	RefreshDashboard(ctx context.Context, dashboardID string) error
	
	// ShareDashboard creates a shareable link for a dashboard
	ShareDashboard(ctx context.Context, dashboardID string, permissions types.SharingPermissions) (string, error)
	
	// GetSharedDashboard retrieves a dashboard via share link
	GetSharedDashboard(ctx context.Context, shareToken string) (*types.Dashboard, error)
}

// ReportGenerator defines the interface for generating telemetry reports
type ReportGenerator interface {
	// GenerateReport generates a telemetry report
	GenerateReport(ctx context.Context, config *types.ReportConfig) (*types.Report, error)
	
	// GeneratePerformanceReport generates a performance report
	GeneratePerformanceReport(ctx context.Context, timeRange types.TimeRange) (*types.PerformanceReport, error)
	
	// GenerateUsageReport generates a usage report
	GenerateUsageReport(ctx context.Context, timeRange types.TimeRange) (*types.UsageReport, error)
	
	// GenerateErrorReport generates an error analysis report
	GenerateErrorReport(ctx context.Context, timeRange types.TimeRange) (*types.ErrorReport, error)
	
	// ScheduleReport schedules periodic report generation
	ScheduleReport(ctx context.Context, config *types.ReportSchedule) error
	
	// CancelScheduledReport cancels a scheduled report
	CancelScheduledReport(ctx context.Context, scheduleID string) error
	
	// GetReportHistory returns previously generated reports
	GetReportHistory(ctx context.Context, filter *types.ReportFilter) ([]*types.Report, error)
	
	// ExportReport exports a report to various formats
	ExportReport(ctx context.Context, reportID string, format types.ExportFormat) ([]byte, error)
}

// AnomalyDetector defines the interface for anomaly detection in telemetry data
type AnomalyDetector interface {
	// DetectAnomalies detects anomalies in metrics data
	DetectAnomalies(ctx context.Context, metrics *types.MetricsSnapshot) ([]*types.Anomaly, error)
	
	// TrainModel trains the anomaly detection model
	TrainModel(ctx context.Context, trainingData []*types.MetricsSnapshot) error
	
	// UpdateModel updates the anomaly detection model
	UpdateModel(ctx context.Context, newData *types.MetricsSnapshot) error
	
	// GetModelInfo returns information about the current model
	GetModelInfo(ctx context.Context) (*types.ModelInfo, error)
	
	// SetThreshold sets anomaly detection thresholds
	SetThreshold(metric string, threshold float64) error
	
	// GetThresholds returns current anomaly detection thresholds
	GetThresholds() map[string]float64
	
	// GetAnomalyHistory returns historical anomalies
	GetAnomalyHistory(ctx context.Context, filter *types.AnomalyFilter) ([]*types.Anomaly, error)
	
	// ValidateModel validates the anomaly detection model
	ValidateModel(ctx context.Context, testData []*types.MetricsSnapshot) (*types.ModelValidation, error)
}