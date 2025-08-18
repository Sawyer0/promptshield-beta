package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// QuotaManager defines the interface for quota management operations
type QuotaManager interface {
	// SetQuota sets quota limits for an entity
	SetQuota(ctx context.Context, quota *types.Quota) error
	
	// GetQuota retrieves quota configuration for an entity
	GetQuota(ctx context.Context, entityID string, entityType string) (*types.Quota, error)
	
	// UpdateQuota updates existing quota limits
	UpdateQuota(ctx context.Context, entityID string, updates *types.Quota) error
	
	// DeleteQuota removes quota limits for an entity
	DeleteQuota(ctx context.Context, entityID string, entityType string) error
	
	// ListQuotas lists all quotas
	ListQuotas(ctx context.Context) ([]*types.Quota, error)
	
	// ResetQuota resets usage counters for a quota
	ResetQuota(ctx context.Context, entityID string, entityType string) error
	
	// GetQuotaStatus returns current quota status and usage
	GetQuotaStatus(ctx context.Context, entityID string, entityType string) (*types.QuotaStatus, error)
}

// UsageTracker defines the interface for tracking resource usage
type UsageTracker interface {
	// IncrementUsage increments usage counters
	IncrementUsage(ctx context.Context, metric *types.UsageMetric) error
	
	// GetUsage retrieves usage data for an entity
	GetUsage(ctx context.Context, entityID string, entityType string, timeRange types.TimeRange) ([]*types.UsageMetric, error)
	
	// GetCurrentUsage returns current usage within a time window
	GetCurrentUsage(ctx context.Context, entityID string, entityType string, window time.Duration) (*types.UsageMetric, error)
	
	// ResetUsage resets usage counters
	ResetUsage(ctx context.Context, entityID string, entityType string) error
	
	// GetUsageHistory returns historical usage data
	GetUsageHistory(ctx context.Context, entityID string, timeRange types.TimeRange, granularity time.Duration) ([]*types.UsageMetric, error)
	
	// AggregateUsage aggregates usage data across multiple entities
	AggregateUsage(ctx context.Context, entityIDs []string, timeRange types.TimeRange) (*types.UsageMetric, error)
	
	// ExportUsage exports usage data in specified format
	ExportUsage(ctx context.Context, filter *types.UsageFilter, format string) ([]byte, error)
}

// RateLimiter defines the interface for rate limiting operations
type RateLimiter interface {
	// CheckLimit checks if a request is within rate limits
	CheckLimit(ctx context.Context, key string, limit *types.RateLimit) (*types.RateLimitResult, error)
	
	// AllowRequest checks and consumes from rate limit
	AllowRequest(ctx context.Context, key string, cost int) (*types.RateLimitResult, error)
	
	// GetRemainingQuota returns remaining quota for a key
	GetRemainingQuota(ctx context.Context, key string) (int, error)
	
	// ResetLimit resets rate limit counters for a key
	ResetLimit(ctx context.Context, key string) error
	
	// SetLimit configures rate limit for a key
	SetLimit(ctx context.Context, key string, limit *types.RateLimit) error
	
	// GetLimit retrieves rate limit configuration for a key
	GetLimit(ctx context.Context, key string) (*types.RateLimit, error)
	
	// DeleteLimit removes rate limit for a key
	DeleteLimit(ctx context.Context, key string) error
}

// APITokenManager defines the interface for API token management
type APITokenManager interface {
	// CreateToken creates a new API token
	CreateToken(ctx context.Context, token *types.APIToken) error
	
	// GetToken retrieves an API token by ID
	GetToken(ctx context.Context, tokenID string) (*types.APIToken, error)
	
	// ValidateToken validates an API token
	ValidateToken(ctx context.Context, tokenValue string) (*types.TokenValidation, error)
	
	// UpdateToken updates API token metadata
	UpdateToken(ctx context.Context, tokenID string, updates *types.APIToken) error
	
	// RevokeToken revokes an API token
	RevokeToken(ctx context.Context, tokenID string) error
	
	// ListTokens lists API tokens for an entity
	ListTokens(ctx context.Context, entityID string, entityType string) ([]*types.APIToken, error)
	
	// RotateToken rotates an API token
	RotateToken(ctx context.Context, tokenID string) (*types.APIToken, error)
	
	// GetTokenUsage returns usage statistics for a token
	GetTokenUsage(ctx context.Context, tokenID string, timeRange types.TimeRange) (*types.TokenUsage, error)
}

// ProviderKeyManager defines the interface for provider key management
type ProviderKeyManager interface {
	// StoreKey stores a provider API key
	StoreKey(ctx context.Context, key *types.ProviderKey) error
	
	// GetKey retrieves a provider API key
	GetKey(ctx context.Context, provider types.Provider, tenantID string) (*types.ProviderKey, error)
	
	// UpdateKey updates a provider API key
	UpdateKey(ctx context.Context, keyID string, key *types.ProviderKey) error
	
	// DeleteKey deletes a provider API key
	DeleteKey(ctx context.Context, keyID string) error
	
	// ListKeys lists provider API keys for a tenant
	ListKeys(ctx context.Context, tenantID string) ([]*types.ProviderKey, error)
	
	// RotateKey rotates a provider API key
	RotateKey(ctx context.Context, keyID string, newKey string) error
	
	// ValidateKey validates a provider API key
	ValidateKey(ctx context.Context, provider types.Provider, keyValue string) (*types.KeyValidation, error)
	
	// GetKeyUsage returns usage statistics for a provider key
	GetKeyUsage(ctx context.Context, keyID string, timeRange types.TimeRange) (*types.KeyUsage, error)
}

// UsageReporter defines the interface for usage reporting
type UsageReporter interface {
	// GenerateReport generates a usage report
	GenerateReport(ctx context.Context, config *types.UsageReportConfig) (*types.UsageReport, error)
	
	// GetUsageSummary returns usage summary for a time period
	GetUsageSummary(ctx context.Context, timeRange types.TimeRange) (*types.UsageSummary, error)
	
	// GetTopUsers returns top users by usage
	GetTopUsers(ctx context.Context, timeRange types.TimeRange, limit int) ([]*types.UserUsage, error)
	
	// GetUsageTrends returns usage trends over time
	GetUsageTrends(ctx context.Context, timeRange types.TimeRange, granularity time.Duration) ([]*types.UsageTrend, error)
	
	// GetCostAnalysis returns cost analysis based on usage
	GetCostAnalysis(ctx context.Context, timeRange types.TimeRange) (*types.CostAnalysis, error)
	
	// ExportReport exports usage report in specified format
	ExportReport(ctx context.Context, reportID string, format string) ([]byte, error)
	
	// ScheduleReport schedules periodic usage report generation
	ScheduleReport(ctx context.Context, config *types.UsageReportSchedule) error
	
	// GetReportHistory returns previously generated reports
	GetReportHistory(ctx context.Context, filter *types.ReportFilter) ([]*types.UsageReport, error)
}

// QuotaNotifier defines the interface for quota-related notifications
type QuotaNotifier interface {
	// NotifyQuotaExceeded sends notification when quota is exceeded
	NotifyQuotaExceeded(ctx context.Context, quota *types.Quota, usage *types.UsageMetric) error
	
	// NotifyQuotaWarning sends warning when quota threshold is reached
	NotifyQuotaWarning(ctx context.Context, quota *types.Quota, usage *types.UsageMetric, threshold float64) error
	
	// NotifyUsageSpike sends notification for unusual usage patterns
	NotifyUsageSpike(ctx context.Context, spike *types.UsageSpike) error
	
	// ConfigureAlerts configures quota-based alerts
	ConfigureAlerts(ctx context.Context, alerts []*types.QuotaAlert) error
	
	// SetThresholds sets notification thresholds for quotas
	SetThresholds(ctx context.Context, entityID string, thresholds map[string]float64) error
	
	// GetAlertHistory returns quota alert history
	GetAlertHistory(ctx context.Context, filter *types.AlertFilter) ([]*types.QuotaAlertLog, error)
}

// BillingIntegration defines the interface for billing system integration
type BillingIntegration interface {
	// RecordUsage records billable usage
	RecordUsage(ctx context.Context, usage *types.BillableUsage) error
	
	// GetBillingSummary returns billing summary for a period
	GetBillingSummary(ctx context.Context, entityID string, timeRange types.TimeRange) (*types.BillingSummary, error)
	
	// CalculateCost calculates cost for usage
	CalculateCost(ctx context.Context, usage *types.UsageMetric, pricing *types.PricingModel) (*types.CostCalculation, error)
	
	// GetInvoiceData returns data for invoice generation
	GetInvoiceData(ctx context.Context, entityID string, timeRange types.TimeRange) (*types.InvoiceData, error)
	
	// UpdatePricing updates pricing models
	UpdatePricing(ctx context.Context, pricing *types.PricingModel) error
	
	// GetPricing returns current pricing models
	GetPricing(ctx context.Context, entityType string) ([]*types.PricingModel, error)
	
	// ProcessPayment processes payment for usage
	ProcessPayment(ctx context.Context, payment *types.PaymentRequest) (*types.PaymentResult, error)
}

// CostOptimizer defines the interface for cost optimization
type CostOptimizer interface {
	// AnalyzeCosts analyzes cost patterns and identifies optimization opportunities
	AnalyzeCosts(ctx context.Context, timeRange types.TimeRange) (*types.CostAnalysis, error)
	
	// GetOptimizationSuggestions returns cost optimization suggestions
	GetOptimizationSuggestions(ctx context.Context, entityID string) ([]*types.OptimizationSuggestion, error)
	
	// PredictCosts predicts future costs based on usage patterns
	PredictCosts(ctx context.Context, entityID string, timeRange types.TimeRange) (*types.CostPrediction, error)
	
	// SetCostAlerts sets alerts for cost thresholds
	SetCostAlerts(ctx context.Context, alerts []*types.CostAlert) error
	
	// GetCostBreakdown returns detailed cost breakdown
	GetCostBreakdown(ctx context.Context, entityID string, timeRange types.TimeRange) (*types.CostBreakdown, error)
	
	// OptimizeQuotas optimizes quota allocations based on usage patterns
	OptimizeQuotas(ctx context.Context, entityID string) ([]*types.QuotaOptimization, error)
}