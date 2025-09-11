package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditEntry represents an immutable audit log entry
// From domain/models.go
type AuditEntry struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   *uuid.UUID      `json:"tenant_id,omitempty"`
	ActorID    *uuid.UUID      `json:"actor_id,omitempty"`
	ActorType  ActorType       `json:"actor_type"`
	ActorEmail string          `json:"actor_email,omitempty"`
	Action     string          `json:"action"`      // tenant.create, key.rotate, etc.
	ObjectType string          `json:"object_type"` // tenant, key, policy, etc.
	ObjectID   uuid.UUID       `json:"object_id"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	Hash       string          `json:"hash"`      // SHA-256 of entry
	PrevHash   string          `json:"prev_hash"` // Chain to previous
}

// ActorType represents who performed an action
type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeSystem ActorType = "system"
	ActorTypeAPIKey ActorType = "api_key"
)

// AuditEvent represents a structured audit event before persistence
// Enhanced from internal/audit/logger.go Event
type AuditEvent struct {
	TenantID   *uuid.UUID             `json:"tenant_id,omitempty"`
	ActorID    *uuid.UUID             `json:"actor_id,omitempty"`
	ActorType  ActorType              `json:"actor_type"`
	ActorEmail string                 `json:"actor_email,omitempty"`
	Action     string                 `json:"action"`
	ObjectType string                 `json:"object_type"`
	ObjectID   uuid.UUID              `json:"object_id"`
	Before     map[string]interface{} `json:"before,omitempty"`
	After      map[string]interface{} `json:"after,omitempty"`
	Hash       string                 `json:"hash,omitempty"`
	PrevHash   string                 `json:"prev_hash,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
}

// AuditAction represents standardized audit actions
type AuditAction string

const (
	// Tenant actions
	AuditActionTenantCreate  AuditAction = "tenant.create"
	AuditActionTenantUpdate  AuditAction = "tenant.update"
	AuditActionTenantSuspend AuditAction = "tenant.suspend"
	AuditActionTenantRestore AuditAction = "tenant.restore"
	AuditActionTenantDelete  AuditAction = "tenant.delete"

	// Policy actions
	AuditActionPolicyCreate   AuditAction = "policy.create"
	AuditActionPolicyUpdate   AuditAction = "policy.update"
	AuditActionPolicyAssign   AuditAction = "policy.assign"
	AuditActionPolicyUnassign AuditAction = "policy.unassign"
	AuditActionPolicyDelete   AuditAction = "policy.delete"

	// Provider key actions
	AuditActionProviderKeyAdd    AuditAction = "provider_key.add"
	AuditActionProviderKeyRotate AuditAction = "provider_key.rotate"
	AuditActionProviderKeyRevoke AuditAction = "provider_key.revoke"

	// API token actions
	AuditActionAPITokenCreate AuditAction = "api_token.create"
	AuditActionAPITokenRevoke AuditAction = "api_token.revoke"

	// Quota actions
	AuditActionQuotaUpdate AuditAction = "quota.update"
	AuditActionQuotaExceed AuditAction = "quota.exceed"

	// Security actions
	AuditActionViolationDetected AuditAction = "violation.detected"
	AuditActionRequestBlocked    AuditAction = "request.blocked"
	AuditActionRequestAllowed    AuditAction = "request.allowed"
)

// ObjectType represents the type of object being audited
type ObjectType string

const (
	ObjectTypeTenant      ObjectType = "tenant"
	ObjectTypePolicy      ObjectType = "policy"
	ObjectTypeProviderKey ObjectType = "provider_key"
	ObjectTypeAPIToken    ObjectType = "api_token"
	ObjectTypeQuota       ObjectType = "quota"
	ObjectTypeRequest     ObjectType = "request"
	ObjectTypeViolation   ObjectType = "violation"
)

// AuditMetadata provides structured metadata for audit entries
type AuditMetadata struct {
	RequestID    string                 `json:"request_id,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	Provider     Provider               `json:"provider,omitempty"`
	Endpoint     string                 `json:"endpoint,omitempty"`
	HTTPMethod   string                 `json:"http_method,omitempty"`
	StatusCode   int                    `json:"status_code,omitempty"`
	ProcessingMs int64                  `json:"processing_ms,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// AuditConfig represents audit configuration
type AuditConfig struct {
	Enabled       bool                   `json:"enabled"`
	Level         string                 `json:"level"`  // "debug", "info", "warn", "error"
	Format        string                 `json:"format"` // "json", "text"
	Output        string                 `json:"output"` // "stdout", "file", "syslog"
	FilePath      string                 `json:"file_path,omitempty"`
	MaxSize       int64                  `json:"max_size,omitempty"`
	MaxAge        time.Duration          `json:"max_age,omitempty"`
	MaxBackups    int                    `json:"max_backups,omitempty"`
	Compress      bool                   `json:"compress,omitempty"`
	BatchSize     int                    `json:"batch_size,omitempty"`
	FlushInterval time.Duration          `json:"flush_interval,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AuditVerification represents audit log integrity verification
type AuditVerification struct {
	TimeRange       TimeRange              `json:"time_range"`
	TotalEvents     int64                  `json:"total_events"`
	ValidEvents     int64                  `json:"valid_events"`
	InvalidEvents   int64                  `json:"invalid_events"`
	MissingEvents   []string               `json:"missing_events,omitempty"`
	CorruptedEvents []string               `json:"corrupted_events,omitempty"`
	HashChainValid  bool                   `json:"hash_chain_valid"`
	VerifiedAt      time.Time              `json:"verified_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// AuditTrailSummary represents a summary of an audit trail
type AuditTrailSummary struct {
	EntityID    string                 `json:"entity_id"`
	EntityType  string                 `json:"entity_type"`
	TotalEvents int64                  `json:"total_events"`
	FirstEvent  *AuditEvent            `json:"first_event,omitempty"`
	LastEvent   *AuditEvent            `json:"last_event,omitempty"`
	TimeRange   TimeRange              `json:"time_range"`
	EventTypes  map[string]int64       `json:"event_types"`
	Users       []string               `json:"users,omitempty"`
	Resources   []string               `json:"resources,omitempty"`
	Actions     []string               `json:"actions,omitempty"`
	RiskScore   float64                `json:"risk_score,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TrailValidation represents audit trail validation results
type TrailValidation struct {
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	IsValid       bool                   `json:"is_valid"`
	TotalEvents   int64                  `json:"total_events"`
	ValidEvents   int64                  `json:"valid_events"`
	InvalidEvents int64                  `json:"invalid_events"`
	Errors        []string               `json:"errors,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	ValidatedAt   time.Time              `json:"validated_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AuditReportConfig represents configuration for audit report generation
type AuditReportConfig struct {
	ReportType     string                 `json:"report_type"`
	TimeRange      TimeRange              `json:"time_range"`
	Format         string                 `json:"format"` // "pdf", "html", "json", "csv"
	Filters        *AuditFilter           `json:"filters,omitempty"`
	GroupBy        []string               `json:"group_by,omitempty"`
	SortBy         string                 `json:"sort_by,omitempty"`
	SortOrder      string                 `json:"sort_order,omitempty"`
	IncludeDetails bool                   `json:"include_details,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// AuditReport represents an audit report
type AuditReport struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	TimeRange   TimeRange              `json:"time_range"`
	GeneratedAt time.Time              `json:"generated_at"`
	GeneratedBy string                 `json:"generated_by"`
	TotalEvents int64                  `json:"total_events"`
	Summary     map[string]interface{} `json:"summary"`
	Details     []*AuditEvent          `json:"details,omitempty"`
	Charts      []*ReportChart         `json:"charts,omitempty"`
	Format      string                 `json:"format"`
	Size        int64                  `json:"size,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceReport represents a compliance-specific audit report
type ComplianceReport struct {
	ID               string                   `json:"id"`
	TenantID         string                   `json:"tenant_id"`
	Standard         string                   `json:"standard"` // "SOC2", "HIPAA", "GDPR", etc.
	TimeRange        TimeRange                `json:"time_range"`
	GeneratedAt      time.Time                `json:"generated_at"`
	GeneratedBy      string                   `json:"generated_by"`
	ComplianceStatus string                   `json:"compliance_status"` // "compliant", "non-compliant", "partial"
	ComplianceScore  float64                  `json:"compliance_score"`
	TotalEvents      int64                    `json:"total_events"`
	EventsByType     map[string]int64         `json:"events_by_type,omitempty"`
	PolicyChangeCount int64                   `json:"policy_change_count"`
	DataAccessCount  int64                    `json:"data_access_count"`
	UserActivityCount int64                   `json:"user_activity_count"`
	Requirements     []*ComplianceRequirement `json:"requirements"`
	Violations       []*ComplianceViolation   `json:"violations,omitempty"`
	Recommendations  []string                 `json:"recommendations,omitempty"`
	Evidence         map[string]interface{}   `json:"evidence,omitempty"`
	Metadata         map[string]interface{}   `json:"metadata,omitempty"`
}

// ActivitySummary represents a summary of activity for a time period
type ActivitySummary struct {
	TimeRange       TimeRange              `json:"time_range"`
	TotalEvents     int64                  `json:"total_events"`
	UniqueUsers     int64                  `json:"unique_users"`
	UniqueResources int64                  `json:"unique_resources"`
	EventTypes      map[string]int64       `json:"event_types"`
	TopUsers        []*UserActivity        `json:"top_users,omitempty"`
	TopResources    []*ResourceActivity    `json:"top_resources,omitempty"`
	TopActions      []*ActionActivity      `json:"top_actions,omitempty"`
	RiskMetrics     *RiskMetrics           `json:"risk_metrics,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// AuditReportSchedule represents a scheduled audit report
type AuditReportSchedule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	ReportType  string                 `json:"report_type"`
	Schedule    string                 `json:"schedule"` // cron expression
	Config      *AuditReportConfig     `json:"config"`
	Enabled     bool                   `json:"enabled"`
	LastRun     *time.Time             `json:"last_run,omitempty"`
	NextRun     *time.Time             `json:"next_run,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditPatternAnalysis represents analysis of patterns in audit data
type AuditPatternAnalysis struct {
	TimeRange      TimeRange              `json:"time_range"`
	Patterns       []*AuditPattern        `json:"patterns"`
	Anomalies      []*AuditAnomaly        `json:"anomalies,omitempty"`
	Trends         []*AuditTrend          `json:"trends,omitempty"`
	Correlations   []*EventCorrelation    `json:"correlations,omitempty"`
	RiskIndicators []*RiskIndicator       `json:"risk_indicators,omitempty"`
	GeneratedAt    time.Time              `json:"generated_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// AuditAnomaly represents an anomalous activity detected in audit data
type AuditAnomaly struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	DetectedAt  time.Time              `json:"detected_at"`
	TimeRange   TimeRange              `json:"time_range"`
	UserID      string                 `json:"user_id,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Confidence  float64                `json:"confidence"`
	Events      []*AuditEvent          `json:"events,omitempty"`
	Baseline    map[string]interface{} `json:"baseline,omitempty"`
	Deviation   map[string]interface{} `json:"deviation,omitempty"`
	Status      string                 `json:"status"` // "open", "investigating", "resolved", "false_positive"
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditTrend represents a trend in audit data
type AuditTrend struct {
	Metric       string                 `json:"metric"`
	TimeRange    TimeRange              `json:"time_range"`
	Granularity  time.Duration          `json:"granularity"`
	DataPoints   []*TrendDataPoint      `json:"data_points"`
	Direction    string                 `json:"direction"` // "increasing", "decreasing", "stable"
	ChangeRate   float64                `json:"change_rate"`
	Significance float64                `json:"significance"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AuditTrendAnalysis represents analysis of trends in audit data
type AuditTrendAnalysis struct {
	TimeRange   TimeRange              `json:"time_range"`
	Granularity string                 `json:"granularity"`
	Trends      []*AuditTrend          `json:"trends"`
	Summary     map[string]interface{} `json:"summary"`
	GeneratedAt time.Time              `json:"generated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RiskAssessment represents a risk assessment based on audit data
type RiskAssessment struct {
	EntityID        string                 `json:"entity_id"`
	EntityType      string                 `json:"entity_type"`
	RiskScore       float64                `json:"risk_score"`
	RiskLevel       string                 `json:"risk_level"` // "low", "medium", "high", "critical"
	OverallRisk     string                 `json:"overall_risk"`
	Factors         []*RiskFactor          `json:"factors"`
	RiskFactors     []*RiskFactor          `json:"risk_factors"`
	Recommendations []string               `json:"recommendations,omitempty"`
	AssessedAt      time.Time              `json:"assessed_at"`
	ValidUntil      time.Time              `json:"valid_until"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// EventCorrelation represents correlation between related audit events
type EventCorrelation struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Severity     string                 `json:"severity"`
	Confidence   float64                `json:"confidence"`
	Events       []*AuditEvent          `json:"events"`
	Pattern      string                 `json:"pattern,omitempty"`
	TimeWindow   time.Duration          `json:"time_window"`
	DetectedAt   time.Time              `json:"detected_at"`
	CorrelatedAt time.Time              `json:"correlated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BehaviorBaseline represents a behavioral baseline for a user
type BehaviorBaseline struct {
	UserID           string                 `json:"user_id"`
	TimeRange        TimeRange              `json:"time_range"`
	Patterns         map[string]interface{} `json:"patterns"`
	NormalPatterns   map[string]interface{} `json:"normal_patterns"`
	Frequencies      map[string]float64     `json:"frequencies"`
	TimePatterns     map[string]interface{} `json:"time_patterns,omitempty"`
	ResourcePatterns map[string]interface{} `json:"resource_patterns,omitempty"`
	ActionPatterns   map[string]interface{} `json:"action_patterns,omitempty"`
	RiskIndicators   []*RiskIndicator       `json:"risk_indicators,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	EstablishedAt    time.Time              `json:"established_at"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// BaselineComparison represents comparison of current activity to baseline
type BaselineComparison struct {
	UserID          string                 `json:"user_id"`
	CurrentActivity []*AuditEvent          `json:"current_activity"`
	Baseline        *BehaviorBaseline      `json:"baseline"`
	Deviations      []*Deviation           `json:"deviations,omitempty"`
	OverallMatch    string                 `json:"overall_match"`
	RiskScore       float64                `json:"risk_score"`
	RiskLevel       string                 `json:"risk_level"`
	Anomalies       []*AuditAnomaly        `json:"anomalies,omitempty"`
	ComparedAt      time.Time              `json:"compared_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceValidation represents compliance validation results
type ComplianceValidation struct {
	Standard        string                   `json:"standard"`
	Status          string                   `json:"status"`
	ComplianceScore float64                  `json:"compliance_score"`
	TimeRange       TimeRange                `json:"time_range"`
	IsCompliant     bool                     `json:"is_compliant"`
	Requirements    []*ComplianceRequirement `json:"requirements"`
	Violations      []*ComplianceViolation   `json:"violations,omitempty"`
	ValidatedAt     time.Time                `json:"validated_at"`
	ValidatedBy     string                   `json:"validated_by"`
	Metadata        map[string]interface{}   `json:"metadata,omitempty"`
}

// RequirementStatus represents the status of a compliance requirement
type RequirementStatus struct {
	RequirementID   string                 `json:"requirement_id"`
	Description     string                 `json:"description"`
	Status          string                 `json:"status"` // "compliant", "non-compliant", "partial", "pending"
	ViolationCount  int                    `json:"violation_count"`
	LastChecked     time.Time              `json:"last_checked"`
	Evidence        []*ComplianceEvidence  `json:"evidence,omitempty"`
	Violations      []*ComplianceViolation `json:"violations,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceStatus represents current compliance status
type ComplianceStatus struct {
	Standard        string                 `json:"standard"`
	Status          string                 `json:"status"` // "compliant", "non-compliant", "partial"
	OverallStatus   string                 `json:"overall_status"`
	ComplianceScore float64                `json:"compliance_score"`
	LastChecked     time.Time              `json:"last_checked"`
	LastValidated   time.Time              `json:"last_validated"`
	NextCheck       time.Time              `json:"next_check"`
	Requirements    []*RequirementStatus   `json:"requirements"`
	Compliant       int                    `json:"compliant"`
	NonCompliant    int                    `json:"non_compliant"`
	Violations      []*ComplianceViolation `json:"violations,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceEvidence represents evidence of compliance
type ComplianceEvidence struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Standard      string                 `json:"standard"`
	Requirement   string                 `json:"requirement"`
	Description   string                 `json:"description"`
	Source        string                 `json:"source"`
	TimeRange     TimeRange              `json:"time_range"`
	EventCount    int64                  `json:"event_count"`
	CollectedAt   time.Time              `json:"collected_at"`
	GeneratedAt   time.Time              `json:"generated_at"`
	GeneratedBy   string                 `json:"generated_by"`
	IntegrityHash string                 `json:"integrity_hash"`
	Events        []*AuditEvent          `json:"events,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID            string                 `json:"id"`
	Standard      string                 `json:"standard"`
	RequirementID string                 `json:"requirement_id"`
	Requirement   string                 `json:"requirement"`
	Severity      string                 `json:"severity"`
	Description   string                 `json:"description"`
	DetectedAt    time.Time              `json:"detected_at"`
	Event         *AuditEvent            `json:"event,omitempty"`
	Events        []*AuditEvent          `json:"events,omitempty"`
	Status        string                 `json:"status"` // "open", "mitigated", "accepted"
	Mitigation    string                 `json:"mitigation,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// RetentionStatus represents data retention status
type RetentionStatus struct {
	TotalEvents           int64                  `json:"total_events"`
	TotalRecords          int64                  `json:"total_records"`
	RetainedRecords       int64                  `json:"retained_records"`
	ExpiredRecords        int64                  `json:"expired_records"`
	EventsOlderThan1Year  int64                  `json:"events_older_than_1_year"`
	EventsOlderThan7Years int64                  `json:"events_older_than_7_years"`
	Policies              []*RetentionPolicy     `json:"policies"`
	RetentionPolicies     []*RetentionPolicy     `json:"retention_policies"`
	Violations            []*RetentionViolation  `json:"violations,omitempty"`
	AssessedAt            time.Time              `json:"assessed_at"`
	LastCleanup           *time.Time             `json:"last_cleanup,omitempty"`
	NextCleanup           *time.Time             `json:"next_cleanup,omitempty"`
	Status                string                 `json:"status"` // "healthy", "warning", "critical"
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// RetentionViolation represents a data retention policy violation
type RetentionViolation struct {
	ID             string                 `json:"id"`
	PolicyID       string                 `json:"policy_id"`
	Policy         string                 `json:"policy"`
	Type           string                 `json:"type"` // "over_retention", "under_retention", "missing_data"
	Description    string                 `json:"description"`
	Severity       string                 `json:"severity"`
	EventCount     int64                  `json:"event_count"`
	RequiredAction string                 `json:"required_action"`
	DetectedAt     time.Time              `json:"detected_at"`
	Status         string                 `json:"status"` // "open", "resolved", "ignored"
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// RetentionPolicy represents a data retention policy
type RetentionPolicy struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	Description           string                 `json:"description,omitempty"`
	EntityType            string                 `json:"entity_type"`
	RetentionPeriod       time.Duration          `json:"retention_period"`
	MaxAge                time.Duration          `json:"max_age"`
	ArchiveAfter          time.Duration          `json:"archive_after,omitempty"`
	ArchiveBeforeDelete   time.Duration          `json:"archive_before_delete,omitempty"`
	DeleteAfter           time.Duration          `json:"delete_after,omitempty"`
	Enabled               bool                   `json:"enabled"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// RetentionReport represents a data retention compliance report
type RetentionReport struct {
	ID               string                 `json:"id"`
	GeneratedAt      time.Time              `json:"generated_at"`
	GeneratedBy      string                 `json:"generated_by"`
	Policies         []*RetentionPolicy     `json:"policies"`
	Status           *RetentionStatus       `json:"status"`
	RetentionStatus  *RetentionStatus       `json:"retention_status"`
	ComplianceStatus *ComplianceStatus      `json:"compliance_status"`
	Compliance       map[string]interface{} `json:"compliance"`
	Recommendations  []string               `json:"recommendations,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}


// AuditAlert represents an audit-based alert configuration
type AuditAlert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Severity    string                 `json:"severity"`
	Condition   string                 `json:"condition,omitempty"`
	Conditions  []*AlertCondition      `json:"conditions"`
	Actions     []*AlertAction         `json:"actions"`
	Recipients  []string               `json:"recipients,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertNotification represents an alert notification
type AlertNotification struct {
	ID          string                 `json:"id"`
	AlertID     string                 `json:"alert_id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	SentAt      time.Time              `json:"sent_at"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
	Status      string                 `json:"status"` // "sent", "delivered", "failed"
	Recipients  []string               `json:"recipients"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ChainVerification represents hash chain verification results
type ChainVerification struct {
	StartHash      string                 `json:"start_hash"`
	EndHash        string                 `json:"end_hash"`
	IsValid        bool                   `json:"is_valid"`
	TotalEvents    int64                  `json:"total_events"`
	VerifiedEvents int64                  `json:"verified_events"`
	BrokenLinks    []*BrokenLink          `json:"broken_links,omitempty"`
	VerifiedAt     time.Time              `json:"verified_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ChainInfo represents information about a hash chain
type ChainInfo struct {
	ChainID      string                 `json:"chain_id"`
	StartHash    string                 `json:"start_hash"`
	EndHash      string                 `json:"end_hash"`
	CurrentHash  string                 `json:"current_hash"`
	TotalEvents  int64                  `json:"total_events"`
	FirstEvent   *AuditEvent            `json:"first_event,omitempty"`
	LastEvent    *AuditEvent            `json:"last_event,omitempty"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	GeneratedAt  time.Time              `json:"generated_at"`
	IsValid      bool                   `json:"is_valid"`
	LastUpdated  time.Time              `json:"last_updated"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ChainExport represents an exported hash chain
type ChainExport struct {
	ChainInfo    *ChainInfo             `json:"chain_info"`
	TimeRange    TimeRange              `json:"time_range"`
	Events       []*AuditEvent          `json:"events"`
	TotalEvents  int64                  `json:"total_events"`
	FirstHash    string                 `json:"first_hash"`
	LastHash     string                 `json:"last_hash"`
	ExportedAt   time.Time              `json:"exported_at"`
	ExportedBy   string                 `json:"exported_by"`
	Signature    string                 `json:"signature"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}


// EventValidation represents validation of an event against the hash chain
type EventValidation struct {
	EventID      string                 `json:"event_id"`
	EventHash    string                 `json:"event_hash"`
	IsValid      bool                   `json:"is_valid"`
	Position     int64                  `json:"position"`
	Hash         string                 `json:"hash"`
	PreviousHash string                 `json:"previous_hash"`
	NextHash     string                 `json:"next_hash"`
	ValidatedAt  time.Time              `json:"validated_at"`
	Errors       []string               `json:"errors,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type ReportChart struct {
	Type    string                 `json:"type"`
	Title   string                 `json:"title"`
	Data    map[string]interface{} `json:"data"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type UserActivity struct {
	UserID       string    `json:"user_id"`
	UserEmail    string    `json:"user_email,omitempty"`
	EventCount   int64     `json:"event_count"`
	LastActivity time.Time `json:"last_activity"`
}

type ResourceActivity struct {
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resource_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	EventCount   int64     `json:"event_count"`
	LastActivity time.Time `json:"last_activity"`
}

type ActionActivity struct {
	Action       string    `json:"action"`
	EventCount   int64     `json:"event_count"`
	LastActivity time.Time `json:"last_activity"`
}

type RiskMetrics struct {
	OverallRisk      float64 `json:"overall_risk"`
	TotalRiskScore   float64 `json:"total_risk_score"`
	HighRiskEvents   int64   `json:"high_risk_events"`
	MediumRiskEvents int64   `json:"medium_risk_events"`
	LowRiskEvents    int64   `json:"low_risk_events"`
}

type AuditPattern struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Confidence  float64                `json:"confidence"`
	Users       []string               `json:"users,omitempty"`
	Actions     []string               `json:"actions,omitempty"`
	Resources   []string               `json:"resources,omitempty"`
	Events      []*AuditEvent          `json:"events"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type RiskFactor struct {
	Name        string  `json:"name"`
	Factor      string  `json:"factor"`
	Weight      float64 `json:"weight"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

type RiskIndicator struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Value       float64 `json:"value"`
	Score       float64 `json:"score"`
	Threshold   float64 `json:"threshold"`
	Description string  `json:"description"`
}

type BehaviorDeviation struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Confidence  float64                `json:"confidence"`
	Details     map[string]interface{} `json:"details"`
}

type ComplianceRequirement struct {
	ID          string `json:"id"`
	Standard    string `json:"standard"`
	Category    string `json:"category"`
	Requirement string `json:"requirement"`
	Description string `json:"description"`
	IsCompliant bool   `json:"is_compliant"`
	Evidence    string `json:"evidence,omitempty"`
}

type AlertAction struct {
	Type       string                 `json:"type"`
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type BrokenLink struct {
	Position     int64  `json:"position"`
	ExpectedHash string `json:"expected_hash"`
	ActualHash   string `json:"actual_hash"`
	EventID      string `json:"event_id"`
}

// Missing types that were referenced in the error messages

// Deviation represents a behavioral deviation
type Deviation struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Confidence  float64                `json:"confidence"`
	UserID      string                 `json:"user_id,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Event       *AuditEvent            `json:"event,omitempty"`
	DetectedAt  time.Time              `json:"detected_at"`
	Baseline    map[string]interface{} `json:"baseline,omitempty"`
	Current     map[string]interface{} `json:"current,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}


// HashChain represents a hash chain structure
type HashChain struct {
	ID          string                 `json:"id"`
	StartHash   string                 `json:"start_hash"`
	EndHash     string                 `json:"end_hash"`
	Events      []*AuditEvent          `json:"events"`
	EventCount  int64                  `json:"event_count"`
	FirstHash   string                 `json:"first_hash"`
	LastHash    string                 `json:"last_hash"`
	IsValid     bool                   `json:"is_valid"`
	Length      int                    `json:"length"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditSystemHealth represents the health status of the audit system
type AuditSystemHealth struct {
	Status           string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	OverallStatus    string                 `json:"overall_status"`
	CheckedAt        time.Time              `json:"checked_at"`
	Services         []*ServiceHealth       `json:"services"`
	TotalEvents      int64                  `json:"total_events"`
	EventsToday      int64                  `json:"events_today"`
	EventsLastHour   int64                  `json:"events_last_hour"`
	ChainIntegrity   bool                   `json:"chain_integrity"`
	StorageHealth    string                 `json:"storage_health"`
	ReplicationLag   time.Duration          `json:"replication_lag,omitempty"`
	PendingEvents    int64                  `json:"pending_events"`
	FailedEvents     int64                  `json:"failed_events"`
	LastHealthCheck  time.Time              `json:"last_health_check"`
	Issues           []string               `json:"issues,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ServiceHealth represents health status of an individual service
type ServiceHealth struct {
	Name     string    `json:"name"`
	Status   string    `json:"status"` // "healthy", "degraded", "unhealthy"
	LastSeen time.Time `json:"last_seen"`
	Version  string    `json:"version,omitempty"`
	Error    string    `json:"error,omitempty"`
	Errors   []string  `json:"errors,omitempty"`
}
