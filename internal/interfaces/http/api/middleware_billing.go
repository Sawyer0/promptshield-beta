package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application"
	sharedcontext "github.com/promptshield/promptshield/internal/shared/context"
	"github.com/promptshield/promptshield/internal/shared/httputil"
	"github.com/promptshield/promptshield/internal/usage"
)

// BillingMiddleware provides billing-related middleware functions
type BillingMiddleware struct {
	usageTracker *usage.UsageTracker
}

// NewBillingMiddleware creates a new billing middleware
func NewBillingMiddleware(usageTracker *usage.UsageTracker) *BillingMiddleware {
	return &BillingMiddleware{
		usageTracker: usageTracker,
	}
}

// EnforceAPICallQuotaMiddleware enforces API call quotas
func (m *BillingMiddleware) EnforceAPICallQuotaMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.usageTracker == nil {
				// No usage tracker configured, allow request
				next.ServeHTTP(w, r)
				return
			}

			// Get tenant ID from context
			tenantID, ok := sharedcontext.GetTenantID(r.Context())
			if !ok {
				// Try to get from header as fallback
				tenantIDStr := r.Header.Get("X-PS-Tenant-ID")
				if tenantIDStr == "" {
					httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
					return
				}
				var err error
				tenantID, err = uuid.Parse(tenantIDStr)
				if err != nil {
					httputil.WriteError(w, http.StatusBadRequest, "Invalid tenant ID", err)
					return
				}
			}

			// Check API call quota
			quotaStatus, err := m.usageTracker.CheckQuota(r.Context(), tenantID, application.QuotaResourceAPICalls)
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
						"error":         "API call quota exceeded",
						"resource_type": "api_calls",
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

// EnforceLLMCallQuotaMiddleware enforces LLM call quotas
func (m *BillingMiddleware) EnforceLLMCallQuotaMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.usageTracker == nil {
				// No usage tracker configured, allow request
				next.ServeHTTP(w, r)
				return
			}

			// Get tenant ID from context
			tenantID, ok := sharedcontext.GetTenantID(r.Context())
			if !ok {
				// Try to get from header as fallback
				tenantIDStr := r.Header.Get("X-PS-Tenant-ID")
				if tenantIDStr == "" {
					httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
					return
				}
				var err error
				tenantID, err = uuid.Parse(tenantIDStr)
				if err != nil {
					httputil.WriteError(w, http.StatusBadRequest, "Invalid tenant ID", err)
					return
				}
			}

			// Check LLM call quota
			quotaStatus, err := m.usageTracker.CheckQuota(r.Context(), tenantID, application.QuotaResourceLLMCalls)
			if err != nil {
				// Log error but don't block request for quota check failures
				httputil.WriteError(w, http.StatusInternalServerError, "Failed to check quota", err)
				return
			}

			// Check if quota is exceeded
			if !quotaStatus.IsUnlimited && quotaStatus.Remaining <= 0 {
				// Check if overage is allowed
				if !quotaStatus.OverageAllowed {
					httputil.WriteJSON(w, http.StatusTooManyRequests, map[string]interface{}{
						"error":         "LLM call quota exceeded",
						"resource_type": "llm_calls",
						"used":          quotaStatus.Used,
						"limit":         quotaStatus.Limit,
						"reset_date":    quotaStatus.ResetDate,
					})
					return
				}
			}

			// Add quota information to response headers
			w.Header().Set("X-LLM-Quota-Used", strconv.Itoa(quotaStatus.Used))
			w.Header().Set("X-LLM-Quota-Limit", strconv.Itoa(quotaStatus.Limit))
			if quotaStatus.Remaining >= 0 {
				w.Header().Set("X-LLM-Quota-Remaining", strconv.Itoa(quotaStatus.Remaining))
			} else {
				w.Header().Set("X-LLM-Quota-Remaining", "unlimited")
			}
			if quotaStatus.ResetDate != nil {
				w.Header().Set("X-LLM-Quota-Reset", quotaStatus.ResetDate.Format("2006-01-02T15:04:05Z"))
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// TrackUsageMiddleware tracks usage for successful requests
func (m *BillingMiddleware) TrackUsageMiddleware(resourceType application.QuotaResource, increment int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.usageTracker == nil {
				// No usage tracker configured, continue without tracking
				next.ServeHTTP(w, r)
				return
			}

			// Get tenant ID from context
			tenantID, ok := sharedcontext.GetTenantID(r.Context())
			if !ok {
				// Try to get from header as fallback
				tenantIDStr := r.Header.Get("X-PS-Tenant-ID")
				if tenantIDStr == "" {
					next.ServeHTTP(w, r)
					return
				}
				var err error
				tenantID, err = uuid.Parse(tenantIDStr)
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Create a response writer that captures the status code
			var statusCode int = http.StatusOK
			rw := &statusCapturingWriter{
				ResponseWriter: w,
				statusCode:     &statusCode,
			}

			// Call next handler
			next.ServeHTTP(rw, r)

			// Only record usage for successful requests (2xx status codes)
			if statusCode >= 200 && statusCode < 300 {
				// Record usage asynchronously to avoid blocking the response
				go func() {
					switch resourceType {
					case application.QuotaResourceAPICalls:
						_ = m.usageTracker.TrackAPICall(context.Background(), tenantID)
					case application.QuotaResourceLLMCalls:
						_ = m.usageTracker.TrackLLMCall(context.Background(), tenantID)
					case application.QuotaResourceRulepacks:
						_ = m.usageTracker.TrackViolation(context.Background(), tenantID)
					}
				}()
			}
		})
	}
}

// statusCapturingWriter wraps http.ResponseWriter to capture status code
type statusCapturingWriter struct {
	http.ResponseWriter
	statusCode *int
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	*w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
