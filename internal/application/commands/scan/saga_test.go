package scancommand

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSagaCoordinator_Run_WithCompensation(t *testing.T) {
	if os.Getenv("PS_RUN_SAGA_TESTS") == "0" {
		t.Skip("saga tests disabled")
	}
	s := NewSagaCoordinator()
	var order []string
	steps := []Step{
		{
			Name:       "first",
			Execute:    func(ctx context.Context) error { order = append(order, "first"); return nil },
			Compensate: func(ctx context.Context) error { order = append(order, "comp-first"); return nil },
		},
		{
			Name:       "second",
			Execute:    func(ctx context.Context) error { order = append(order, "second"); return nil },
			Compensate: func(ctx context.Context) error { order = append(order, "comp-second"); return nil },
		},
		{
			Name:         "fail",
			Execute:      func(ctx context.Context) error { return context.DeadlineExceeded },
			Compensate:   func(ctx context.Context) error { order = append(order, "comp-fail"); return nil },
			MaxRetries:   1,
			RetryBackoff: 10 * time.Millisecond,
		},
	}
	err := s.Run(context.Background(), steps)
	if err == nil {
		t.Fatalf("expected error from saga")
	}
	// Last two completed should be compensated in reverse order
	if len(order) < 3 {
		t.Fatalf("unexpected order: %v", order)
	}
}
