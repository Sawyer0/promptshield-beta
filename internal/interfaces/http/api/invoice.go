package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application"
	"github.com/promptshield/promptshield/internal/domain"
	sharedcontext "github.com/promptshield/promptshield/internal/shared/context"
	"github.com/promptshield/promptshield/internal/shared/httputil"
)

type InvoiceHandler struct {
	invoiceService application.InvoiceService
}

// NewInvoiceHandler creates a new invoice handler
func NewInvoiceHandler(invoiceService application.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: invoiceService,
	}
}

// RegisterRoutes registers invoice routes
func (h *InvoiceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/invoices", func(r chi.Router) {
		r.Get("/", h.ListInvoices)
		r.Post("/generate", h.GenerateInvoice)
		r.Get("/summary", h.GetInvoiceSummary)
		r.Route("/{invoiceID}", func(r chi.Router) {
			r.Get("/", h.GetInvoice)
			r.Put("/status", h.UpdateInvoiceStatus)
			r.Post("/pdf", h.GenerateInvoicePDF)
			r.Post("/send", h.SendInvoiceEmail)
			r.Put("/mark-paid", h.MarkInvoiceAsPaid)
		})
	})
}

// ListInvoices handles GET /invoices
func (h *InvoiceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID, err := sharedcontext.GetTenantID(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant ID required", nil)
		return
	}

	// Parse query parameters
	filters := application.InvoiceFilters{}

	if status := r.URL.Query().Get("status"); status != "" {
		statusEnum := domain.InvoiceStatus(status)
		filters.Status = &statusEnum
	}

	if startDate := r.URL.Query().Get("billing_period_start"); startDate != "" {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			filters.BillingPeriodStart = &parsed
		}
	}

	if endDate := r.URL.Query().Get("billing_period_end"); endDate != "" {
		if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
			filters.BillingPeriodEnd = &parsed
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 {
			filters.Limit = parsed
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if parsed, err := strconv.Atoi(offset); err == nil && parsed >= 0 {
			filters.Offset = parsed
		}
	}

	invoices, err := h.invoiceService.ListInvoices(r.Context(), tenantID, filters)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list invoices", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"invoices": invoices,
		"count":    len(invoices),
	})
}

// GenerateInvoice handles POST /invoices/generate
func (h *InvoiceHandler) GenerateInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, err := sharedcontext.GetTenantID(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant ID required", nil)
		return
	}

	var req domain.InvoiceGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON", err)
		return
	}

	// Set tenant ID from context
	req.TenantID = tenantID

	invoice, err := h.invoiceService.GenerateInvoice(r.Context(), req)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate invoice", err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, invoice)
}

// GetInvoice handles GET /invoices/{invoiceID}
func (h *InvoiceHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := chi.URLParam(r, "invoiceID")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid invoice ID", err)
		return
	}

	invoice, err := h.invoiceService.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "NOT_FOUND", "invoice not found", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, invoice)
}

// UpdateInvoiceStatus handles PUT /invoices/{invoiceID}/status
func (h *InvoiceHandler) UpdateInvoiceStatus(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := chi.URLParam(r, "invoiceID")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid invoice ID", err)
		return
	}

	var req struct {
		Status domain.InvoiceStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON", err)
		return
	}

	if err := h.invoiceService.UpdateInvoiceStatus(r.Context(), invoiceID, req.Status); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update invoice status", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Invoice status updated successfully",
	})
}

// GenerateInvoicePDF handles POST /invoices/{invoiceID}/pdf
func (h *InvoiceHandler) GenerateInvoicePDF(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := chi.URLParam(r, "invoiceID")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid invoice ID", err)
		return
	}

	pdfURL, err := h.invoiceService.GenerateInvoicePDF(r.Context(), invoiceID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate PDF", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"pdf_url": pdfURL,
	})
}

// SendInvoiceEmail handles POST /invoices/{invoiceID}/send
func (h *InvoiceHandler) SendInvoiceEmail(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := chi.URLParam(r, "invoiceID")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid invoice ID", err)
		return
	}

	if err := h.invoiceService.SendInvoiceEmail(r.Context(), invoiceID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send invoice email", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Invoice email sent successfully",
	})
}

// MarkInvoiceAsPaid handles PUT /invoices/{invoiceID}/mark-paid
func (h *InvoiceHandler) MarkInvoiceAsPaid(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := chi.URLParam(r, "invoiceID")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid invoice ID", err)
		return
	}

	var req struct {
		PaidAt          time.Time `json:"paid_at"`
		StripeInvoiceID *string   `json:"stripe_invoice_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON", err)
		return
	}

	if err := h.invoiceService.MarkInvoiceAsPaid(r.Context(), invoiceID, req.PaidAt, req.StripeInvoiceID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to mark invoice as paid", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Invoice marked as paid successfully",
	})
}

// GetInvoiceSummary handles GET /invoices/summary
func (h *InvoiceHandler) GetInvoiceSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, err := sharedcontext.GetTenantID(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant ID required", nil)
		return
	}

	// Parse date range
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	var err1, err2 error

	if startDateStr != "" {
		startDate, err1 = time.Parse("2006-01-02", startDateStr)
	} else {
		startDate = time.Now().AddDate(0, -12, 0) // Default to 12 months ago
	}

	if endDateStr != "" {
		endDate, err2 = time.Parse("2006-01-02", endDateStr)
	} else {
		endDate = time.Now() // Default to now
	}

	if err1 != nil || err2 != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid date format", nil)
		return
	}

	summary, err := h.invoiceService.GetInvoiceSummary(r.Context(), tenantID, startDate, endDate)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get invoice summary", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, summary)
}
