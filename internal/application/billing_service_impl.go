package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/subscription"
	"github.com/stripe/stripe-go/v78/webhook"
)

type billingService struct {
	config           BillingServiceConfig
	subscriptionRepo SubscriptionRepository
	usageRepo        UsageRepository
	billingRepo      BillingRepository
	planRepo         PlanRepository
}

// CreateSubscription creates a new subscription for a tenant
func (s *billingService) CreateSubscription(ctx context.Context, tenantID uuid.UUID, planID uuid.UUID, billingCycle domain.BillingCycle) (*domain.Subscription, error) {
	_, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	now := time.Now()
	trialEnd := now.AddDate(0, 0, s.config.DefaultTrialDays)

	sub := &domain.Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		PlanID:             planID,
		Status:             domain.SubscriptionStatusTrial,
		BillingCycle:       billingCycle,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   trialEnd,
		TrialStart:         &now,
		TrialEnd:           &trialEnd,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.subscriptionRepo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	slog.Info("Created subscription", "tenant_id", tenantID, "plan_id", planID, "subscription_id", sub.ID)
	return sub, nil
}

// GetSubscription retrieves a subscription by ID
func (s *billingService) GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	return s.subscriptionRepo.GetByID(ctx, subscriptionID)
}

// GetSubscriptionByTenant retrieves a tenant's subscription
func (s *billingService) GetSubscriptionByTenant(ctx context.Context, tenantID uuid.UUID) (*domain.Subscription, error) {
	return s.subscriptionRepo.GetByTenantID(ctx, tenantID)
}

// UpdateSubscription updates a subscription
func (s *billingService) UpdateSubscription(ctx context.Context, subscriptionID uuid.UUID, updates SubscriptionUpdates) (*domain.Subscription, error) {
	sub, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if updates.PlanID != nil {
		sub.PlanID = *updates.PlanID
	}
	if updates.BillingCycle != nil {
		sub.BillingCycle = *updates.BillingCycle
	}
	if updates.CancelAtPeriodEnd != nil {
		sub.CancelAtPeriodEnd = *updates.CancelAtPeriodEnd
	}

	sub.UpdatedAt = time.Now()

	if err := s.subscriptionRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	return sub, nil
}

// CancelSubscription cancels a subscription
func (s *billingService) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID, cancelAtPeriodEnd bool) error {
	sub, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if cancelAtPeriodEnd {
		sub.CancelAtPeriodEnd = true
	} else {
		now := time.Now()
		sub.Status = domain.SubscriptionStatusCanceled
		sub.CanceledAt = &now
	}

	sub.UpdatedAt = time.Now()

	if err := s.subscriptionRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	// Cancel in Stripe if we have a Stripe subscription ID
	if sub.StripeSubscriptionID != nil {
		_, err := subscription.Cancel(*sub.StripeSubscriptionID, &stripe.SubscriptionCancelParams{})
		if err != nil {
			slog.Error("Failed to cancel Stripe subscription", "stripe_subscription_id", *sub.StripeSubscriptionID, "error", err)
		}
	}

	return nil
}

// GetPlans retrieves all available subscription plans
func (s *billingService) GetPlans(ctx context.Context) ([]*domain.SubscriptionPlan, error) {
	return s.planRepo.GetAll(ctx)
}

// GetPlan retrieves a specific subscription plan
func (s *billingService) GetPlan(ctx context.Context, planID uuid.UUID) (*domain.SubscriptionPlan, error) {
	return s.planRepo.GetByID(ctx, planID)
}

// RecordUsage records usage for billing
func (s *billingService) RecordUsage(ctx context.Context, tenantID uuid.UUID, usage UsageRecord) error {
	// Get or create usage record for today
	today := time.Now().Truncate(24 * time.Hour)
	existingUsage, err := s.usageRepo.GetUsage(ctx, tenantID, today, today.Add(24*time.Hour))
	if err != nil {
		// Create new usage record
		usageRecord := &domain.LLMUsage{
			ID:         uuid.New(),
			TenantID:   tenantID,
			UsageDate:  today,
			LLMCalls:   usage.LLMCalls,
			APICalls:   usage.APICalls,
			Violations: usage.Violations,
			CreatedAt:  time.Now(),
		}
		return s.usageRepo.RecordUsage(ctx, usageRecord)
	}

	// Update existing usage
	existingUsage.LLMCalls += usage.LLMCalls
	existingUsage.APICalls += usage.APICalls
	existingUsage.Violations += usage.Violations

	return s.usageRepo.RecordUsage(ctx, existingUsage)
}

// GetUsage retrieves usage for a date range
func (s *billingService) GetUsage(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*domain.LLMUsage, error) {
	return s.usageRepo.GetUsage(ctx, tenantID, startDate, endDate)
}

// GetUsageForBilling retrieves usage breakdown for billing
func (s *billingService) GetUsageForBilling(ctx context.Context, tenantID uuid.UUID, billingPeriodStart, billingPeriodEnd time.Time) (*domain.UsageBreakdown, error) {
	return s.usageRepo.GetUsageForBilling(ctx, tenantID, billingPeriodStart, billingPeriodEnd)
}

// ProcessBilling processes billing for a tenant
func (s *billingService) ProcessBilling(ctx context.Context, tenantID uuid.UUID, billingPeriodStart, billingPeriodEnd time.Time) (*domain.UsageBilling, error) {
	// Get subscription
	sub, err := s.GetSubscription(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get plan
	plan, err := s.GetPlan(ctx, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	// Get usage breakdown
	usageBreakdown, err := s.GetUsageForBilling(ctx, tenantID, billingPeriodStart, billingPeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage breakdown: %w", err)
	}

	// Calculate costs
	baseAmount := plan.PriceMonthly
	if sub.BillingCycle == domain.BillingCycleYearly && plan.PriceYearly != nil {
		baseAmount = *plan.PriceYearly
	}

	// Calculate overage costs
	usageAmount := 0
	if usageBreakdown.LLMCallsOverage > 0 {
		usageAmount += usageBreakdown.LLMCallsOverageCost
	}
	if usageBreakdown.APICallsOverage > 0 {
		usageAmount += usageBreakdown.APICallsOverageCost
	}
	if usageBreakdown.ViolationsCost > 0 {
		usageAmount += usageBreakdown.ViolationsCost
	}

	totalAmount := baseAmount + usageAmount

	// Create billing record
	billing := &domain.UsageBilling{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		SubscriptionID:     sub.ID,
		BillingPeriodStart: billingPeriodStart,
		BillingPeriodEnd:   billingPeriodEnd,
		BaseAmount:         baseAmount,
		UsageAmount:        usageAmount,
		TotalAmount:        totalAmount,
		Status:             domain.BillingStatusDraft,
		UsageBreakdown:     *usageBreakdown,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.billingRepo.CreateBilling(ctx, billing); err != nil {
		return nil, fmt.Errorf("failed to create billing record: %w", err)
	}

	return billing, nil
}

// GetBillingHistory retrieves billing history for a tenant
func (s *billingService) GetBillingHistory(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageBilling, error) {
	return s.billingRepo.GetBillingHistory(ctx, tenantID)
}

// CreateStripeCustomer creates a Stripe customer
func (s *billingService) CreateStripeCustomer(ctx context.Context, tenantID uuid.UUID, email string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Metadata: map[string]string{
			"tenant_id": tenantID.String(),
		},
	}

	customer, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return customer.ID, nil
}

// CreateStripeSubscription creates a Stripe subscription
func (s *billingService) CreateStripeSubscription(ctx context.Context, customerID string, priceID string) (string, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
		PaymentBehavior: stripe.String("default_incomplete"),
		PaymentSettings: &stripe.SubscriptionPaymentSettingsParams{
			SaveDefaultPaymentMethod: stripe.String("on_subscription"),
		},
		Expand: []*string{stripe.String("latest_invoice.payment_intent")},
	}

	sub, err := subscription.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	return sub.ID, nil
}

// HandleStripeWebhook handles Stripe webhook events
func (s *billingService) HandleStripeWebhook(ctx context.Context, eventType string, eventData []byte) error {
	event, err := webhook.ConstructEvent(eventData, "", s.config.StripeWebhookSecret)
	if err != nil {
		return fmt.Errorf("failed to construct webhook event: %w", err)
	}

	switch event.Type {
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(ctx, event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event)
	case "invoice.payment_succeeded":
		return s.handleInvoicePaymentSucceeded(ctx, event)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(ctx, event)
	default:
		slog.Info("Unhandled Stripe webhook event", "event_type", event.Type)
		return nil
	}
}

// CheckQuota checks if a tenant has remaining quota
func (s *billingService) CheckQuota(ctx context.Context, tenantID uuid.UUID, resourceType QuotaResource) (*QuotaStatus, error) {
	sub, err := s.GetSubscription(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	plan, err := s.GetPlan(ctx, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	// Get current usage
	currentUsage, err := s.usageRepo.GetCurrentUsage(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}

	var used, limit int
	var isUnlimited bool

	switch resourceType {
	case QuotaResourceAPICalls:
		used = currentUsage.APICallsTotal
		limit = plan.Limits.APICallsMonthly
		isUnlimited = limit == -1
	case QuotaResourceLLMCalls:
		used = currentUsage.LLMCallsTotal
		limit = plan.Limits.LLMCallsMonthly
		isUnlimited = limit == -1
	case QuotaResourceRulepacks:
		// This would need to be tracked separately
		used = 0 // TODO: implement rulepack counting
		limit = plan.Limits.Rulepacks
		isUnlimited = limit == -1
	case QuotaResourceUsers:
		// This would need to be tracked separately
		used = 0 // TODO: implement user counting
		limit = plan.Limits.Users
		isUnlimited = limit == -1
	}

	remaining := limit - used
	if isUnlimited {
		remaining = -1
	}

	// Calculate reset date (end of current billing period)
	resetDate := sub.CurrentPeriodEnd

	return &QuotaStatus{
		ResourceType:   resourceType,
		Used:           used,
		Limit:          limit,
		Remaining:      remaining,
		IsUnlimited:    isUnlimited,
		ResetDate:      &resetDate,
		OverageAllowed: true, // Allow overage with billing
	}, nil
}

// EnforceQuota enforces quota limits
func (s *billingService) EnforceQuota(ctx context.Context, tenantID uuid.UUID, resourceType QuotaResource) error {
	quotaStatus, err := s.CheckQuota(ctx, tenantID, resourceType)
	if err != nil {
		return fmt.Errorf("failed to check quota: %w", err)
	}

	// Allow if unlimited or has remaining quota
	if quotaStatus.IsUnlimited || quotaStatus.Remaining > 0 {
		return nil
	}

	// Allow overage for now (will be billed)
	if quotaStatus.OverageAllowed {
		return nil
	}

	return fmt.Errorf("quota exceeded for resource %s", resourceType)
}

// Webhook handlers
func (s *billingService) handleSubscriptionCreated(ctx context.Context, event stripe.Event) error {
	// Implementation for subscription created
	return nil
}

func (s *billingService) handleSubscriptionUpdated(ctx context.Context, event stripe.Event) error {
	// Implementation for subscription updated
	return nil
}

func (s *billingService) handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	// Implementation for subscription deleted
	return nil
}

func (s *billingService) handleInvoicePaymentSucceeded(ctx context.Context, event stripe.Event) error {
	// Implementation for successful payment
	return nil
}

func (s *billingService) handleInvoicePaymentFailed(ctx context.Context, event stripe.Event) error {
	// Implementation for failed payment
	return nil
}
