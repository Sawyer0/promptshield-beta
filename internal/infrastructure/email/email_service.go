package email

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	texttemplate "text/template"
	"time"

	"github.com/promptshield/promptshield/internal/application"
	"github.com/promptshield/promptshield/internal/domain"
)

// SimpleEmailService is a basic email service implementation
// In production, you would use a proper email service like SendGrid, SES, etc.
type SimpleEmailService struct {
	// In production, this would have actual email sending capabilities
}

// NewSimpleEmailService creates a new email service
func NewSimpleEmailService() application.EmailService {
	return &SimpleEmailService{}
}

func (s *SimpleEmailService) SendInvoiceEmail(ctx context.Context, invoice *domain.Invoice, template *domain.InvoiceTemplate) error {
	// This is a simplified implementation
	// In production, you would use a proper email service

	// Render email template
	subject, _, err := s.renderEmailTemplate(invoice, template)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// In production, this would actually send the email
	slog.Info("Mock email sent",
		"invoice_id", invoice.ID,
		"invoice_number", invoice.InvoiceNumber,
		"subject", subject,
		"total_cents", invoice.TotalCents,
	)

	// Simulate email sending delay
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (s *SimpleEmailService) renderEmailTemplate(invoice *domain.Invoice, template *domain.InvoiceTemplate) (string, string, error) {
	// Template data
	data := map[string]interface{}{
		"invoice_number":       invoice.InvoiceNumber,
		"billing_period_start": invoice.BillingPeriodStart.Format("January 2, 2006"),
		"billing_period_end":   invoice.BillingPeriodEnd.Format("January 2, 2006"),
		"due_date":             invoice.DueDate.Format("January 2, 2006"),
		"subtotal":             fmt.Sprintf("$%.2f", float64(invoice.SubtotalCents)/100.0),
		"tax":                  fmt.Sprintf("$%.2f", float64(invoice.TaxCents)/100.0),
		"total":                fmt.Sprintf("$%.2f", float64(invoice.TotalCents)/100.0),
		"currency":             invoice.Currency,
		"line_items":           invoice.LineItems,
	}

	// Render subject
	subjectTmpl, err := texttemplate.New("subject").Parse(template.SubjectTemplate)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse subject template: %w", err)
	}

	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute subject template: %w", err)
	}

	// Render body
	bodyTmpl, err := texttemplate.New("body").Parse(template.BodyTemplate)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse body template: %w", err)
	}

	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute body template: %w", err)
	}

	// Add footer if present
	if template.FooterTemplate != nil {
		footerTmpl, err := texttemplate.New("footer").Parse(*template.FooterTemplate)
		if err != nil {
			return "", "", fmt.Errorf("failed to parse footer template: %w", err)
		}

		var footerBuf bytes.Buffer
		if err := footerTmpl.Execute(&footerBuf, data); err != nil {
			return "", "", fmt.Errorf("failed to execute footer template: %w", err)
		}

		bodyBuf.WriteString("\n\n")
		bodyBuf.WriteString(footerBuf.String())
	}

	return subjectBuf.String(), bodyBuf.String(), nil
}
