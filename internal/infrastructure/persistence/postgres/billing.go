package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/promptshield/promptshield/internal/domain"
)

// BillingRepository implements the billing repository interface
type BillingRepository struct {
	db *pgxpool.Pool
}

// NewBillingRepository creates a new billing repository
func NewBillingRepository(db *pgxpool.Pool) *BillingRepository {
	return &BillingRepository{db: db}
}

// SubscriptionRepository implementation
type SubscriptionRepository struct {
	db *pgxpool.Pool
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Create creates a new subscription
func (r *SubscriptionRepository) Create(ctx context.Context, subscription *domain.Subscription) error {
	query := `
		INSERT INTO subscriptions (
			id, tenant_id, plan_id, status, billing_cycle,
			current_period_start, current_period_end, trial_start, trial_end,
			stripe_customer_id, stripe_subscription_id, stripe_price_id,
			cancel_at_period_end, canceled_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err := r.db.Exec(ctx, query,
		subscription.ID,
		subscription.TenantID,
		subscription.PlanID,
		subscription.Status,
		subscription.BillingCycle,
		subscription.CurrentPeriodStart,
		subscription.CurrentPeriodEnd,
		subscription.TrialStart,
		subscription.TrialEnd,
		subscription.StripeCustomerID,
		subscription.StripeSubscriptionID,
		subscription.StripePriceID,
		subscription.CancelAtPeriodEnd,
		subscription.CanceledAt,
		subscription.CreatedAt,
		subscription.UpdatedAt,
	)

	return err
}

// GetByTenantID retrieves a subscription by tenant ID
func (r *SubscriptionRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT id, tenant_id, plan_id, status, billing_cycle,
		       current_period_start, current_period_end, trial_start, trial_end,
		       stripe_customer_id, stripe_subscription_id, stripe_price_id,
		       cancel_at_period_end, canceled_at, created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	var sub domain.Subscription
	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&sub.ID,
		&sub.TenantID,
		&sub.PlanID,
		&sub.Status,
		&sub.BillingCycle,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.TrialStart,
		&sub.TrialEnd,
		&sub.StripeCustomerID,
		&sub.StripeSubscriptionID,
		&sub.StripePriceID,
		&sub.CancelAtPeriodEnd,
		&sub.CanceledAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, err
	}

	return &sub, nil
}

// GetByID retrieves a subscription by ID
func (r *SubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT id, tenant_id, plan_id, status, billing_cycle,
		       current_period_start, current_period_end, trial_start, trial_end,
		       stripe_customer_id, stripe_subscription_id, stripe_price_id,
		       cancel_at_period_end, canceled_at, created_at, updated_at
		FROM subscriptions
		WHERE id = $1`

	var sub domain.Subscription
	err := r.db.QueryRow(ctx, query, id).Scan(
		&sub.ID,
		&sub.TenantID,
		&sub.PlanID,
		&sub.Status,
		&sub.BillingCycle,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.TrialStart,
		&sub.TrialEnd,
		&sub.StripeCustomerID,
		&sub.StripeSubscriptionID,
		&sub.StripePriceID,
		&sub.CancelAtPeriodEnd,
		&sub.CanceledAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, err
	}

	return &sub, nil
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(ctx context.Context, subscription *domain.Subscription) error {
	query := `
		UPDATE subscriptions SET
			plan_id = $2,
			status = $3,
			billing_cycle = $4,
			current_period_start = $5,
			current_period_end = $6,
			trial_start = $7,
			trial_end = $8,
			stripe_customer_id = $9,
			stripe_subscription_id = $10,
			stripe_price_id = $11,
			cancel_at_period_end = $12,
			canceled_at = $13,
			updated_at = $14
		WHERE id = $1`

	_, err := r.db.Exec(ctx, query,
		subscription.ID,
		subscription.PlanID,
		subscription.Status,
		subscription.BillingCycle,
		subscription.CurrentPeriodStart,
		subscription.CurrentPeriodEnd,
		subscription.TrialStart,
		subscription.TrialEnd,
		subscription.StripeCustomerID,
		subscription.StripeSubscriptionID,
		subscription.StripePriceID,
		subscription.CancelAtPeriodEnd,
		subscription.CanceledAt,
		subscription.UpdatedAt,
	)

	return err
}

// Delete deletes a subscription
func (r *SubscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM subscriptions WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// UsageRepository implementation
type UsageRepository struct {
	db *pgxpool.Pool
}

// NewUsageRepository creates a new usage repository
func NewUsageRepository(db *pgxpool.Pool) *UsageRepository {
	return &UsageRepository{db: db}
}

// RecordUsage records usage for a tenant
func (r *UsageRepository) RecordUsage(ctx context.Context, usage *domain.LLMUsage) error {
	query := `
		INSERT INTO llm_usage (id, tenant_id, usage_date, llm_calls, api_calls, violations, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, usage_date) 
		DO UPDATE SET
			llm_calls = llm_usage.llm_calls + EXCLUDED.llm_calls,
			api_calls = llm_usage.api_calls + EXCLUDED.api_calls,
			violations = llm_usage.violations + EXCLUDED.violations`

	_, err := r.db.Exec(ctx, query,
		usage.ID,
		usage.TenantID,
		usage.UsageDate,
		usage.LLMCalls,
		usage.APICalls,
		usage.Violations,
		usage.CreatedAt,
	)

	return err
}

// GetUsage retrieves usage for a date range
func (r *UsageRepository) GetUsage(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*domain.LLMUsage, error) {
	query := `
		SELECT 
			COALESCE(SUM(llm_calls), 0) as llm_calls,
			COALESCE(SUM(api_calls), 0) as api_calls,
			COALESCE(SUM(violations), 0) as violations
		FROM llm_usage
		WHERE tenant_id = $1 AND usage_date >= $2 AND usage_date < $3`

	var usage domain.LLMUsage
	err := r.db.QueryRow(ctx, query, tenantID, startDate, endDate).Scan(
		&usage.LLMCalls,
		&usage.APICalls,
		&usage.Violations,
	)

	if err != nil {
		return nil, err
	}

	usage.TenantID = tenantID
	usage.UsageDate = startDate
	return &usage, nil
}

// GetUsageForBilling retrieves usage breakdown for billing
func (r *UsageRepository) GetUsageForBilling(ctx context.Context, tenantID uuid.UUID, billingPeriodStart, billingPeriodEnd time.Time) (*domain.UsageBreakdown, error) {
	query := `
		SELECT 
			COALESCE(SUM(api_calls), 0) as api_calls_total,
			COALESCE(SUM(llm_calls), 0) as llm_calls_total,
			COALESCE(SUM(violations), 0) as violations_total
		FROM llm_usage
		WHERE tenant_id = $1 AND usage_date >= $2 AND usage_date < $3`

	var apiCallsTotal, llmCallsTotal, violationsTotal int
	err := r.db.QueryRow(ctx, query, tenantID, billingPeriodStart, billingPeriodEnd).Scan(
		&apiCallsTotal,
		&llmCallsTotal,
		&violationsTotal,
	)

	if err != nil {
		return nil, err
	}

	// Get subscription and plan to calculate limits
	subQuery := `
		SELECT sp.limits
		FROM subscriptions s
		JOIN subscription_plans sp ON s.plan_id = sp.id
		WHERE s.tenant_id = $1 AND s.status IN ('active', 'trial')`

	var limitsJSON []byte
	err = r.db.QueryRow(ctx, subQuery, tenantID).Scan(&limitsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription limits: %w", err)
	}

	// Parse limits (this would need proper JSON unmarshaling)
	// For now, we'll use default values
	limits := domain.Limits{
		APICallsMonthly: 1000000, // 1M default
		LLMCallsMonthly: 10000,   // 10K default
	}

	// Calculate overage
	apiCallsOverage := 0
	if limits.APICallsMonthly > 0 && apiCallsTotal > limits.APICallsMonthly {
		apiCallsOverage = apiCallsTotal - limits.APICallsMonthly
	}

	llmCallsOverage := 0
	if limits.LLMCallsMonthly > 0 && llmCallsTotal > limits.LLMCallsMonthly {
		llmCallsOverage = llmCallsTotal - limits.LLMCallsMonthly
	}

	// Calculate costs
	apiCallsOverageCost := (apiCallsOverage / 1000) * domain.APICallPriceCents
	llmCallsOverageCost := llmCallsOverage * domain.LLMCallPriceCents
	violationsCost := violationsTotal * domain.ViolationPriceCents

	return &domain.UsageBreakdown{
		APICallsTotal:       apiCallsTotal,
		APICallsIncluded:    limits.APICallsMonthly,
		APICallsOverage:     apiCallsOverage,
		APICallsOverageCost: apiCallsOverageCost,
		LLMCallsTotal:       llmCallsTotal,
		LLMCallsIncluded:    limits.LLMCallsMonthly,
		LLMCallsOverage:     llmCallsOverage,
		LLMCallsOverageCost: llmCallsOverageCost,
		ViolationsTotal:     violationsTotal,
		ViolationsCost:      violationsCost,
	}, nil
}

// GetCurrentUsage retrieves current usage for quota checking
func (r *UsageRepository) GetCurrentUsage(ctx context.Context, tenantID uuid.UUID) (*domain.UsageBreakdown, error) {
	// Get current billing period
	subQuery := `
		SELECT current_period_start, current_period_end
		FROM subscriptions
		WHERE tenant_id = $1 AND status IN ('active', 'trial')
		ORDER BY created_at DESC
		LIMIT 1`

	var periodStart, periodEnd time.Time
	err := r.db.QueryRow(ctx, subQuery, tenantID).Scan(&periodStart, &periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get current billing period: %w", err)
	}

	return r.GetUsageForBilling(ctx, tenantID, periodStart, periodEnd)
}

// CreateBilling creates a billing record
func (r *BillingRepository) CreateBilling(ctx context.Context, billing *domain.UsageBilling) error {
	query := `
		INSERT INTO usage_billing (
			id, tenant_id, subscription_id, billing_period_start, billing_period_end,
			base_amount, usage_amount, total_amount, status, stripe_invoice_id,
			usage_breakdown, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	// Convert usage breakdown to JSON
	usageBreakdownJSON, err := json.Marshal(billing.UsageBreakdown)
	if err != nil {
		return fmt.Errorf("failed to marshal usage breakdown: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		billing.ID,
		billing.TenantID,
		billing.SubscriptionID,
		billing.BillingPeriodStart,
		billing.BillingPeriodEnd,
		billing.BaseAmount,
		billing.UsageAmount,
		billing.TotalAmount,
		billing.Status,
		billing.StripeInvoiceID,
		usageBreakdownJSON,
		billing.CreatedAt,
		billing.UpdatedAt,
	)

	return err
}

// GetBillingHistory retrieves billing history for a tenant
func (r *BillingRepository) GetBillingHistory(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageBilling, error) {
	query := `
		SELECT id, tenant_id, subscription_id, billing_period_start, billing_period_end,
		       base_amount, usage_amount, total_amount, status, stripe_invoice_id,
		       usage_breakdown, created_at, updated_at
		FROM usage_billing
		WHERE tenant_id = $1
		ORDER BY billing_period_start DESC`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var billings []*domain.UsageBilling
	for rows.Next() {
		var billing domain.UsageBilling
		var usageBreakdownJSON []byte

		err := rows.Scan(
			&billing.ID,
			&billing.TenantID,
			&billing.SubscriptionID,
			&billing.BillingPeriodStart,
			&billing.BillingPeriodEnd,
			&billing.BaseAmount,
			&billing.UsageAmount,
			&billing.TotalAmount,
			&billing.Status,
			&billing.StripeInvoiceID,
			&usageBreakdownJSON,
			&billing.CreatedAt,
			&billing.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal usage breakdown
		if err := json.Unmarshal(usageBreakdownJSON, &billing.UsageBreakdown); err != nil {
			return nil, fmt.Errorf("failed to unmarshal usage breakdown: %w", err)
		}

		billings = append(billings, &billing)
	}

	return billings, nil
}

// GetBillingByID retrieves a billing record by ID
func (r *BillingRepository) GetBillingByID(ctx context.Context, id uuid.UUID) (*domain.UsageBilling, error) {
	query := `
		SELECT id, tenant_id, subscription_id, billing_period_start, billing_period_end,
		       base_amount, usage_amount, total_amount, status, stripe_invoice_id,
		       usage_breakdown, created_at, updated_at
		FROM usage_billing
		WHERE id = $1`

	var billing domain.UsageBilling
	var usageBreakdownJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&billing.ID,
		&billing.TenantID,
		&billing.SubscriptionID,
		&billing.BillingPeriodStart,
		&billing.BillingPeriodEnd,
		&billing.BaseAmount,
		&billing.UsageAmount,
		&billing.TotalAmount,
		&billing.Status,
		&billing.StripeInvoiceID,
		&usageBreakdownJSON,
		&billing.CreatedAt,
		&billing.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("billing record not found")
		}
		return nil, err
	}

	// Unmarshal usage breakdown
	if err := json.Unmarshal(usageBreakdownJSON, &billing.UsageBreakdown); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage breakdown: %w", err)
	}

	return &billing, nil
}

// UpdateBilling updates a billing record
func (r *BillingRepository) UpdateBilling(ctx context.Context, billing *domain.UsageBilling) error {
	query := `
		UPDATE usage_billing SET
			base_amount = $2,
			usage_amount = $3,
			total_amount = $4,
			status = $5,
			stripe_invoice_id = $6,
			usage_breakdown = $7,
			updated_at = $8
		WHERE id = $1`

	// Convert usage breakdown to JSON
	usageBreakdownJSON, err := json.Marshal(billing.UsageBreakdown)
	if err != nil {
		return fmt.Errorf("failed to marshal usage breakdown: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		billing.ID,
		billing.BaseAmount,
		billing.UsageAmount,
		billing.TotalAmount,
		billing.Status,
		billing.StripeInvoiceID,
		usageBreakdownJSON,
		billing.UpdatedAt,
	)

	return err
}

// PlanRepository implementation
type PlanRepository struct {
	db *pgxpool.Pool
}

// NewPlanRepository creates a new plan repository
func NewPlanRepository(db *pgxpool.Pool) *PlanRepository {
	return &PlanRepository{db: db}
}

// GetAll retrieves all subscription plans
func (r *PlanRepository) GetAll(ctx context.Context) ([]*domain.SubscriptionPlan, error) {
	query := `
		SELECT id, name, display_name, description, price_monthly, price_yearly,
		       features, limits, is_active, created_at, updated_at
		FROM subscription_plans
		WHERE is_active = true
		ORDER BY price_monthly ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*domain.SubscriptionPlan
	for rows.Next() {
		var plan domain.SubscriptionPlan
		var featuresJSON, limitsJSON []byte

		err := rows.Scan(
			&plan.ID,
			&plan.Name,
			&plan.DisplayName,
			&plan.Description,
			&plan.PriceMonthly,
			&plan.PriceYearly,
			&featuresJSON,
			&limitsJSON,
			&plan.IsActive,
			&plan.CreatedAt,
			&plan.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal features and limits
		if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}
		if err := json.Unmarshal(limitsJSON, &plan.Limits); err != nil {
			return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
		}

		plans = append(plans, &plan)
	}

	return plans, nil
}

// GetByID retrieves a subscription plan by ID
func (r *PlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	query := `
		SELECT id, name, display_name, description, price_monthly, price_yearly,
		       features, limits, is_active, created_at, updated_at
		FROM subscription_plans
		WHERE id = $1`

	var plan domain.SubscriptionPlan
	var featuresJSON, limitsJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&plan.ID,
		&plan.Name,
		&plan.DisplayName,
		&plan.Description,
		&plan.PriceMonthly,
		&plan.PriceYearly,
		&featuresJSON,
		&limitsJSON,
		&plan.IsActive,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("plan not found")
		}
		return nil, err
	}

	// Unmarshal features and limits
	if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
		return nil, fmt.Errorf("failed to unmarshal features: %w", err)
	}
	if err := json.Unmarshal(limitsJSON, &plan.Limits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
	}

	return &plan, nil
}

// GetByName retrieves a subscription plan by name
func (r *PlanRepository) GetByName(ctx context.Context, name string) (*domain.SubscriptionPlan, error) {
	query := `
		SELECT id, name, display_name, description, price_monthly, price_yearly,
		       features, limits, is_active, created_at, updated_at
		FROM subscription_plans
		WHERE name = $1`

	var plan domain.SubscriptionPlan
	var featuresJSON, limitsJSON []byte

	err := r.db.QueryRow(ctx, query, name).Scan(
		&plan.ID,
		&plan.Name,
		&plan.DisplayName,
		&plan.Description,
		&plan.PriceMonthly,
		&plan.PriceYearly,
		&featuresJSON,
		&limitsJSON,
		&plan.IsActive,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("plan not found")
		}
		return nil, err
	}

	// Unmarshal features and limits
	if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
		return nil, fmt.Errorf("failed to unmarshal features: %w", err)
	}
	if err := json.Unmarshal(limitsJSON, &plan.Limits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
	}

	return &plan, nil
}
