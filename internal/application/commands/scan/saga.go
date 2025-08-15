package scancommand

import (
	"context"
	"fmt"
	"time"

	"github.com/promptshield/promptshield/internal/observability/chaos"
)

// SagaCoordinator coordinates distributed scan steps with compensations.
// This is a minimal in-process skeleton designed to evolve to distributed orchestration.
type SagaCoordinator struct {
	chaos *chaos.Controller
}

func NewSagaCoordinator() *SagaCoordinator { return &SagaCoordinator{chaos: chaos.NewFromEnv()} }

// Step represents a unit of work with an optional compensation in case of failure.
type Step struct {
	Name         string
	Execute      func(ctx context.Context) error
	Compensate   func(ctx context.Context) error
	MaxRetries   int
	RetryBackoff time.Duration
}

// Run executes steps sequentially with backtracking compensation on failure.
func (s *SagaCoordinator) Run(ctx context.Context, steps []Step) error {
	completed := make([]Step, 0, len(steps))
	for _, st := range steps {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Chaos: random delay/failure injection between steps
		s.chaos.MaybeDelay()
		if s.chaos.ShouldFail() {
			return s.compensate(ctx, completed, fmt.Errorf("chaos: fail before step %s", st.Name))
		}
		var err error
		for attempt := 0; attempt <= st.MaxRetries; attempt++ {
			err = st.Execute(ctx)
			if err == nil {
				break
			}
			if st.RetryBackoff > 0 {
				t := time.NewTimer(st.RetryBackoff)
				<-t.C
			}
		}
		if err != nil {
			return s.compensate(ctx, completed, fmt.Errorf("step %s failed: %w", st.Name, err))
		}
		completed = append(completed, st)
	}
	return nil
}

func (s *SagaCoordinator) compensate(ctx context.Context, completed []Step, cause error) error {
	for i := len(completed) - 1; i >= 0; i-- {
		st := completed[i]
		if st.Compensate != nil {
			_ = st.Compensate(ctx) // best-effort
		}
	}
	return cause
}
