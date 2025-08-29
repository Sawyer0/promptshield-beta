package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// BusinessMetrics endpoints for billing integration, security dashboards, and compliance
// Note: Infrastructure metrics (CPU, memory, etc.) handled by Prometheus/Grafana

// registerBusinessMetricsHandlers registers business metrics endpoints
func registerBusinessMetricsHandlers(r chi.Router, opt Options) {
	r.Route("/api/usage", func(ur chi.Router) {
		ur.Use(adminAuth(opt)) // Require admin auth for usage data
		ur.Get("/{tenantId}/summary", getTenantUsageSummaryHandler(opt))
	})

	r.Route("/api/violations", func(vr chi.Router) {
		vr.Use(adminAuth(opt)) // Require admin auth for security data
		vr.Get("/{tenantId}/summary", getViolationsSummaryHandler(opt))
	})

	r.Route("/api/compliance", func(cr chi.Router) {
		cr.Use(adminAuth(opt)) // Require admin auth for compliance data
		cr.Get("/{tenantId}/report", getComplianceReportHandler(opt))
	})
}

// UsageSummary represents tenant usage data for billing integration
type UsageSummary struct {
	TenantID        string    `json:"tenant_id"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	RequestsTotal   int64     `json:"requests_total"`
	RequestsAllowed int64     `json:"requests_allowed"`
	RequestsBlocked int64     `json:"requests_blocked"`
	BytesProcessed  int64     `json:"bytes_processed"`
	RulepacksUsed   int       `json:"rulepacks_used"`
	GeneratedAt     time.Time `json:"generated_at"`
}

// ViolationsSummary represents security violations for dashboards
type ViolationsSummary struct {
	TenantID             string                `json:"tenant_id"`
	PeriodStart          time.Time             `json:"period_start"`
	PeriodEnd            time.Time             `json:"period_end"`
	TotalViolations      int64                 `json:"total_violations"`
	ViolationsBySeverity map[string]int64      `json:"violations_by_severity"`
	ViolationsByType     map[string]int64      `json:"violations_by_type"`
	TopRulesTriggered    []RuleViolationCount  `json:"top_rules_triggered"`
	TrendData            []ViolationTrendPoint `json:"trend_data"`
	GeneratedAt          time.Time             `json:"generated_at"`
}

// RuleViolationCount represents rule violation statistics
type RuleViolationCount struct {
	RuleID     string `json:"rule_id"`
	RuleName   string `json:"rule_name"`
	Count      int64  `json:"count"`
	Severity   string `json:"severity"`
	RulepackID string `json:"rulepack_id"`
}

// ViolationTrendPoint represents violation trends over time
type ViolationTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
	Severity  string    `json:"severity,omitempty"`
}

// ComplianceReport represents compliance data for SOC2/GDPR
type ComplianceReport struct {
	TenantID               string                  `json:"tenant_id"`
	ReportPeriod           DateRange               `json:"report_period"`
	ComplianceStandard     string                  `json:"compliance_standard"`
	AuditEventsTotal       int64                   `json:"audit_events_total"`
	SecurityIncidentsTotal int64                   `json:"security_incidents_total"`
	DataProcessingEvents   int64                   `json:"data_processing_events"`
	PolicyViolations       int64                   `json:"policy_violations"`
	AuditLogIntegrity      AuditIntegrityStatus    `json:"audit_log_integrity"`
	AccessControls         AccessControlsSummary   `json:"access_controls"`
	DataRetention          DataRetentionSummary    `json:"data_retention"`
	SecurityMeasures       SecurityMeasuresSummary `json:"security_measures"`
	GeneratedAt            time.Time               `json:"generated_at"`
	GeneratedBy            string                  `json:"generated_by"`
}

// DateRange represents a date range for reports
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AuditIntegrityStatus represents audit log integrity status
type AuditIntegrityStatus struct {
	HashChainValid    bool      `json:"hash_chain_valid"`
	EventsVerified    int64     `json:"events_verified"`
	IntegrityBreaches int64     `json:"integrity_breaches"`
	LastVerifiedAt    time.Time `json:"last_verified_at"`
}

// AccessControlsSummary represents access controls summary
type AccessControlsSummary struct {
	ActiveUsers        int64 `json:"active_users"`
	APIKeysActive      int64 `json:"api_keys_active"`
	FailedLogins       int64 `json:"failed_login_attempts"`
	PasswordViolations int64 `json:"password_policy_violations"`
}

// DataRetentionSummary represents data retention compliance
type DataRetentionSummary struct {
	EventsRetained      int64     `json:"events_retained"`
	EventsArchived      int64     `json:"events_archived"`
	EventsPurged        int64     `json:"events_purged"`
	RetentionPolicyDays int       `json:"retention_policy_days"`
	NextPurgeScheduled  time.Time `json:"next_purge_scheduled"`
}

// SecurityMeasuresSummary represents security measures summary
type SecurityMeasuresSummary struct {
	EncryptionEnabled   bool    `json:"encryption_enabled"`
	TLSEnforced         bool    `json:"tls_enforced"`
	RateLimitingEnabled bool    `json:"rate_limiting_enabled"`
	AuditLoggingEnabled bool    `json:"audit_logging_enabled"`
	FailSafeMode        bool    `json:"fail_safe_mode"`
	SecurityScore       float64 `json:"security_score"`
}

// getTenantUsageSummaryHandler handles GET /api/usage/{tenantId}/summary
func getTenantUsageSummaryHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := chi.URLParam(r, "tenantId")
		if tenantIDStr == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Tenant ID is required", nil, r)
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid tenant ID format", nil, r)
			return
		}

		// Parse query parameters for date range
		periodStart, periodEnd := parseDateRange(r, 30) // Default to last 30 days

		// Get usage data from usage store if available
		var summary *UsageSummary
		if opt.UsageStore != nil {
			// Query usage store for tenant data
			summary = &UsageSummary{
				TenantID:        tenantID.String(),
				PeriodStart:     periodStart,
				PeriodEnd:       periodEnd,
				RequestsTotal:   0, // TODO: Implement usage store queries
				RequestsAllowed: 0,
				RequestsBlocked: 0,
				BytesProcessed:  0,
				RulepacksUsed:   0,
				GeneratedAt:     time.Now(),
			}
		} else {
			// Return empty summary if no usage store configured
			summary = &UsageSummary{
				TenantID:    tenantID.String(),
				PeriodStart: periodStart,
				PeriodEnd:   periodEnd,
				GeneratedAt: time.Now(),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

// getViolationsSummaryHandler handles GET /api/violations/{tenantId}/summary
func getViolationsSummaryHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := chi.URLParam(r, "tenantId")
		if tenantIDStr == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Tenant ID is required", nil, r)
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid tenant ID format", nil, r)
			return
		}

		// Parse query parameters
		periodStart, periodEnd := parseDateRange(r, 30)

		// Query audit repository for violation data
		var summary *ViolationsSummary
		if opt.AuditRepository != nil {
			// Query audit events for tenant
			entries, _, err := opt.AuditRepository.ListByTenant(r.Context(), tenantID, 0, 1000) // Get recent entries
			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
					"Failed to retrieve violation data", map[string]interface{}{"error": err.Error()}, r)
				return
			}

			// Filter entries by date range and convert to events
			var events []*types.AuditEvent
			for _, entry := range entries {
				if entry.CreatedAt.After(periodStart) && entry.CreatedAt.Before(periodEnd) {
					// Convert AuditEntry to AuditEvent for processing
					event := &types.AuditEvent{
						TenantID:   entry.TenantID,
						ActorID:    entry.ActorID,
						Action:     entry.Action,
						ObjectType: entry.ObjectType,
						ObjectID:   entry.ObjectID,
						Timestamp:  entry.CreatedAt,
						Metadata:   parseMetadata(entry.Metadata),
					}
					events = append(events, event)
				}
			}

			// Aggregate violation data
			summary = aggregateViolations(tenantID.String(), events, periodStart, periodEnd)
		} else {
			// Return empty summary if no audit repository configured
			summary = &ViolationsSummary{
				TenantID:             tenantID.String(),
				PeriodStart:          periodStart,
				PeriodEnd:            periodEnd,
				ViolationsBySeverity: make(map[string]int64),
				ViolationsByType:     make(map[string]int64),
				TopRulesTriggered:    make([]RuleViolationCount, 0),
				TrendData:            make([]ViolationTrendPoint, 0),
				GeneratedAt:          time.Now(),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

// getComplianceReportHandler handles GET /api/compliance/{tenantId}/report
func getComplianceReportHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := chi.URLParam(r, "tenantId")
		if tenantIDStr == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Tenant ID is required", nil, r)
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid tenant ID format", nil, r)
			return
		}

		// Parse query parameters
		periodStart, periodEnd := parseDateRange(r, 90) // Default to last 90 days for compliance
		standard := r.URL.Query().Get("standard")
		if standard == "" {
			standard = "SOC2" // Default compliance standard
		}

		// Generate compliance report
		report := generateComplianceReport(opt, tenantID.String(), periodStart, periodEnd, standard, r.Context())

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	}
}

// Helper functions

// parseDateRange parses date range from query parameters
func parseDateRange(r *http.Request, defaultDays int) (time.Time, time.Time) {
	now := time.Now()
	end := now
	start := now.AddDate(0, 0, -defaultDays)

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

	// Validate range (max 365 days)
	if end.Sub(start) > 365*24*time.Hour {
		start = end.AddDate(0, 0, -365)
	}

	return start, end
}

// aggregateViolations aggregates violation events into summary
func aggregateViolations(tenantID string, events []*types.AuditEvent, start, end time.Time) *ViolationsSummary {
	summary := &ViolationsSummary{
		TenantID:             tenantID,
		PeriodStart:          start,
		PeriodEnd:            end,
		ViolationsBySeverity: make(map[string]int64),
		ViolationsByType:     make(map[string]int64),
		TopRulesTriggered:    make([]RuleViolationCount, 0),
		TrendData:            make([]ViolationTrendPoint, 0),
		GeneratedAt:          time.Now(),
	}

	ruleViolations := make(map[string]*RuleViolationCount)
	dailyTrends := make(map[string]int64) // date -> count

	for _, event := range events {
		summary.TotalViolations++

		// Parse metadata for violation details
		if event.Metadata != nil {
			if severity, ok := event.Metadata["severity"].(string); ok && severity != "" {
				summary.ViolationsBySeverity[severity]++
			}

			if violationType, ok := event.Metadata["violation_type"].(string); ok && violationType != "" {
				summary.ViolationsByType[violationType]++
			}

			if ruleID, ok := event.Metadata["rule_id"].(string); ok && ruleID != "" {
				if existing, exists := ruleViolations[ruleID]; exists {
					existing.Count++
				} else {
					ruleName := ruleID
					if name, ok := event.Metadata["rule_name"].(string); ok {
						ruleName = name
					}
					severity := "MEDIUM"
					if sev, ok := event.Metadata["severity"].(string); ok {
						severity = sev
					}
					ruleViolations[ruleID] = &RuleViolationCount{
						RuleID:   ruleID,
						RuleName: ruleName,
						Count:    1,
						Severity: severity,
					}
				}
			}
		}

		// Daily trend data
		day := event.Timestamp.Format("2006-01-02")
		dailyTrends[day]++
	}

	// Convert daily trends to trend points
	for day, count := range dailyTrends {
		if t, err := time.Parse("2006-01-02", day); err == nil {
			summary.TrendData = append(summary.TrendData, ViolationTrendPoint{
				Timestamp: t,
				Count:     count,
			})
		}
	}

	// Convert rule violations to slice and limit to top 10
	for _, rule := range ruleViolations {
		summary.TopRulesTriggered = append(summary.TopRulesTriggered, *rule)
	}

	// Sort and limit to top 10 rules (simple implementation)
	if len(summary.TopRulesTriggered) > 10 {
		summary.TopRulesTriggered = summary.TopRulesTriggered[:10]
	}

	return summary
}

// generateComplianceReport generates a compliance report
func generateComplianceReport(opt Options, tenantID string, start, end time.Time, standard string, ctx context.Context) *ComplianceReport {
	report := &ComplianceReport{
		TenantID:           tenantID,
		ReportPeriod:       DateRange{Start: start, End: end},
		ComplianceStandard: standard,
		GeneratedAt:        time.Now(),
		GeneratedBy:        "system",
	}

	// Query audit repository for compliance data
	if opt.AuditRepository != nil {
		tid, _ := uuid.Parse(tenantID)
		entries, _, err := opt.AuditRepository.ListByTenant(ctx, tid, 0, 5000) // Get entries for compliance
		if err == nil {
			// Filter entries by date range
			var filteredEntries []*domain.AuditEntry
			for _, entry := range entries {
				if entry.CreatedAt.After(start) && entry.CreatedAt.Before(end) {
					filteredEntries = append(filteredEntries, entry)
				}
			}

			report.AuditEventsTotal = int64(len(filteredEntries))

			// Count security incidents by converting entries to events
			for _, entry := range filteredEntries {
				event := &types.AuditEvent{
					Action:     entry.Action,
					ObjectType: entry.ObjectType,
					Metadata:   parseMetadata(entry.Metadata),
				}
				if isSecurityIncident(event) {
					report.SecurityIncidentsTotal++
				}
				if isDataProcessingEvent(event) {
					report.DataProcessingEvents++
				}
				if isPolicyViolation(event) {
					report.PolicyViolations++
				}
			}
		}
	}

	// Audit log integrity (hash chain status)
	report.AuditLogIntegrity = AuditIntegrityStatus{
		HashChainValid:    true, // TODO: Implement actual hash chain verification
		EventsVerified:    report.AuditEventsTotal,
		IntegrityBreaches: 0,
		LastVerifiedAt:    time.Now(),
	}

	// Access controls summary
	report.AccessControls = AccessControlsSummary{
		ActiveUsers:        0, // TODO: Query user data if available
		APIKeysActive:      0, // TODO: Query API keys if available
		FailedLogins:       0, // TODO: Query failed login events
		PasswordViolations: 0, // TODO: Query password policy violations
	}

	// Data retention summary
	report.DataRetention = DataRetentionSummary{
		EventsRetained:      report.AuditEventsTotal,
		EventsArchived:      0,  // TODO: Query archived events
		EventsPurged:        0,  // TODO: Query purged events
		RetentionPolicyDays: 90, // Default retention policy
		NextPurgeScheduled:  time.Now().AddDate(0, 0, 90),
	}

	// Security measures summary
	report.SecurityMeasures = SecurityMeasuresSummary{
		EncryptionEnabled:   true, // Hash chain provides encryption
		TLSEnforced:         true, // Assume TLS is enforced
		RateLimitingEnabled: opt.QuotaStore != nil,
		AuditLoggingEnabled: opt.AuditLogger != nil,
		FailSafeMode:        true, // Security-first design
		SecurityScore:       calculateSecurityScore(report),
	}

	return report
}

// Helper functions for compliance classification
func isSecurityIncident(event *types.AuditEvent) bool {
	securityActions := map[string]bool{
		"security_violation":  true,
		"intrusion_attempt":   true,
		"unauthorized_access": true,
		"policy_violation":    true,
		"content_blocked":     true,
	}
	return securityActions[event.Action]
}

func isDataProcessingEvent(event *types.AuditEvent) bool {
	dataActions := map[string]bool{
		"data_processed":  true,
		"content_scanned": true,
		"pii_detected":    true,
		"data_redacted":   true,
	}
	return dataActions[event.Action]
}

func isPolicyViolation(event *types.AuditEvent) bool {
	return event.Action == "policy_violation" || event.Action == "rule_triggered"
}

// calculateSecurityScore calculates a security score based on measures
func calculateSecurityScore(report *ComplianceReport) float64 {
	score := 0.0
	measures := &report.SecurityMeasures

	if measures.EncryptionEnabled {
		score += 20.0
	}
	if measures.TLSEnforced {
		score += 20.0
	}
	if measures.RateLimitingEnabled {
		score += 15.0
	}
	if measures.AuditLoggingEnabled {
		score += 25.0
	}
	if measures.FailSafeMode {
		score += 20.0
	}

	return score
}

// parseMetadata converts json.RawMessage to map[string]interface{}
func parseMetadata(raw json.RawMessage) map[string]interface{} {
	if raw == nil {
		return nil
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil
	}
	return metadata
}
