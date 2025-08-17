package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/promptshield/promptshield/internal/usage"
)

// registerUsageHandlers registers all usage metering endpoints
func registerUsageHandlers(r chi.Router, opt Options) {
	r.Route("/v1/admin/usage", func(ur chi.Router) {
		ur.Use(adminAuth(opt))
		ur.Use(correlationIDMiddleware)
		ur.Use(tenantContextMiddleware)
		
		ur.Get("/minute", getUsageByMinuteHandler(opt))
		ur.Get("/hour", getUsageByHourHandler(opt))
		ur.Get("/day", getUsageByDayHandler(opt))
		ur.Get("/summary", getUsageSummaryHandler(opt))
	})
	
	// Real-time metrics endpoints
	r.Route("/v1/metrics", func(mr chi.Router) {
		mr.Use(userAuth(opt))
		mr.Use(correlationIDMiddleware)
		mr.Use(tenantContextMiddleware)
		
		mr.Get("/stream", getMetricsStreamHandler(opt))
		mr.Get("/requests", getRequestMetricsHandler(opt))
		mr.Get("/tokens", getTokenMetricsHandler(opt))
		mr.Get("/violations", getViolationMetricsHandler(opt))
		mr.Get("/latency", getLatencyMetricsHandler(opt))
	})
	
	// Analytics query endpoint
	r.Route("/v1/analytics", func(ar chi.Router) {
		ar.Use(adminAuth(opt))
		ar.Use(correlationIDMiddleware)
		ar.Use(tenantContextMiddleware)
		
		ar.Post("/query", analyticsQueryHandler(opt))
	})
}

// UsageMetrics represents aggregated usage data
type UsageMetrics struct {
	WindowStart time.Time           `json:"window_start"`
	WindowEnd   time.Time           `json:"window_end"`
	Interval    string              `json:"interval"`
	Rows        []usage.Row         `json:"rows"`
	Summary     *UsageSummary       `json:"summary,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UsageSummary represents summary statistics
type UsageSummary struct {
	TotalRequests int64   `json:"total_requests"`
	TotalBytes    int64   `json:"total_bytes"`
	TotalTokens   int64   `json:"total_tokens,omitempty"`
	AverageRPS    float64 `json:"average_rps"`
	PeakRPS       float64 `json:"peak_rps"`
	TopTenants    []TenantUsage `json:"top_tenants,omitempty"`
}

// TenantUsage represents usage by tenant
type TenantUsage struct {
	TenantID string `json:"tenant_id"`
	Requests int64  `json:"requests"`
	Bytes    int64  `json:"bytes"`
	Tokens   int64  `json:"tokens,omitempty"`
}

// getUsageByMinuteHandler handles GET /v1/admin/usage/minute
func getUsageByMinuteHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics, err := getUsageMetrics(r, opt, usage.IntervalMinute)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				err.Error(), nil, r)
			return
		}
		
		writeJSON(w, http.StatusOK, metrics, r)
	}
}

// getUsageByHourHandler handles GET /v1/admin/usage/hour
func getUsageByHourHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics, err := getUsageMetrics(r, opt, usage.IntervalHour)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				err.Error(), nil, r)
			return
		}
		
		writeJSON(w, http.StatusOK, metrics, r)
	}
}

// getUsageByDayHandler handles GET /v1/admin/usage/day
func getUsageByDayHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics, err := getUsageMetrics(r, opt, usage.IntervalDay)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				err.Error(), nil, r)
			return
		}
		
		writeJSON(w, http.StatusOK, metrics, r)
	}
}

// getUsageSummaryHandler handles GET /v1/admin/usage/summary
func getUsageSummaryHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.UsageStore == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Usage tracking not configured", nil, r)
			return
		}
		
		// Get time range (default to last 24 hours)
		end := time.Now().UTC()
		start := end.Add(-24 * time.Hour)
		
		if startStr := r.URL.Query().Get("start"); startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				start = t
			}
		}
		
		if endStr := r.URL.Query().Get("end"); endStr != "" {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				end = t
			}
		}
		
		// Query usage data with hourly granularity
		query := usage.Query{
			Start:    start,
			End:      end,
			Interval: usage.IntervalHour,
			GroupBy:  []usage.GroupBy{usage.GroupByTenant, usage.GroupByRoute, usage.GroupByDecision},
		}
		
		result, err := opt.UsageStore.Query(r.Context(), query)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to query usage data", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Calculate summary statistics
		summary := calculateUsageSummary(result.Rows, start, end)
		
		response := map[string]interface{}{
			"window_start": start.Format(time.RFC3339),
			"window_end":   end.Format(time.RFC3339),
			"summary":      summary,
			"data_points":  len(result.Rows),
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// getUsageMetrics is a helper function to query usage metrics
func getUsageMetrics(r *http.Request, opt Options, interval usage.Interval) (*UsageMetrics, error) {
	if opt.UsageStore == nil {
		return nil, writeErrorResponse("Usage tracking not configured")
	}
	
	query := r.URL.Query()
	
	// Parse time range
	end := time.Now().UTC()
	var start time.Time
	
	// Set default start time based on interval
	switch interval {
	case usage.IntervalMinute:
		start = end.Add(-1 * time.Hour) // Last hour for minute data
	case usage.IntervalHour:
		start = end.Add(-24 * time.Hour) // Last 24 hours for hour data
	case usage.IntervalDay:
		start = end.Add(-30 * 24 * time.Hour) // Last 30 days for day data
	}
	
	// Override with query parameters if provided
	if startStr := query.Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	
	if endStr := query.Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}
	
	// Validate time range
	if end.Before(start) {
		return nil, writeErrorResponse("End time must be after start time")
	}
	
	// Parse grouping parameters
	var groupBy []usage.GroupBy
	if groupStr := query.Get("group"); groupStr != "" {
		groups := strings.Split(groupStr, ",")
		for _, g := range groups {
			switch strings.TrimSpace(strings.ToLower(g)) {
			case "tenant":
				groupBy = append(groupBy, usage.GroupByTenant)
			case "route":
				groupBy = append(groupBy, usage.GroupByRoute)
			case "decision":
				groupBy = append(groupBy, usage.GroupByDecision)
			}
		}
	}
	
	// Default grouping if none specified
	if len(groupBy) == 0 {
		groupBy = []usage.GroupBy{usage.GroupByTenant}
	}
	
	// Build usage query
	usageQuery := usage.Query{
		Tenant:   query.Get("tenant"),
		Start:    start,
		End:      end,
		Interval: interval,
		GroupBy:  groupBy,
	}
	
	// Query usage data
	result, err := opt.UsageStore.Query(r.Context(), usageQuery)
	if err != nil {
		return nil, err
	}
	
	// Build response
	metrics := &UsageMetrics{
		WindowStart: result.WindowStart,
		WindowEnd:   result.WindowEnd,
		Interval:    string(interval),
		Rows:        result.Rows,
		Metadata: map[string]interface{}{
			"query": map[string]interface{}{
				"tenant":   usageQuery.Tenant,
				"group_by": groupBy,
			},
		},
	}
	
	// Add summary if requested
	if query.Get("include_summary") == "true" {
		metrics.Summary = calculateUsageSummary(result.Rows, start, end)
	}
	
	return metrics, nil
}

// calculateUsageSummary calculates summary statistics from usage rows
func calculateUsageSummary(rows []usage.Row, start, end time.Time) *UsageSummary {
	var totalRequests, totalBytes int64
	var maxRPS float64
	tenantMap := make(map[string]*TenantUsage)
	
	for _, row := range rows {
		totalRequests += row.Count
		totalBytes += row.Bytes
		
		// Calculate RPS for this row (approximate)
		duration := end.Sub(start).Seconds()
		if duration > 0 {
			rps := float64(row.Count) / duration
			if rps > maxRPS {
				maxRPS = rps
			}
		}
		
		// Aggregate by tenant
		if row.Tenant != "" {
			if tenant, exists := tenantMap[row.Tenant]; exists {
				tenant.Requests += row.Count
				tenant.Bytes += row.Bytes
			} else {
				tenantMap[row.Tenant] = &TenantUsage{
					TenantID: row.Tenant,
					Requests: row.Count,
					Bytes:    row.Bytes,
				}
			}
		}
	}
	
	// Convert tenant map to sorted slice (top 10)
	var topTenants []TenantUsage
	for _, tenant := range tenantMap {
		topTenants = append(topTenants, *tenant)
	}
	
	// Sort by request count (descending) and take top 10
	// TODO: Implement proper sorting
	if len(topTenants) > 10 {
		topTenants = topTenants[:10]
	}
	
	duration := end.Sub(start).Seconds()
	avgRPS := 0.0
	if duration > 0 {
		avgRPS = float64(totalRequests) / duration
	}
	
	return &UsageSummary{
		TotalRequests: totalRequests,
		TotalBytes:    totalBytes,
		AverageRPS:    avgRPS,
		PeakRPS:       maxRPS,
		TopTenants:    topTenants,
	}
}

// Placeholder handlers for real-time metrics
func getMetricsStreamHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Metrics streaming not yet implemented", nil, r)
	}
}

func getRequestMetricsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Request metrics not yet implemented", nil, r)
	}
}

func getTokenMetricsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Token metrics not yet implemented", nil, r)
	}
}

func getViolationMetricsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Violation metrics not yet implemented", nil, r)
	}
}

func getLatencyMetricsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Latency metrics not yet implemented", nil, r)
	}
}

func analyticsQueryHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Analytics query not yet implemented", nil, r)
	}
}

// Helper function to create error responses
func writeErrorResponse(message string) error {
	return &ApiError{Message: message}
}

type ApiError struct {
	Message string
}

func (e *ApiError) Error() string {
	return e.Message
}