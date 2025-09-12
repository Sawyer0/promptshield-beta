package usage

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application"
	"github.com/promptshield/promptshield/pkg/types"
)

// UsageTracker tracks usage for billing purposes
type UsageTracker struct {
	billingService application.BillingService
	logger         *slog.Logger
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker(billingService application.BillingService, logger *slog.Logger) *UsageTracker {
	return &UsageTracker{
		billingService: billingService,
		logger:         logger,
	}
}

// TrackScanResult tracks usage from a scan result
func (t *UsageTracker) TrackScanResult(ctx context.Context, tenantID uuid.UUID, result types.ScanResult) error {
	if t.billingService == nil {
		// No billing service configured, skip tracking
		return nil
	}

	// Count violations
	violationCount := len(result.Violations)

	// Count LLM calls (semantic analysis attempts)
	llmCallCount := int(result.Metrics.SemanticAttempts)

	// Count API calls (always 1 per scan request)
	apiCallCount := 1

	// Record usage
	usage := application.UsageRecord{
		APICalls:   apiCallCount,
		LLMCalls:   llmCallCount,
		Violations: violationCount,
		Timestamp:  time.Now(),
	}

	if err := t.billingService.RecordUsage(ctx, tenantID, usage); err != nil {
		t.logger.Error("Failed to record usage", "error", err, "tenant_id", tenantID)
		return err
	}

	t.logger.Debug("Recorded usage",
		"tenant_id", tenantID,
		"api_calls", apiCallCount,
		"llm_calls", llmCallCount,
		"violations", violationCount,
	)

	return nil
}

// TrackAPICall tracks a single API call
func (t *UsageTracker) TrackAPICall(ctx context.Context, tenantID uuid.UUID) error {
	if t.billingService == nil {
		return nil
	}

	usage := application.UsageRecord{
		APICalls:  1,
		Timestamp: time.Now(),
	}

	return t.billingService.RecordUsage(ctx, tenantID, usage)
}

// TrackLLMCall tracks a single LLM call
func (t *UsageTracker) TrackLLMCall(ctx context.Context, tenantID uuid.UUID) error {
	if t.billingService == nil {
		return nil
	}

	usage := application.UsageRecord{
		LLMCalls:  1,
		Timestamp: time.Now(),
	}

	return t.billingService.RecordUsage(ctx, tenantID, usage)
}

// TrackViolation tracks a single violation
func (t *UsageTracker) TrackViolation(ctx context.Context, tenantID uuid.UUID) error {
	if t.billingService == nil {
		return nil
	}

	usage := application.UsageRecord{
		Violations: 1,
		Timestamp:  time.Now(),
	}

	return t.billingService.RecordUsage(ctx, tenantID, usage)
}

// CheckQuota checks if a tenant has remaining quota for a specific resource
func (t *UsageTracker) CheckQuota(ctx context.Context, tenantID uuid.UUID, resourceType application.QuotaResource) (*application.QuotaStatus, error) {
	if t.billingService == nil {
		// No billing service, allow unlimited usage
		return &application.QuotaStatus{
			ResourceType:   resourceType,
			Used:           0,
			Limit:          -1,
			Remaining:      -1,
			IsUnlimited:    true,
			OverageAllowed: true,
		}, nil
	}

	return t.billingService.CheckQuota(ctx, tenantID, resourceType)
}

// EnforceQuota enforces quota limits for a specific resource
func (t *UsageTracker) EnforceQuota(ctx context.Context, tenantID uuid.UUID, resourceType application.QuotaResource) error {
	if t.billingService == nil {
		// No billing service, allow unlimited usage
		return nil
	}

	return t.billingService.EnforceQuota(ctx, tenantID, resourceType)
}
