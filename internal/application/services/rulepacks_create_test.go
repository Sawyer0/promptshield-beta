package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/contracts"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
)

// racyRepo stores rulepacks without enforcing unique names and sleeps between
// reads and writes, so an unguarded check-then-create would let duplicates in.
type racyRepo struct {
	contracts.RulepackRepository
	mu    sync.Mutex
	packs []contracts.RulepackInfo
}

func (r *racyRepo) ListByTenant(context.Context, uuid.UUID) ([]contracts.RulepackInfo, error) {
	r.mu.Lock()
	out := append([]contracts.RulepackInfo(nil), r.packs...)
	r.mu.Unlock()
	time.Sleep(time.Millisecond)
	return out, nil
}

func (r *racyRepo) Create(_ context.Context, _ uuid.UUID, name, desc string) (uuid.UUID, error) {
	time.Sleep(time.Millisecond)
	id := uuid.New()
	r.mu.Lock()
	r.packs = append(r.packs, contracts.RulepackInfo{ID: id, Name: name, Description: desc})
	r.mu.Unlock()
	return id, nil
}

func createConcurrently(t *testing.T, svc *RulepackService, n int) (created int, dupes int) {
	t.Helper()
	tenant := uuid.New()
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = svc.Create(context.Background(), tenant, "same-name", "")
		}(i)
	}
	close(start)
	wg.Wait()
	for _, err := range errs {
		switch {
		case err == nil:
			created++
		case errors.Is(err, ErrRulepackNameExists):
			dupes++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	return created, dupes
}

func TestRulepackCreate_ConcurrentDuplicateNames(t *testing.T) {
	const n = 20
	repo := &racyRepo{}
	created, dupes := createConcurrently(t, &RulepackService{repo: repo}, n)
	if created != 1 || dupes != n-1 {
		t.Fatalf("created=%d dupes=%d, want 1 and %d", created, dupes, n-1)
	}
	if len(repo.packs) != 1 {
		t.Fatalf("repository holds %d packs, want 1", len(repo.packs))
	}
}

// Two services sharing one store model two processes; only the repository's
// own constraint can stop the race between them.
func TestRulepackCreate_RepositoryEnforcesUniqueness(t *testing.T) {
	repo := memory.NewRulepackRepository()
	tenant := uuid.New()
	a, b := &RulepackService{repo: repo}, &RulepackService{repo: repo}

	var wg sync.WaitGroup
	results := make([]error, 2)
	for i, svc := range []*RulepackService{a, b} {
		wg.Add(1)
		go func(i int, svc *RulepackService) {
			defer wg.Done()
			_, results[i] = svc.Create(context.Background(), tenant, "shared", "")
		}(i, svc)
	}
	wg.Wait()

	ok := 0
	for _, err := range results {
		if err == nil {
			ok++
		} else if !errors.Is(err, ErrRulepackNameExists) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d creates succeeded, want 1", ok)
	}

	if _, err := repo.Create(context.Background(), tenant, "shared", ""); !errors.Is(err, contracts.ErrRulepackNameExists) {
		t.Fatalf("memory repo allowed a duplicate name: %v", err)
	}
	if _, err := repo.Create(context.Background(), uuid.New(), "shared", ""); err != nil {
		t.Fatalf("same name for another tenant should be allowed: %v", err)
	}
}
