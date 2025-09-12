package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application"
	"github.com/promptshield/promptshield/internal/domain"
)

type invoiceRepository struct {
	db *sql.DB
}

// NewInvoiceRepository creates a new invoice repository
func NewInvoiceRepository(db *sql.DB) application.InvoiceRepository {
	return &invoiceRepository{db: db}
}

func (r *invoiceRepository) CreateInvoice(ctx context.Context, invoice *domain.Invoice) error {
	query := `
		INSERT INTO invoices (
			id, tenant_id, subscription_id, invoice_number, status,
			billing_period_start, billing_period_end, subtotal_cents,
			tax_cents, total_cents, currency, due_date, stripe_invoice_id,
			pdf_url, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err := r.db.ExecContext(ctx, query,
		invoice.ID, invoice.TenantID, invoice.SubscriptionID, invoice.InvoiceNumber,
		invoice.Status, invoice.BillingPeriodStart, invoice.BillingPeriodEnd,
		invoice.SubtotalCents, invoice.TaxCents, invoice.TotalCents,
		invoice.Currency, invoice.DueDate, invoice.StripeInvoiceID,
		invoice.PDFURL, invoice.CreatedAt, invoice.UpdatedAt,
	)

	return err
}

func (r *invoiceRepository) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.Invoice, error) {
	query := `
		SELECT id, tenant_id, subscription_id, invoice_number, status,
			   billing_period_start, billing_period_end, subtotal_cents,
			   tax_cents, total_cents, currency, due_date, paid_at,
			   stripe_invoice_id, pdf_url, created_at, updated_at
		FROM invoices WHERE id = $1`

	var invoice domain.Invoice
	err := r.db.QueryRowContext(ctx, query, invoiceID).Scan(
		&invoice.ID, &invoice.TenantID, &invoice.SubscriptionID, &invoice.InvoiceNumber,
		&invoice.Status, &invoice.BillingPeriodStart, &invoice.BillingPeriodEnd,
		&invoice.SubtotalCents, &invoice.TaxCents, &invoice.TotalCents,
		&invoice.Currency, &invoice.DueDate, &invoice.PaidAt,
		&invoice.StripeInvoiceID, &invoice.PDFURL, &invoice.CreatedAt, &invoice.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice not found")
		}
		return nil, err
	}

	return &invoice, nil
}

func (r *invoiceRepository) UpdateInvoice(ctx context.Context, invoice *domain.Invoice) error {
	query := `
		UPDATE invoices SET
			status = $2, subtotal_cents = $3, tax_cents = $4, total_cents = $5,
			due_date = $6, paid_at = $7, stripe_invoice_id = $8, pdf_url = $9,
			updated_at = $10
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		invoice.ID, invoice.Status, invoice.SubtotalCents, invoice.TaxCents,
		invoice.TotalCents, invoice.DueDate, invoice.PaidAt,
		invoice.StripeInvoiceID, invoice.PDFURL, invoice.UpdatedAt,
	)

	return err
}

func (r *invoiceRepository) ListInvoices(ctx context.Context, tenantID uuid.UUID, filters application.InvoiceFilters) ([]*domain.Invoice, error) {
	query := `
		SELECT id, tenant_id, subscription_id, invoice_number, status,
			   billing_period_start, billing_period_end, subtotal_cents,
			   tax_cents, total_cents, currency, due_date, paid_at,
			   stripe_invoice_id, pdf_url, created_at, updated_at
		FROM invoices WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	argIndex := 2

	// Add filters
	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filters.Status)
		argIndex++
	}

	if filters.BillingPeriodStart != nil {
		query += fmt.Sprintf(" AND billing_period_start >= $%d", argIndex)
		args = append(args, *filters.BillingPeriodStart)
		argIndex++
	}

	if filters.BillingPeriodEnd != nil {
		query += fmt.Sprintf(" AND billing_period_end <= $%d", argIndex)
		args = append(args, *filters.BillingPeriodEnd)
		argIndex++
	}

	if filters.DueDateStart != nil {
		query += fmt.Sprintf(" AND due_date >= $%d", argIndex)
		args = append(args, *filters.DueDateStart)
		argIndex++
	}

	if filters.DueDateEnd != nil {
		query += fmt.Sprintf(" AND due_date <= $%d", argIndex)
		args = append(args, *filters.DueDateEnd)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		var invoice domain.Invoice
		err := rows.Scan(
			&invoice.ID, &invoice.TenantID, &invoice.SubscriptionID, &invoice.InvoiceNumber,
			&invoice.Status, &invoice.BillingPeriodStart, &invoice.BillingPeriodEnd,
			&invoice.SubtotalCents, &invoice.TaxCents, &invoice.TotalCents,
			&invoice.Currency, &invoice.DueDate, &invoice.PaidAt,
			&invoice.StripeInvoiceID, &invoice.PDFURL, &invoice.CreatedAt, &invoice.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, &invoice)
	}

	return invoices, nil
}

func (r *invoiceRepository) GetInvoiceByNumber(ctx context.Context, invoiceNumber string) (*domain.Invoice, error) {
	query := `
		SELECT id, tenant_id, subscription_id, invoice_number, status,
			   billing_period_start, billing_period_end, subtotal_cents,
			   tax_cents, total_cents, currency, due_date, paid_at,
			   stripe_invoice_id, pdf_url, created_at, updated_at
		FROM invoices WHERE invoice_number = $1`

	var invoice domain.Invoice
	err := r.db.QueryRowContext(ctx, query, invoiceNumber).Scan(
		&invoice.ID, &invoice.TenantID, &invoice.SubscriptionID, &invoice.InvoiceNumber,
		&invoice.Status, &invoice.BillingPeriodStart, &invoice.BillingPeriodEnd,
		&invoice.SubtotalCents, &invoice.TaxCents, &invoice.TotalCents,
		&invoice.Currency, &invoice.DueDate, &invoice.PaidAt,
		&invoice.StripeInvoiceID, &invoice.PDFURL, &invoice.CreatedAt, &invoice.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice not found")
		}
		return nil, err
	}

	return &invoice, nil
}

func (r *invoiceRepository) CreateLineItem(ctx context.Context, lineItem *domain.InvoiceLineItem) error {
	query := `
		INSERT INTO invoice_line_items (
			id, invoice_id, description, quantity, unit_price_cents,
			total_cents, line_type, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		lineItem.ID, lineItem.InvoiceID, lineItem.Description, lineItem.Quantity,
		lineItem.UnitPriceCents, lineItem.TotalCents, lineItem.LineType,
		lineItem.Metadata, lineItem.CreatedAt,
	)

	return err
}

func (r *invoiceRepository) GetLineItems(ctx context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceLineItem, error) {
	query := `
		SELECT id, invoice_id, description, quantity, unit_price_cents,
			   total_cents, line_type, metadata, created_at
		FROM invoice_line_items WHERE invoice_id = $1 ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, query, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []*domain.InvoiceLineItem
	for rows.Next() {
		var lineItem domain.InvoiceLineItem
		err := rows.Scan(
			&lineItem.ID, &lineItem.InvoiceID, &lineItem.Description, &lineItem.Quantity,
			&lineItem.UnitPriceCents, &lineItem.TotalCents, &lineItem.LineType,
			&lineItem.Metadata, &lineItem.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lineItems = append(lineItems, &lineItem)
	}

	return lineItems, nil
}

func (r *invoiceRepository) DeleteLineItems(ctx context.Context, invoiceID uuid.UUID) error {
	query := `DELETE FROM invoice_line_items WHERE invoice_id = $1`
	_, err := r.db.ExecContext(ctx, query, invoiceID)
	return err
}

func (r *invoiceRepository) GetInvoiceTemplates(ctx context.Context, planName string) ([]*domain.InvoiceTemplate, error) {
	query := `
		SELECT id, plan_name, template_name, subject_template, body_template,
			   footer_template, is_active, created_at, updated_at
		FROM invoice_templates WHERE plan_name = $1 AND is_active = true
		ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, query, planName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*domain.InvoiceTemplate
	for rows.Next() {
		var template domain.InvoiceTemplate
		err := rows.Scan(
			&template.ID, &template.PlanName, &template.TemplateName,
			&template.SubjectTemplate, &template.BodyTemplate, &template.FooterTemplate,
			&template.IsActive, &template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		templates = append(templates, &template)
	}

	return templates, nil
}

func (r *invoiceRepository) GetInvoiceTemplate(ctx context.Context, planName, templateName string) (*domain.InvoiceTemplate, error) {
	query := `
		SELECT id, plan_name, template_name, subject_template, body_template,
			   footer_template, is_active, created_at, updated_at
		FROM invoice_templates WHERE plan_name = $1 AND template_name = $2`

	var template domain.InvoiceTemplate
	err := r.db.QueryRowContext(ctx, query, planName, templateName).Scan(
		&template.ID, &template.PlanName, &template.TemplateName,
		&template.SubjectTemplate, &template.BodyTemplate, &template.FooterTemplate,
		&template.IsActive, &template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice template not found")
		}
		return nil, err
	}

	return &template, nil
}

func (r *invoiceRepository) UpdateInvoiceTemplate(ctx context.Context, template *domain.InvoiceTemplate) error {
	query := `
		UPDATE invoice_templates SET
			subject_template = $3, body_template = $4, footer_template = $5,
			is_active = $6, updated_at = $7
		WHERE plan_name = $1 AND template_name = $2`

	_, err := r.db.ExecContext(ctx, query,
		template.PlanName, template.TemplateName, template.SubjectTemplate,
		template.BodyTemplate, template.FooterTemplate, template.IsActive,
		template.UpdatedAt,
	)

	return err
}

func (r *invoiceRepository) GetInvoiceSummary(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*domain.InvoiceSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_invoices,
			COALESCE(SUM(total_cents), 0) as total_amount_cents,
			COALESCE(SUM(CASE WHEN status = 'paid' THEN total_cents ELSE 0 END), 0) as paid_amount_cents,
			COALESCE(SUM(CASE WHEN status IN ('sent', 'overdue') THEN total_cents ELSE 0 END), 0) as outstanding_cents,
			COALESCE(SUM(CASE WHEN status = 'overdue' THEN total_cents ELSE 0 END), 0) as overdue_cents,
			COALESCE(AVG(total_cents), 0) as average_invoice_cents
		FROM invoices 
		WHERE tenant_id = $1 
		AND billing_period_start >= $2 
		AND billing_period_end <= $3`

	var summary domain.InvoiceSummary
	err := r.db.QueryRowContext(ctx, query, tenantID, startDate, endDate).Scan(
		&summary.TotalInvoices, &summary.TotalAmountCents, &summary.PaidAmountCents,
		&summary.OutstandingCents, &summary.OverdueCents, &summary.AverageInvoiceCents,
	)

	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (r *invoiceRepository) GetOverdueInvoices(ctx context.Context) ([]*domain.Invoice, error) {
	query := `
		SELECT id, tenant_id, subscription_id, invoice_number, status,
			   billing_period_start, billing_period_end, subtotal_cents,
			   tax_cents, total_cents, currency, due_date, paid_at,
			   stripe_invoice_id, pdf_url, created_at, updated_at
		FROM invoices 
		WHERE status = 'sent' AND due_date < $1
		ORDER BY due_date`

	rows, err := r.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		var invoice domain.Invoice
		err := rows.Scan(
			&invoice.ID, &invoice.TenantID, &invoice.SubscriptionID, &invoice.InvoiceNumber,
			&invoice.Status, &invoice.BillingPeriodStart, &invoice.BillingPeriodEnd,
			&invoice.SubtotalCents, &invoice.TaxCents, &invoice.TotalCents,
			&invoice.Currency, &invoice.DueDate, &invoice.PaidAt,
			&invoice.StripeInvoiceID, &invoice.PDFURL, &invoice.CreatedAt, &invoice.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, &invoice)
	}

	return invoices, nil
}
