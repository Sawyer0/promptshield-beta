package pdf

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application"
	"github.com/promptshield/promptshield/internal/domain"
)

// SimplePDFGenerator is a basic PDF generator implementation
// In production, you would use a proper PDF library like unidoc/unipdf or gofpdf
type SimplePDFGenerator struct {
	storageService PDFStorageService
}

// PDFStorageService defines the interface for storing PDFs
type PDFStorageService interface {
	UploadPDF(ctx context.Context, invoiceID uuid.UUID, pdfData []byte) (string, error)
}

// NewSimplePDFGenerator creates a new PDF generator
func NewSimplePDFGenerator(storageService PDFStorageService) application.PDFGenerator {
	return &SimplePDFGenerator{
		storageService: storageService,
	}
}

func (g *SimplePDFGenerator) GenerateInvoicePDF(ctx context.Context, invoice *domain.Invoice) ([]byte, error) {
	// This is a simplified implementation
	// In production, you would use a proper PDF library

	var buf bytes.Buffer

	// Generate a simple text-based "PDF" (in reality, this would be proper PDF)
	fmt.Fprintf(&buf, "INVOICE\n")
	fmt.Fprintf(&buf, "========\n\n")
	fmt.Fprintf(&buf, "Invoice Number: %s\n", invoice.InvoiceNumber)
	fmt.Fprintf(&buf, "Date: %s\n", invoice.CreatedAt.Format("2006-01-02"))
	fmt.Fprintf(&buf, "Due Date: %s\n", invoice.DueDate.Format("2006-01-02"))
	fmt.Fprintf(&buf, "Status: %s\n\n", invoice.Status)

	fmt.Fprintf(&buf, "Billing Period: %s to %s\n\n",
		invoice.BillingPeriodStart.Format("2006-01-02"),
		invoice.BillingPeriodEnd.Format("2006-01-02"))

	fmt.Fprintf(&buf, "Line Items:\n")
	fmt.Fprintf(&buf, "----------\n")

	for _, item := range invoice.LineItems {
		fmt.Fprintf(&buf, "%s\n", item.Description)
		fmt.Fprintf(&buf, "  Quantity: %d\n", item.Quantity)
		fmt.Fprintf(&buf, "  Unit Price: $%.2f\n", float64(item.UnitPriceCents)/100.0)
		fmt.Fprintf(&buf, "  Total: $%.2f\n\n", float64(item.TotalCents)/100.0)
	}

	fmt.Fprintf(&buf, "Subtotal: $%.2f\n", float64(invoice.SubtotalCents)/100.0)
	fmt.Fprintf(&buf, "Tax: $%.2f\n", float64(invoice.TaxCents)/100.0)
	fmt.Fprintf(&buf, "Total: $%.2f\n", float64(invoice.TotalCents)/100.0)

	return buf.Bytes(), nil
}

func (g *SimplePDFGenerator) UploadPDF(ctx context.Context, invoiceID uuid.UUID, pdfData []byte) (string, error) {
	return g.storageService.UploadPDF(ctx, invoiceID, pdfData)
}

// LocalFileStorageService is a simple file storage implementation
type LocalFileStorageService struct {
	basePath string
}

// NewLocalFileStorageService creates a new local file storage service
func NewLocalFileStorageService(basePath string) PDFStorageService {
	return &LocalFileStorageService{
		basePath: basePath,
	}
}

func (s *LocalFileStorageService) UploadPDF(ctx context.Context, invoiceID uuid.UUID, pdfData []byte) (string, error) {
	// In production, this would upload to S3, GCS, or similar
	// For now, we'll just return a mock URL
	filename := fmt.Sprintf("invoice_%s_%d.pdf", invoiceID.String(), time.Now().Unix())
	url := fmt.Sprintf("https://storage.promptshield.com/invoices/%s", filename)

	slog.Info("Mock PDF upload", "invoice_id", invoiceID, "filename", filename, "size", len(pdfData))

	return url, nil
}
