package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application"
	sharedcontext "github.com/promptshield/promptshield/internal/shared/context"
	"github.com/promptshield/promptshield/internal/shared/httputil"
)

// QuotaMiddleware enforces usage quotas for API requests
type QuotaMiddleware struct {
	billingService application.BillingService
}

// NewQuotaMiddleware creates a new quota middleware
func NewQuotaMiddleware(billingService application.BillingService) *QuotaMiddleware {
	return &QuotaMiddleware{
		billingService: billingService,
	}
}

// EnforceQuotaMiddleware returns a middleware function that enforces quotas
func (m *QuotaMiddleware) EnforceQuotaMiddleware(resourceType application.QuotaResource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get tenant ID from context (set by auth middleware)
			tenantID, ok := sharedcontext.GetTenantID(r.Context())
			if !ok {
				httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
				return
			}

			// Check quota
			quotaStatus, err := m.billingService.CheckQuota(r.Context(), tenantID, resourceType)
			if err != nil {
				// Log error but don't block request for quota check failures
				// This ensures service availability even if billing service is down
				httputil.WriteError(w, http.StatusInternalServerError, "Failed to check quota", err)
				return
			}

			// Check if quota is exceeded
			if !quotaStatus.IsUnlimited && quotaStatus.Remaining <= 0 {
				// Check if overage is allowed
				if !quotaStatus.OverageAllowed {
					httputil.WriteJSON(w, http.StatusTooManyRequests, map[string]interface{}{
						"error":         "Quota exceeded",
						"resource_type": resourceType,
						"used":          quotaStatus.Used,
						"limit":         quotaStatus.Limit,
						"reset_date":    quotaStatus.ResetDate,
					})
					return
				}
			}

			// Add quota information to response headers
			w.Header().Set("X-Quota-Used", strconv.Itoa(quotaStatus.Used))
			w.Header().Set("X-Quota-Limit", strconv.Itoa(quotaStatus.Limit))
			if quotaStatus.Remaining >= 0 {
				w.Header().Set("X-Quota-Remaining", strconv.Itoa(quotaStatus.Remaining))
			} else {
				w.Header().Set("X-Quota-Remaining", "unlimited")
			}
			if quotaStatus.ResetDate != nil {
				w.Header().Set("X-Quota-Reset", quotaStatus.ResetDate.Format("2006-01-02T15:04:05Z"))
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RecordUsageMiddleware returns a middleware function that records usage
func (m *QuotaMiddleware) RecordUsageMiddleware(resourceType application.QuotaResource, increment int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get tenant ID from context
			tenantID, ok := sharedcontext.GetTenantID(r.Context())
			if !ok {
				// Continue without recording usage if no tenant ID
				next.ServeHTTP(w, r)
				return
			}

			// Create a response writer that captures the status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler
			next.ServeHTTP(rw, r)

			// Only record usage for successful requests (2xx status codes)
			if rw.statusCode >= 200 && rw.statusCode < 300 {
				// Record usage asynchronously to avoid blocking the response
				go func() {
					usage := application.UsageRecord{
						Timestamp: time.Now(),
					}

					switch resourceType {
					case application.QuotaResourceAPICalls:
						usage.APICalls = increment
					case application.QuotaResourceLLMCalls:
						usage.LLMCalls = increment
					case application.QuotaResourceRulepacks:
						// Rulepack usage is typically recorded when created/updated
						return
					case application.QuotaResourceUsers:
						// User usage is typically recorded when users are added
						return
					}

					if err := m.billingService.RecordUsage(context.Background(), tenantID, usage); err != nil {
						// Log error but don't fail the request
						// TODO: Add proper logging
					}
				}()
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Usage tracking for specific endpoints
type UsageTracker struct {
	billingService application.BillingService
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker(billingService application.BillingService) *UsageTracker {
	return &UsageTracker{
		billingService: billingService,
	}
}

// TrackAPICall records an API call
func (t *UsageTracker) TrackAPICall(ctx context.Context, tenantID uuid.UUID) error {
	usage := application.UsageRecord{
		APICalls:  1,
		Timestamp: time.Now(),
	}
	return t.billingService.RecordUsage(ctx, tenantID, usage)
}

// TrackLLMCall records an LLM call
func (t *UsageTracker) TrackLLMCall(ctx context.Context, tenantID uuid.UUID) error {
	usage := application.UsageRecord{
		LLMCalls:  1,
		Timestamp: time.Now(),
	}
	return t.billingService.RecordUsage(ctx, tenantID, usage)
}

// TrackViolation records a violation
func (t *UsageTracker) TrackViolation(ctx context.Context, tenantID uuid.UUID) error {
	usage := application.UsageRecord{
		Violations: 1,
		Timestamp:  time.Now(),
	}
	return t.billingService.RecordUsage(ctx, tenantID, usage)
}
