package application

import (
	"context"
	"log/slog"
	"time"
)

// SchedulerService defines the interface for scheduled tasks
type SchedulerService interface {
	// Start starts the scheduler
	Start(ctx context.Context) error

	// Stop stops the scheduler
	Stop() error

	// ScheduleOverdueInvoiceProcessing schedules overdue invoice processing
	ScheduleOverdueInvoiceProcessing(interval time.Duration) error
}

type schedulerService struct {
	invoiceService InvoiceService
	ticker         *time.Ticker
	done           chan bool
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(invoiceService InvoiceService) SchedulerService {
	return &schedulerService{
		invoiceService: invoiceService,
		done:           make(chan bool),
	}
}

func (s *schedulerService) Start(ctx context.Context) error {
	slog.Info("Starting scheduler service")

	// Start overdue invoice processing every hour
	if err := s.ScheduleOverdueInvoiceProcessing(time.Hour); err != nil {
		return err
	}

	// Keep the scheduler running
	go func() {
		for {
			select {
			case <-s.done:
				slog.Info("Scheduler service stopped")
				return
			case <-ctx.Done():
				slog.Info("Scheduler service context cancelled")
				return
			}
		}
	}()

	return nil
}

func (s *schedulerService) Stop() error {
	slog.Info("Stopping scheduler service")

	if s.ticker != nil {
		s.ticker.Stop()
	}

	close(s.done)
	return nil
}

func (s *schedulerService) ScheduleOverdueInvoiceProcessing(interval time.Duration) error {
	s.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				slog.Info("Processing overdue invoices")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				if err := s.invoiceService.ProcessOverdueInvoices(ctx); err != nil {
					slog.Error("Failed to process overdue invoices", "error", err)
				}
				cancel()

			case <-s.done:
				return
			}
		}
	}()

	return nil
}
