package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application"
	"github.com/promptshield/promptshield/internal/domain"
	sharedcontext "github.com/promptshield/promptshield/internal/shared/context"
	"github.com/promptshield/promptshield/internal/shared/httputil"
)

// BillingHandler handles billing-related HTTP endpoints
type BillingHandler struct {
	billingService application.BillingService
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingService application.BillingService) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
	}
}

// RegisterRoutes registers billing routes
func (h *BillingHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/billing", func(r chi.Router) {
		// Plans
		r.Get("/plans", h.GetPlans)
		r.Get("/plans/{planId}", h.GetPlan)

		// Subscriptions
		r.Get("/subscription", h.GetSubscription)
		r.Post("/subscription", h.CreateSubscription)
		r.Put("/subscription/{subscriptionId}", h.UpdateSubscription)
		r.Delete("/subscription/{subscriptionId}", h.CancelSubscription)

		// Usage
		r.Get("/usage", h.GetUsage)
		r.Post("/usage", h.RecordUsage)

		// Billing
		r.Get("/history", h.GetBillingHistory)
		r.Post("/process", h.ProcessBilling)

		// Quota
		r.Get("/quota", h.CheckQuota)

		// Stripe
		r.Post("/stripe/customer", h.CreateStripeCustomer)
		r.Post("/stripe/subscription", h.CreateStripeSubscription)
		r.Post("/stripe/webhook", h.HandleStripeWebhook)
	})
}

// GetPlans retrieves all available subscription plans
func (h *BillingHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.billingService.GetPlans(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to retrieve plans", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"plans": plans,
	})
}

// GetPlan retrieves a specific subscription plan
func (h *BillingHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	planIDStr := chi.URLParam(r, "planId")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid plan ID", err)
		return
	}

	plan, err := h.billingService.GetPlan(r.Context(), planID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "Plan not found", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, plan)
}

// GetSubscription retrieves the current tenant's subscription
func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantIDFromContext(r)
	if tenantID == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	subscription, err := h.billingService.GetSubscription(r.Context(), *tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "Subscription not found", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, subscription)
}

// CreateSubscriptionRequest represents a request to create a subscription
type CreateSubscriptionRequest struct {
	PlanID       uuid.UUID           `json:"plan_id" validate:"required"`
	BillingCycle domain.BillingCycle `json:"billing_cycle" validate:"required,oneof=monthly yearly"`
}

// CreateSubscription creates a new subscription
func (h *BillingHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantIDFromContext(r)
	if tenantID == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	var req CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	subscription, err := h.billingService.CreateSubscription(r.Context(), *tenantID, req.PlanID, req.BillingCycle)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to create subscription", err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, subscription)
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	PlanID            *uuid.UUID           `json:"plan_id,omitempty"`
	BillingCycle      *domain.BillingCycle `json:"billing_cycle,omitempty"`
	CancelAtPeriodEnd *bool                `json:"cancel_at_period_end,omitempty"`
}

// UpdateSubscription updates a subscription
func (h *BillingHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionIDStr := chi.URLParam(r, "subscriptionId")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid subscription ID", err)
		return
	}

	var req UpdateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	updates := application.SubscriptionUpdates{
		PlanID:            req.PlanID,
		BillingCycle:      req.BillingCycle,
		CancelAtPeriodEnd: req.CancelAtPeriodEnd,
	}

	subscription, err := h.billingService.UpdateSubscription(r.Context(), subscriptionID, updates)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to update subscription", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, subscription)
}

// CancelSubscription cancels a subscription
func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionIDStr := chi.URLParam(r, "subscriptionId")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid subscription ID", err)
		return
	}

	cancelAtPeriodEnd := r.URL.Query().Get("cancel_at_period_end") == "true"

	if err := h.billingService.CancelSubscription(r.Context(), subscriptionID, cancelAtPeriodEnd); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to cancel subscription", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Subscription canceled"})
}

// GetUsage retrieves usage for the current tenant
func (h *BillingHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantIDFromContext(r)
	if tenantID == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	// Parse date range from query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "Invalid start_date format", err)
			return
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -30) // Default to last 30 days
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "Invalid end_date format", err)
			return
		}
	} else {
		endDate = time.Now()
	}

	usage, err := h.billingService.GetUsage(r.Context(), *tenantID, startDate, endDate)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to retrieve usage", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, usage)
}

// RecordUsageRequest represents a request to record usage
type RecordUsageRequest struct {
	APICalls   int `json:"api_calls" validate:"min=0"`
	LLMCalls   int `json:"llm_calls" validate:"min=0"`
	Violations int `json:"violations" validate:"min=0"`
}

// RecordUsage records usage for billing
func (h *BillingHandler) RecordUsage(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantIDFromContext(r)
	if tenantID == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	var req RecordUsageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	usage := application.UsageRecord{
		APICalls:   req.APICalls,
		LLMCalls:   req.LLMCalls,
		Violations: req.Violations,
		Timestamp:  time.Now(),
	}

	if err := h.billingService.RecordUsage(r.Context(), *tenantID, usage); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to record usage", err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{"message": "Usage recorded"})
}

// GetBillingHistory retrieves billing history for the current tenant
func (h *BillingHandler) GetBillingHistory(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantIDFromContext(r)
	if tenantID == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	history, err := h.billingService.GetBillingHistory(r.Context(), *tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to retrieve billing history", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"billing_history": history,
	})
}

// ProcessBillingRequest represents a request to process billing
type ProcessBillingRequest struct {
	BillingPeriodStart string `json:"billing_period_start" validate:"required"`
	BillingPeriodEnd   string `json:"billing_period_end" validate:"required"`
}

// ProcessBilling processes billing for the current tenant
func (h *BillingHandler) ProcessBilling(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantIDFromContext(r)
	if tenantID == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	var req ProcessBillingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	startDate, err := time.Parse("2006-01-02", req.BillingPeriodStart)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid billing_period_start format", err)
		return
	}

	endDate, err := time.Parse("2006-01-02", req.BillingPeriodEnd)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid billing_period_end format", err)
		return
	}

	billing, err := h.billingService.ProcessBilling(r.Context(), *tenantID, startDate, endDate)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to process billing", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, billing)
}

// CheckQuota checks quota for the current tenant
func (h *BillingHandler) CheckQuota(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantIDFromContext(r)
	if tenantID == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	resourceTypeStr := r.URL.Query().Get("resource_type")
	if resourceTypeStr == "" {
		httputil.WriteError(w, http.StatusBadRequest, "resource_type parameter is required", nil)
		return
	}

	resourceType := application.QuotaResource(resourceTypeStr)
	quotaStatus, err := h.billingService.CheckQuota(r.Context(), *tenantID, resourceType)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to check quota", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, quotaStatus)
}

// CreateStripeCustomerRequest represents a request to create a Stripe customer
type CreateStripeCustomerRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// CreateStripeCustomer creates a Stripe customer
func (h *BillingHandler) CreateStripeCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantIDFromContext(r)
	if tenantID == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	var req CreateStripeCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	customerID, err := h.billingService.CreateStripeCustomer(r.Context(), *tenantID, req.Email)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to create Stripe customer", err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"customer_id": customerID,
	})
}

// CreateStripeSubscriptionRequest represents a request to create a Stripe subscription
type CreateStripeSubscriptionRequest struct {
	CustomerID string `json:"customer_id" validate:"required"`
	PriceID    string `json:"price_id" validate:"required"`
}

// CreateStripeSubscription creates a Stripe subscription
func (h *BillingHandler) CreateStripeSubscription(w http.ResponseWriter, r *http.Request) {
	var req CreateStripeSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	subscriptionID, err := h.billingService.CreateStripeSubscription(r.Context(), req.CustomerID, req.PriceID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to create Stripe subscription", err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"subscription_id": subscriptionID,
	})
}

// HandleStripeWebhook handles Stripe webhook events
func (h *BillingHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := httputil.ReadBody(r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}

	// Get the Stripe signature header
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		httputil.WriteError(w, http.StatusBadRequest, "Missing Stripe signature", nil)
		return
	}

	// Handle the webhook
	if err := h.billingService.HandleStripeWebhook(r.Context(), "", body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Failed to handle webhook", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// getTenantIDFromContext extracts tenant ID from request context
func getTenantIDFromContext(r *http.Request) *uuid.UUID {
	tenantID, ok := sharedcontext.GetTenantID(r.Context())
	if !ok {
		return nil
	}
	return &tenantID
}
