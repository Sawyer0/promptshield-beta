package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionPlan represents a billing plan with pricing and limits
type SubscriptionPlan struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	DisplayName  string    `json:"display_name" db:"display_name"`
	Description  string    `json:"description" db:"description"`
	PriceMonthly int       `json:"price_monthly" db:"price_monthly"`         // in cents
	PriceYearly  *int      `json:"price_yearly,omitempty" db:"price_yearly"` // in cents
	Features     Features  `json:"features" db:"features"`
	Limits       Limits    `json:"limits" db:"limits"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Features defines what's included in a plan
type Features struct {
	UnlimitedRulepacks  bool     `json:"unlimited_rulepacks"`
	ComplianceReporting []string `json:"compliance_reporting"` // ["SOC2", "GDPR", "HIPAA", "NIST"]
	SupportLevel        string   `json:"support_level"`        // "email", "phone", "24/7"
	SLA                 string   `json:"sla"`                  // "99.5%", "99.9%"
	CustomIntegrations  bool     `json:"custom_integrations"`
	AdvancedAnalytics   bool     `json:"advanced_analytics"`
	AirGappedDeployment bool     `json:"air_gapped_deployment"`
	PriorityProcessing  bool     `json:"priority_processing"`
	WhiteLabeling       bool     `json:"white_labeling"`
	SSO                 bool     `json:"sso"`
	AuditLogs           bool     `json:"audit_logs"`
	CustomRetention     bool     `json:"custom_retention"`
}

// Limits defines usage limits for a plan
type Limits struct {
	APICallsMonthly   int `json:"api_calls_monthly"`   // -1 for unlimited
	LLMCallsMonthly   int `json:"llm_calls_monthly"`   // -1 for unlimited
	Rulepacks         int `json:"rulepacks"`           // -1 for unlimited
	Users             int `json:"users"`               // -1 for unlimited
	DataRetentionDays int `json:"data_retention_days"` // -1 for unlimited
}

// Subscription represents a tenant's active subscription
type Subscription struct {
	ID                   uuid.UUID          `json:"id" db:"id"`
	TenantID             uuid.UUID          `json:"tenant_id" db:"tenant_id"`
	PlanID               uuid.UUID          `json:"plan_id" db:"plan_id"`
	Status               SubscriptionStatus `json:"status" db:"status"`
	BillingCycle         BillingCycle       `json:"billing_cycle" db:"billing_cycle"`
	CurrentPeriodStart   time.Time          `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end" db:"current_period_end"`
	TrialStart           *time.Time         `json:"trial_start,omitempty" db:"trial_start"`
	TrialEnd             *time.Time         `json:"trial_end,omitempty" db:"trial_end"`
	StripeCustomerID     *string            `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	StripeSubscriptionID *string            `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	StripePriceID        *string            `json:"stripe_price_id,omitempty" db:"stripe_price_id"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end" db:"cancel_at_period_end"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty" db:"canceled_at"`
	CreatedAt            time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at" db:"updated_at"`
}

// LLMUsage tracks daily LLM usage for billing
type LLMUsage struct {
	ID         uuid.UUID `json:"id" db:"id"`
	TenantID   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	UsageDate  time.Time `json:"usage_date" db:"usage_date"`
	LLMCalls   int       `json:"llm_calls" db:"llm_calls"`
	APICalls   int       `json:"api_calls" db:"api_calls"`
	Violations int       `json:"violations" db:"violations"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// UsageBilling represents a billing period with usage and charges
type UsageBilling struct {
	ID                 uuid.UUID      `json:"id" db:"id"`
	TenantID           uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	SubscriptionID     uuid.UUID      `json:"subscription_id" db:"subscription_id"`
	BillingPeriodStart time.Time      `json:"billing_period_start" db:"billing_period_start"`
	BillingPeriodEnd   time.Time      `json:"billing_period_end" db:"billing_period_end"`
	BaseAmount         int            `json:"base_amount" db:"base_amount"`   // in cents
	UsageAmount        int            `json:"usage_amount" db:"usage_amount"` // in cents
	TotalAmount        int            `json:"total_amount" db:"total_amount"` // in cents
	Status             BillingStatus  `json:"status" db:"status"`
	StripeInvoiceID    *string        `json:"stripe_invoice_id,omitempty" db:"stripe_invoice_id"`
	UsageBreakdown     UsageBreakdown `json:"usage_breakdown" db:"usage_breakdown"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at"`
}

// UsageBreakdown provides detailed usage metrics
type UsageBreakdown struct {
	APICallsTotal       int `json:"api_calls_total"`
	APICallsIncluded    int `json:"api_calls_included"`
	APICallsOverage     int `json:"api_calls_overage"`
	APICallsOverageCost int `json:"api_calls_overage_cost"` // in cents
	LLMCallsTotal       int `json:"llm_calls_total"`
	LLMCallsIncluded    int `json:"llm_calls_included"`
	LLMCallsOverage     int `json:"llm_calls_overage"`
	LLMCallsOverageCost int `json:"llm_calls_overage_cost"` // in cents
	ViolationsTotal     int `json:"violations_total"`
	ViolationsCost      int `json:"violations_cost"` // in cents
}

// Enums
type SubscriptionStatus string

const (
	SubscriptionStatusActive     SubscriptionStatus = "active"
	SubscriptionStatusTrial      SubscriptionStatus = "trial"
	SubscriptionStatusPastDue    SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled   SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid     SubscriptionStatus = "unpaid"
	SubscriptionStatusIncomplete SubscriptionStatus = "incomplete"
)

type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

type BillingStatus string

const (
	BillingStatusDraft         BillingStatus = "draft"
	BillingStatusOpen          BillingStatus = "open"
	BillingStatusPaid          BillingStatus = "paid"
	BillingStatusVoid          BillingStatus = "void"
	BillingStatusUncollectible BillingStatus = "uncollectible"
)

// Pricing constants (in cents)
const (
	// LLM usage pricing
	LLMCallPriceCents = 2 // $0.02 per LLM call

	// API call pricing (for overage)
	APICallPriceCents = 1 // $0.01 per 1000 API calls

	// Violation pricing (for compliance reporting)
	ViolationPriceCents = 5 // $0.05 per violation
)

// Plan constants
const (
	PlanProfessional   = "professional"
	PlanEnterprise     = "enterprise"
	PlanEnterprisePlus = "enterprise_plus"
)
