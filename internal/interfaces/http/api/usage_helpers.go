package api

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/usage"
	"golang.org/x/time/rate"
)

// Helper functions for usage tracking

// RecordAPIUsage records API usage metrics
func RecordAPIUsage(ctx context.Context, store usage.UsageStore, tenantID, route, decision string, bytes int64, duration time.Duration) error {
	if store == nil {
		return nil // Usage tracking disabled
	}
	
	return store.Record(ctx, tenantID, route, decision, bytes, time.Now())
}

// CheckQuota checks if a tenant is within their quota limits
func CheckQuota(store usage.QuotaStore, tenantID string) bool {
	if store == nil {
		return true // No quota enforcement
	}
	
	return store.Allow(tenantID)
}

// SetTenantQuota configures quota limits for a tenant
func SetTenantQuota(store usage.QuotaStore, tenantID string, requestsPerMinute float64, burst int) {
	if store == nil {
		return
	}
	
	store.Set(tenantID, rate.Limit(requestsPerMinute), burst)
}

