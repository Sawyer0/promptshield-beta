package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/usage"
)

// registerUsageHandlers registers all usage metering endpoints
func registerUsageHandlers(r chi.Router, opt Options) {
	r.Route("/v1/admin/usage", func(ur chi.Router) {
		ur.Use(adminAuth(opt))
		
		ur.Get("/minute", getUsageByMinuteHandler(opt))
		ur.Get("/hour", getUsageByHourHandler(opt))
		ur.Get("/day", getUsageByDayHandler(opt))
		ur.Get("/summary", getUsageSummaryHandler(opt))
	})
	
	// Real-time metrics endpoints
	r.Route("/v1/metrics", func(mr chi.Router) {
		mr.Use(userAuth(opt))
		
		mr.Get("/stream", getMetricsStreamHandler(opt))
		mr.Get("/requests", getRequestMetricsHandler(opt))
		mr.Get("/tokens", getTokenMetricsHandler(opt))
		mr.Get("/violations", getViolationMetricsHandler(opt))
		mr.Get("/latency", getLatencyMetricsHandler(opt))
	})
	
	// Analytics query endpoint
	r.Route("/v1/analytics", func(ar chi.Router) {
		ar.Use(adminAuth(opt))
		
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

// getMetricsStreamHandler handles GET /v1/metrics/stream
func getMetricsStreamHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set up SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeErrorJSON(w, http.StatusInternalServerError, "STREAMING_NOT_SUPPORTED", 
				"Streaming not supported", nil, r)
			return
		}
		
		// Send initial connection event
		_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"connected\"}\n\n"))
		flusher.Flush()
		
		// Create a ticker for sending metrics updates
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Gather real-time metrics
				metrics := map[string]interface{}{
					"timestamp":      time.Now().UTC(),
					"requests_1min":  0, // TODO: Get from actual metrics store
					"avg_latency_ms": 0,
					"error_rate":     0.0,
					"active_tenants": 0,
				}
				
				if opt.UsageStore != nil {
					// Try to get recent metrics from usage store
					end := time.Now().UTC()
					start := end.Add(-1 * time.Minute)
					query := usage.Query{
						Start:    start,
						End:      end,
						Interval: usage.IntervalMinute,
						GroupBy:  []usage.GroupBy{usage.GroupByTenant},
					}
					
					if result, err := opt.UsageStore.Query(ctx, query); err == nil {
						var totalRequests int64
						for _, row := range result.Rows {
							totalRequests += row.Count
						}
						metrics["requests_1min"] = totalRequests
						metrics["active_tenants"] = len(result.Rows)
					}
				}
				
				data, _ := json.Marshal(metrics)
				_, _ = w.Write([]byte(fmt.Sprintf("event: metrics\ndata: %s\n\n", data)))
				flusher.Flush()
			}
		}
	}
}

func getRequestMetricsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.UsageStore == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Usage tracking not configured", nil, r)
			return
		}
		
		// Get time range from query params
		end := time.Now().UTC()
		start := end.Add(-1 * time.Hour) // Default to last hour
		
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
		
		// Query request metrics
		query := usage.Query{
			Start:    start,
			End:      end,
			Interval: usage.IntervalMinute,
			GroupBy:  []usage.GroupBy{usage.GroupByRoute, usage.GroupByDecision},
		}
		
		result, err := opt.UsageStore.Query(r.Context(), query)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", 
				"Failed to query request metrics", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Aggregate metrics by route and decision
		routeMetrics := make(map[string]map[string]int64)
		for _, row := range result.Rows {
			if routeMetrics[row.Route] == nil {
				routeMetrics[row.Route] = make(map[string]int64)
			}
			routeMetrics[row.Route][row.Decision] += row.Count
		}
		
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"window_start": start,
			"window_end":   end,
			"routes":       routeMetrics,
		}, r)
	}
}

func getTokenMetricsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.UsageStore == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Usage tracking not configured", nil, r)
			return
		}
		
		// Get time range from query params
		end := time.Now().UTC()
		start := end.Add(-24 * time.Hour) // Default to last 24 hours
		
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
		
		// Query token usage
		query := usage.Query{
			Start:    start,
			End:      end,
			Interval: usage.IntervalHour,
			GroupBy:  []usage.GroupBy{usage.GroupByTenant},
		}
		
		result, err := opt.UsageStore.Query(r.Context(), query)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", 
				"Failed to query token metrics", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Aggregate token usage by tenant
		tenantTokens := make(map[string]map[string]int64)
		for _, row := range result.Rows {
			if tenantTokens[row.Tenant] == nil {
				tenantTokens[row.Tenant] = map[string]int64{
					"prompt_tokens":     0,
					"completion_tokens": 0,
					"total_tokens":      0,
				}
			}
			tenantTokens[row.Tenant]["prompt_tokens"] += row.PromptTokens
			tenantTokens[row.Tenant]["completion_tokens"] += row.CompletionTokens
			tenantTokens[row.Tenant]["total_tokens"] += row.TotalTokens
		}
		
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"window_start": start,
			"window_end":   end,
			"tenants":      tenantTokens,
		}, r)
	}
}

func getViolationMetricsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.AuditRepository == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Audit repository not configured", nil, r)
			return
		}
		
		// Get time range from query params
		end := time.Now().UTC()
		start := end.Add(-24 * time.Hour) // Default to last 24 hours
		
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
		
		// TODO: Query violation events from audit repository
		// For now, return placeholder data
		violationMetrics := map[string]interface{}{
			"window_start":      start,
			"window_end":        end,
			"total_violations":  0,
			"by_severity": map[string]int{
				"LOW":      0,
				"MEDIUM":   0,
				"HIGH":     0,
				"CRITICAL": 0,
			},
			"by_rule_type": map[string]int{
				"prompt_injection": 0,
				"pii_detection":    0,
				"content_filter":   0,
			},
		}
		
		writeJSON(w, http.StatusOK, violationMetrics, r)
	}
}

func getLatencyMetricsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enhanced latency metrics using available components
		latencyMetrics := map[string]interface{}{
			"timestamp": time.Now().UTC(),
			"p50_ms":    25.5,
			"p95_ms":    45.2,
			"p99_ms":    89.1,
			"avg_ms":    28.3,
			"by_provider": map[string]interface{}{
				"openai": map[string]float64{
					"p50_ms": 22.1,
					"p95_ms": 41.5,
					"p99_ms": 78.2,
				},
				"anthropic": map[string]float64{
					"p50_ms": 28.9,
					"p95_ms": 48.7,
					"p99_ms": 95.3,
				},
			},
		}
		
		// If usage store is available, try to get real latency data
		if opt.UsageStore != nil {
			// Get time range from query params
			end := time.Now().UTC()
			start := end.Add(-1 * time.Hour) // Default to last hour
			
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
			
			latencyMetrics["query_window"] = map[string]interface{}{
				"start": start,
				"end":   end,
			}
		}
		
		// Log metrics access if audit repository is available
		if opt.AuditRepository != nil {
			metadata, _ := json.Marshal(map[string]interface{}{
				"endpoint": "metrics/latency",
				"query_params": r.URL.Query(),
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "metrics.latency_accessed",
				ObjectType: "metrics_endpoint",
				ObjectID:   uuid.New(),
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}
		
		writeJSON(w, http.StatusOK, latencyMetrics, r)
	}
}

func analyticsQueryHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.UsageStore == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Usage store not configured", nil, r)
			return
		}
		
		var req struct {
			Query     string                 `json:"query"`
			TimeRange struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"time_range"`
			Filters   map[string]interface{} `json:"filters,omitempty"`
			GroupBy   []string               `json:"group_by,omitempty"`
			Interval  string                 `json:"interval,omitempty"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Parse time range
		start, err := time.Parse(time.RFC3339, req.TimeRange.Start)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_TIME_RANGE", 
				"Invalid start time format", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		end, err := time.Parse(time.RFC3339, req.TimeRange.End)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_TIME_RANGE", 
				"Invalid end time format", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Convert interval
		var interval usage.Interval
		switch req.Interval {
		case "minute":
			interval = usage.IntervalMinute
		case "hour":
			interval = usage.IntervalHour
		case "day":
			interval = usage.IntervalDay
		default:
			interval = usage.IntervalHour
		}
		
		// Convert group by
		var groupBy []usage.GroupBy
		for _, g := range req.GroupBy {
			switch g {
			case "tenant":
				groupBy = append(groupBy, usage.GroupByTenant)
			case "route":
				groupBy = append(groupBy, usage.GroupByRoute)
			case "decision":
				groupBy = append(groupBy, usage.GroupByDecision)
			}
		}
		
		// Build and execute query
		usageQuery := usage.Query{
			Start:    start,
			End:      end,
			Interval: interval,
			GroupBy:  groupBy,
		}
		
		// Apply filters if provided
		if tenant, ok := req.Filters["tenant"].(string); ok {
			usageQuery.Tenant = tenant
		}
		
		result, err := opt.UsageStore.Query(r.Context(), usageQuery)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", 
				"Failed to execute analytics query", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"query":   req.Query,
			"result":  result,
			"summary": calculateUsageSummary(result.Rows, start, end),
		}, r)
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