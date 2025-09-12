package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

type invoiceService struct {
	repo         InvoiceRepository
	pdfGenerator PDFGenerator
	emailService EmailService
	billingSvc   BillingService
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(
	repo InvoiceRepository,
	pdfGenerator PDFGenerator,
	emailService EmailService,
	billingSvc BillingService,
) InvoiceService {
	return &invoiceService{
		repo:         repo,
		pdfGenerator: pdfGenerator,
		emailService: emailService,
		billingSvc:   billingSvc,
	}
}

func (s *invoiceService) GenerateInvoice(ctx context.Context, req domain.InvoiceGenerationRequest) (*domain.Invoice, error) {
	// Check if invoice already exists for this period
	existingInvoice, err := s.repo.GetInvoiceByNumber(ctx, s.generateInvoiceNumber(req.TenantID, req.BillingPeriodStart))
	if err == nil && existingInvoice != nil && !req.ForceRegenerate {
		return existingInvoice, nil
	}

	// Get subscription details
	subscription, err := s.billingSvc.GetSubscription(ctx, req.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get usage data for the billing period
	usage, err := s.billingSvc.GetUsage(ctx, req.TenantID, req.BillingPeriodStart, req.BillingPeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage data: %w", err)
	}

	// Calculate costs
	subtotalCents, lineItems, err := s.calculateInvoiceAmounts(ctx, subscription, usage, req.BillingPeriodStart, req.BillingPeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate invoice amounts: %w", err)
	}

	// Calculate tax (simplified - in production, use proper tax calculation service)
	taxCents := int64(float64(subtotalCents) * 0.08) // 8% tax rate

	// Create invoice
	invoice := &domain.Invoice{
		ID:                 uuid.New(),
		TenantID:           req.TenantID,
		SubscriptionID:     req.SubscriptionID,
		InvoiceNumber:      s.generateInvoiceNumber(req.TenantID, req.BillingPeriodStart),
		Status:             domain.InvoiceStatusDraft,
		BillingPeriodStart: req.BillingPeriodStart,
		BillingPeriodEnd:   req.BillingPeriodEnd,
		SubtotalCents:      subtotalCents,
		TaxCents:           taxCents,
		TotalCents:         subtotalCents + taxCents,
		Currency:           "usd",
		DueDate:            req.BillingPeriodEnd.AddDate(0, 0, 30), // 30 days after period end
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		LineItems:          lineItems,
	}

	// Save invoice
	if err := s.repo.CreateInvoice(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Save line items
	for _, lineItem := range lineItems {
		lineItem.InvoiceID = invoice.ID
		if err := s.repo.CreateLineItem(ctx, &lineItem); err != nil {
			slog.Error("Failed to create line item", "error", err, "invoice_id", invoice.ID)
		}
	}

	slog.Info("Generated invoice", "invoice_id", invoice.ID, "invoice_number", invoice.InvoiceNumber, "total_cents", invoice.TotalCents)

	return invoice, nil
}

func (s *invoiceService) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.Invoice, error) {
	return s.repo.GetInvoice(ctx, invoiceID)
}

func (s *invoiceService) ListInvoices(ctx context.Context, tenantID uuid.UUID, filters InvoiceFilters) ([]*domain.Invoice, error) {
	return s.repo.ListInvoices(ctx, tenantID, filters)
}

func (s *invoiceService) UpdateInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status domain.InvoiceStatus) error {
	invoice, err := s.repo.GetInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	invoice.Status = status
	invoice.UpdatedAt = time.Now()

	return s.repo.UpdateInvoice(ctx, invoice)
}

func (s *invoiceService) MarkInvoiceAsPaid(ctx context.Context, invoiceID uuid.UUID, paidAt time.Time, stripeInvoiceID *string) error {
	invoice, err := s.repo.GetInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	invoice.Status = domain.InvoiceStatusPaid
	invoice.PaidAt = &paidAt
	invoice.StripeInvoiceID = stripeInvoiceID
	invoice.UpdatedAt = time.Now()

	return s.repo.UpdateInvoice(ctx, invoice)
}

func (s *invoiceService) GenerateInvoicePDF(ctx context.Context, invoiceID uuid.UUID) (string, error) {
	invoice, err := s.repo.GetInvoice(ctx, invoiceID)
	if err != nil {
		return "", fmt.Errorf("failed to get invoice: %w", err)
	}

	// Get line items
	lineItems, err := s.repo.GetLineItems(ctx, invoiceID)
	if err != nil {
		return "", fmt.Errorf("failed to get line items: %w", err)
	}
	invoice.LineItems = lineItems

	// Generate PDF
	pdfData, err := s.pdfGenerator.GenerateInvoicePDF(ctx, invoice)
	if err != nil {
		return "", fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Upload PDF
	pdfURL, err := s.pdfGenerator.UploadPDF(ctx, invoiceID, pdfData)
	if err != nil {
		return "", fmt.Errorf("failed to upload PDF: %w", err)
	}

	// Update invoice with PDF URL
	invoice.PDFURL = &pdfURL
	if err := s.repo.UpdateInvoice(ctx, invoice); err != nil {
		slog.Error("Failed to update invoice with PDF URL", "error", err, "invoice_id", invoiceID)
	}

	return pdfURL, nil
}

func (s *invoiceService) SendInvoiceEmail(ctx context.Context, invoiceID uuid.UUID) error {
	invoice, err := s.repo.GetInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Get subscription to determine plan
	subscription, err := s.billingSvc.GetSubscription(ctx, invoice.SubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get invoice template
	templates, err := s.repo.GetInvoiceTemplates(ctx, subscription.PlanName)
	if err != nil {
		return fmt.Errorf("failed to get invoice templates: %w", err)
	}

	if len(templates) == 0 {
		return fmt.Errorf("no invoice template found for plan: %s", subscription.PlanName)
	}

	template := templates[0] // Use first template

	// Send email
	return s.emailService.SendInvoiceEmail(ctx, invoice, template)
}

func (s *invoiceService) GetInvoiceSummary(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*domain.InvoiceSummary, error) {
	return s.repo.GetInvoiceSummary(ctx, tenantID, startDate, endDate)
}

func (s *invoiceService) ProcessOverdueInvoices(ctx context.Context) error {
	overdueInvoices, err := s.repo.GetOverdueInvoices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get overdue invoices: %w", err)
	}

	for _, invoice := range overdueInvoices {
		// Update status to overdue
		if err := s.UpdateInvoiceStatus(ctx, invoice.ID, domain.InvoiceStatusOverdue); err != nil {
			slog.Error("Failed to mark invoice as overdue", "error", err, "invoice_id", invoice.ID)
			continue
		}

		// Send reminder email
		if err := s.SendInvoiceEmail(ctx, invoice.ID); err != nil {
			slog.Error("Failed to send overdue invoice email", "error", err, "invoice_id", invoice.ID)
		}
	}

	return nil
}

func (s *invoiceService) GetInvoiceTemplates(ctx context.Context, planName string) ([]*domain.InvoiceTemplate, error) {
	return s.repo.GetInvoiceTemplates(ctx, planName)
}

func (s *invoiceService) UpdateInvoiceTemplate(ctx context.Context, template *domain.InvoiceTemplate) error {
	return s.repo.UpdateInvoiceTemplate(ctx, template)
}

// Helper methods

func (s *invoiceService) generateInvoiceNumber(tenantID uuid.UUID, periodStart time.Time) string {
	year := periodStart.Year()
	month := int(periodStart.Month())
	tenantShort := tenantID.String()[:8]
	return fmt.Sprintf("INV-%d%02d-%s", year, month, tenantShort)
}

func (s *invoiceService) calculateInvoiceAmounts(ctx context.Context, subscription *domain.Subscription, usage *domain.UsageMetric, periodStart, periodEnd time.Time) (int64, []domain.InvoiceLineItem, error) {
	var lineItems []domain.InvoiceLineItem
	var subtotalCents int64

	// Get subscription plan details
	plan, err := s.billingSvc.GetPlan(ctx, subscription.PlanID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	// Base subscription cost
	basePriceCents := plan.PriceMonthly
	if subscription.BillingCycle == "yearly" {
		basePriceCents = plan.PriceYearly
	}

	// Add subscription line item
	lineItems = append(lineItems, domain.InvoiceLineItem{
		ID:             uuid.New(),
		Description:    fmt.Sprintf("%s Plan - %s", plan.DisplayName, subscription.BillingCycle),
		Quantity:       1,
		UnitPriceCents: basePriceCents,
		TotalCents:     basePriceCents,
		LineType:       domain.LineTypeSubscription,
		CreatedAt:      time.Now(),
	})
	subtotalCents += basePriceCents

	// Calculate LLM usage overage
	if usage.LLMCalls > int64(plan.Limits["llm_calls_monthly"].(int)) {
		overageCalls := usage.LLMCalls - int64(plan.Limits["llm_calls_monthly"].(int))
		overagePriceCents := int64(0.01 * 100) // $0.01 per overage call
		overageTotalCents := overageCalls * overagePriceCents

		lineItems = append(lineItems, domain.InvoiceLineItem{
			ID:             uuid.New(),
			Description:    fmt.Sprintf("LLM Calls Overage (%d calls)", overageCalls),
			Quantity:       int(overageCalls),
			UnitPriceCents: overagePriceCents,
			TotalCents:     overageTotalCents,
			LineType:       domain.LineTypeOverage,
			Metadata: map[string]interface{}{
				"included_calls": plan.Limits["llm_calls_monthly"],
				"used_calls":     usage.LLMCalls,
				"overage_calls":  overageCalls,
			},
			CreatedAt: time.Now(),
		})
		subtotalCents += overageTotalCents
	}

	return subtotalCents, lineItems, nil
}
