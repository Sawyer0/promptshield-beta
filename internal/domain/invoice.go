package domain

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceStatus represents the status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusDraft   InvoiceStatus = "draft"
	InvoiceStatusSent    InvoiceStatus = "sent"
	InvoiceStatusPaid    InvoiceStatus = "paid"
	InvoiceStatusOverdue InvoiceStatus = "overdue"
	InvoiceStatusVoid    InvoiceStatus = "void"
)

// LineType represents the type of invoice line item
type LineType string

const (
	LineTypeSubscription LineType = "subscription"
	LineTypeUsage        LineType = "usage"
	LineTypeOverage      LineType = "overage"
	LineTypeDiscount     LineType = "discount"
	LineTypeTax          LineType = "tax"
)

// Invoice represents a billing invoice
type Invoice struct {
	ID                 uuid.UUID     `json:"id" db:"id"`
	TenantID           uuid.UUID     `json:"tenant_id" db:"tenant_id"`
	SubscriptionID     uuid.UUID     `json:"subscription_id" db:"subscription_id"`
	InvoiceNumber      string        `json:"invoice_number" db:"invoice_number"`
	Status             InvoiceStatus `json:"status" db:"status"`
	BillingPeriodStart time.Time     `json:"billing_period_start" db:"billing_period_start"`
	BillingPeriodEnd   time.Time     `json:"billing_period_end" db:"billing_period_end"`
	SubtotalCents      int64         `json:"subtotal_cents" db:"subtotal_cents"`
	TaxCents           int64         `json:"tax_cents" db:"tax_cents"`
	TotalCents         int64         `json:"total_cents" db:"total_cents"`
	Currency           string        `json:"currency" db:"currency"`
	DueDate            time.Time     `json:"due_date" db:"due_date"`
	PaidAt             *time.Time    `json:"paid_at" db:"paid_at"`
	StripeInvoiceID    *string       `json:"stripe_invoice_id" db:"stripe_invoice_id"`
	PDFURL             *string       `json:"pdf_url" db:"pdf_url"`
	CreatedAt          time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at" db:"updated_at"`

	// Related data
	LineItems []InvoiceLineItem `json:"line_items,omitempty"`
}

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	InvoiceID      uuid.UUID              `json:"invoice_id" db:"invoice_id"`
	Description    string                 `json:"description" db:"description"`
	Quantity       int                    `json:"quantity" db:"quantity"`
	UnitPriceCents int64                  `json:"unit_price_cents" db:"unit_price_cents"`
	TotalCents     int64                  `json:"total_cents" db:"total_cents"`
	LineType       LineType               `json:"line_type" db:"line_type"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// InvoiceTemplate represents an email template for invoices
type InvoiceTemplate struct {
	ID              uuid.UUID `json:"id" db:"id"`
	PlanName        string    `json:"plan_name" db:"plan_name"`
	TemplateName    string    `json:"template_name" db:"template_name"`
	SubjectTemplate string    `json:"subject_template" db:"subject_template"`
	BodyTemplate    string    `json:"body_template" db:"body_template"`
	FooterTemplate  *string   `json:"footer_template" db:"footer_template"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// InvoiceGenerationRequest represents a request to generate invoices
type InvoiceGenerationRequest struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	SubscriptionID     uuid.UUID `json:"subscription_id"`
	BillingPeriodStart time.Time `json:"billing_period_start"`
	BillingPeriodEnd   time.Time `json:"billing_period_end"`
	ForceRegenerate    bool      `json:"force_regenerate"`
}

// InvoiceSummary represents a summary of invoice data for reporting
type InvoiceSummary struct {
	TotalInvoices       int64 `json:"total_invoices"`
	TotalAmountCents    int64 `json:"total_amount_cents"`
	PaidAmountCents     int64 `json:"paid_amount_cents"`
	OutstandingCents    int64 `json:"outstanding_cents"`
	OverdueCents        int64 `json:"overdue_cents"`
	AverageInvoiceCents int64 `json:"average_invoice_cents"`
}
