package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// BillingService handles subscription management, usage tracking, and billing
type BillingService interface {
	// Subscription Management
	CreateSubscription(ctx context.Context, tenantID uuid.UUID, planID uuid.UUID, billingCycle domain.BillingCycle) (*domain.Subscription, error)
	GetSubscription(ctx context.Context, tenantID uuid.UUID) (*domain.Subscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID uuid.UUID, updates SubscriptionUpdates) (*domain.Subscription, error)
	CancelSubscription(ctx context.Context, subscriptionID uuid.UUID, cancelAtPeriodEnd bool) error

	// Plan Management
	GetPlans(ctx context.Context) ([]*domain.SubscriptionPlan, error)
	GetPlan(ctx context.Context, planID uuid.UUID) (*domain.SubscriptionPlan, error)

	// Usage Tracking
	RecordUsage(ctx context.Context, tenantID uuid.UUID, usage UsageRecord) error
	GetUsage(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*domain.LLMUsage, error)
	GetUsageForBilling(ctx context.Context, tenantID uuid.UUID, billingPeriodStart, billingPeriodEnd time.Time) (*domain.UsageBreakdown, error)

	// Billing
	ProcessBilling(ctx context.Context, tenantID uuid.UUID, billingPeriodStart, billingPeriodEnd time.Time) (*domain.UsageBilling, error)
	GetBillingHistory(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageBilling, error)

	// Stripe Integration
	CreateStripeCustomer(ctx context.Context, tenantID uuid.UUID, email string) (string, error)
	CreateStripeSubscription(ctx context.Context, customerID string, priceID string) (string, error)
	HandleStripeWebhook(ctx context.Context, eventType string, eventData []byte) error

	// Quota Enforcement
	CheckQuota(ctx context.Context, tenantID uuid.UUID, resourceType QuotaResource) (*QuotaStatus, error)
	EnforceQuota(ctx context.Context, tenantID uuid.UUID, resourceType QuotaResource) error
}

// SubscriptionUpdates represents updates to a subscription
type SubscriptionUpdates struct {
	PlanID            *uuid.UUID
	BillingCycle      *domain.BillingCycle
	CancelAtPeriodEnd *bool
}

// UsageRecord represents a single usage event
type UsageRecord struct {
	APICalls   int
	LLMCalls   int
	Violations int
	Timestamp  time.Time
}

// QuotaResource represents the type of resource being checked
type QuotaResource string

const (
	QuotaResourceAPICalls  QuotaResource = "api_calls"
	QuotaResourceLLMCalls  QuotaResource = "llm_calls"
	QuotaResourceRulepacks QuotaResource = "rulepacks"
	QuotaResourceUsers     QuotaResource = "users"
)

// QuotaStatus represents the current quota status
type QuotaStatus struct {
	ResourceType   QuotaResource
	Used           int
	Limit          int
	Remaining      int
	IsUnlimited    bool
	ResetDate      *time.Time
	OverageAllowed bool
}

// BillingServiceConfig holds configuration for the billing service
type BillingServiceConfig struct {
	StripeSecretKey     string
	StripeWebhookSecret string
	DefaultTrialDays    int
	OverageGracePeriod  time.Duration
	BillingDayOfMonth   int // 1-31, day of month to bill
}

// NewBillingService creates a new billing service instance
func NewBillingService(
	config BillingServiceConfig,
	subscriptionRepo SubscriptionRepository,
	usageRepo UsageRepository,
	billingRepo BillingRepository,
	planRepo PlanRepository,
) BillingService {
	return &billingService{
		config:           config,
		subscriptionRepo: subscriptionRepo,
		usageRepo:        usageRepo,
		billingRepo:      billingRepo,
		planRepo:         planRepo,
	}
}

// Repository interfaces
type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *domain.Subscription) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.Subscription, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	Update(ctx context.Context, subscription *domain.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type UsageRepository interface {
	RecordUsage(ctx context.Context, usage *domain.LLMUsage) error
	GetUsage(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*domain.LLMUsage, error)
	GetUsageForBilling(ctx context.Context, tenantID uuid.UUID, billingPeriodStart, billingPeriodEnd time.Time) (*domain.UsageBreakdown, error)
	GetCurrentUsage(ctx context.Context, tenantID uuid.UUID) (*domain.UsageBreakdown, error)
}

type BillingRepository interface {
	CreateBilling(ctx context.Context, billing *domain.UsageBilling) error
	GetBillingHistory(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageBilling, error)
	GetBillingByID(ctx context.Context, id uuid.UUID) (*domain.UsageBilling, error)
	UpdateBilling(ctx context.Context, billing *domain.UsageBilling) error
}

type PlanRepository interface {
	GetAll(ctx context.Context) ([]*domain.SubscriptionPlan, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error)
	GetByName(ctx context.Context, name string) (*domain.SubscriptionPlan, error)
}
