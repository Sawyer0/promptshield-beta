package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// InvoiceService defines the interface for invoice operations
type InvoiceService interface {
	// Generate invoice for a billing period
	GenerateInvoice(ctx context.Context, req domain.InvoiceGenerationRequest) (*domain.Invoice, error)

	// Get invoice by ID
	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.Invoice, error)

	// List invoices for a tenant
	ListInvoices(ctx context.Context, tenantID uuid.UUID, filters InvoiceFilters) ([]*domain.Invoice, error)

	// Update invoice status
	UpdateInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status domain.InvoiceStatus) error

	// Mark invoice as paid
	MarkInvoiceAsPaid(ctx context.Context, invoiceID uuid.UUID, paidAt time.Time, stripeInvoiceID *string) error

	// Generate PDF for invoice
	GenerateInvoicePDF(ctx context.Context, invoiceID uuid.UUID) (string, error)

	// Send invoice via email
	SendInvoiceEmail(ctx context.Context, invoiceID uuid.UUID) error

	// Get invoice summary for tenant
	GetInvoiceSummary(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*domain.InvoiceSummary, error)

	// Process overdue invoices
	ProcessOverdueInvoices(ctx context.Context) error

	// Get invoice templates
	GetInvoiceTemplates(ctx context.Context, planName string) ([]*domain.InvoiceTemplate, error)

	// Update invoice template
	UpdateInvoiceTemplate(ctx context.Context, template *domain.InvoiceTemplate) error
}

// InvoiceFilters represents filters for listing invoices
type InvoiceFilters struct {
	Status             *domain.InvoiceStatus `json:"status,omitempty"`
	BillingPeriodStart *time.Time            `json:"billing_period_start,omitempty"`
	BillingPeriodEnd   *time.Time            `json:"billing_period_end,omitempty"`
	DueDateStart       *time.Time            `json:"due_date_start,omitempty"`
	DueDateEnd         *time.Time            `json:"due_date_end,omitempty"`
	Limit              int                   `json:"limit,omitempty"`
	Offset             int                   `json:"offset,omitempty"`
}

// InvoiceRepository defines the interface for invoice data operations
type InvoiceRepository interface {
	// Invoice operations
	CreateInvoice(ctx context.Context, invoice *domain.Invoice) error
	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.Invoice, error)
	UpdateInvoice(ctx context.Context, invoice *domain.Invoice) error
	ListInvoices(ctx context.Context, tenantID uuid.UUID, filters InvoiceFilters) ([]*domain.Invoice, error)
	GetInvoiceByNumber(ctx context.Context, invoiceNumber string) (*domain.Invoice, error)

	// Line item operations
	CreateLineItem(ctx context.Context, lineItem *domain.InvoiceLineItem) error
	GetLineItems(ctx context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceLineItem, error)
	DeleteLineItems(ctx context.Context, invoiceID uuid.UUID) error

	// Template operations
	GetInvoiceTemplates(ctx context.Context, planName string) ([]*domain.InvoiceTemplate, error)
	GetInvoiceTemplate(ctx context.Context, planName, templateName string) (*domain.InvoiceTemplate, error)
	UpdateInvoiceTemplate(ctx context.Context, template *domain.InvoiceTemplate) error

	// Summary operations
	GetInvoiceSummary(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*domain.InvoiceSummary, error)
	GetOverdueInvoices(ctx context.Context) ([]*domain.Invoice, error)
}

// PDFGenerator defines the interface for generating invoice PDFs
type PDFGenerator interface {
	GenerateInvoicePDF(ctx context.Context, invoice *domain.Invoice) ([]byte, error)
	UploadPDF(ctx context.Context, invoiceID uuid.UUID, pdfData []byte) (string, error)
}

// EmailService defines the interface for sending invoice emails
type EmailService interface {
	SendInvoiceEmail(ctx context.Context, invoice *domain.Invoice, template *domain.InvoiceTemplate) error
}
