package decisions

import (
	"fmt"
	"sync"
	"testing"
	"time"

	pkg "github.com/promptshield/promptshield/pkg/types"
)

func rec(id, decision string, latency float64) Record {
	return Record{RequestID: id, Decision: decision, LatencyMs: latency}
}

func TestLogRecentNewestFirstAndFiltered(t *testing.T) {
	l := NewLog(10)
	l.Add(rec("a", Allow, 1))
	l.Add(rec("b", Deny, 2))
	l.Add(rec("c", Allow, 3))

	got := l.Recent(Query{})
	if len(got) != 3 || got[0].RequestID != "c" || got[2].RequestID != "a" {
		t.Fatalf("want newest first c..a, got %+v", got)
	}
	denies := l.Recent(Query{Decision: Deny})
	if len(denies) != 1 || denies[0].RequestID != "b" {
		t.Fatalf("want only b, got %+v", denies)
	}
	if limited := l.Recent(Query{Limit: 2}); len(limited) != 2 || limited[1].RequestID != "b" {
		t.Fatalf("want c,b, got %+v", limited)
	}
}

func TestLogEvictsOldestButKeepsLifetimeCounts(t *testing.T) {
	l := NewLog(3)
	for i := 0; i < 5; i++ {
		l.Add(rec(fmt.Sprintf("r%d", i), Allow, 1))
	}
	l.Add(rec("blocked", Deny, 1))

	got := l.Recent(Query{})
	if len(got) != 3 || got[0].RequestID != "blocked" || got[2].RequestID != "r3" {
		t.Fatalf("unexpected retained records: %+v", got)
	}
	if _, ok := l.Get("r0"); ok {
		t.Fatal("evicted record should not be found")
	}
	if r, ok := l.Get("r4"); !ok || r.Decision != Allow {
		t.Fatalf("want r4 retained, got %+v %v", r, ok)
	}
	s := l.Stats()
	if s.Total != 6 || s.ByDecision[Allow] != 5 || s.ByDecision[Deny] != 1 {
		t.Fatalf("lifetime counts wrong: %+v", s)
	}
	if s.Retained != 3 || s.Capacity != 3 {
		t.Fatalf("retained/capacity wrong: %+v", s)
	}
}

func TestStatsPercentilesAndBlockRate(t *testing.T) {
	l := NewLog(100)
	for i := 1; i <= 20; i++ {
		decision := Allow
		if i%4 == 0 {
			decision = Quarantine
		}
		l.Add(rec(fmt.Sprintf("r%d", i), decision, float64(i)))
	}
	s := l.Stats()
	if s.LatencyP50Ms != 10 || s.LatencyP95Ms != 19 {
		t.Fatalf("p50/p95 = %v/%v, want 10/19", s.LatencyP50Ms, s.LatencyP95Ms)
	}
	if s.BlockRate != 0.25 {
		t.Fatalf("block rate = %v, want 0.25", s.BlockRate)
	}
	if s.LastMinute != 20 {
		t.Fatalf("last minute = %d, want 20", s.LastMinute)
	}
}

func TestStatsEmptyLog(t *testing.T) {
	s := NewLog(0).Stats()
	if s.Total != 0 || s.LatencyP95Ms != 0 || s.BlockRate != 0 || s.Capacity != DefaultCapacity {
		t.Fatalf("unexpected empty stats: %+v", s)
	}
	for _, d := range []string{Allow, Deny, Quarantine} {
		if _, ok := s.ByDecision[d]; !ok {
			t.Fatalf("by_decision missing %q", d)
		}
	}
}

func TestStatsLastMinuteIgnoresOlderRecords(t *testing.T) {
	l := NewLog(10)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }
	l.Add(Record{RequestID: "old", Decision: Allow, Time: now.Add(-2 * time.Minute)})
	l.Add(Record{RequestID: "new", Decision: Allow})
	if got := l.Stats().LastMinute; got != 1 {
		t.Fatalf("last minute = %d, want 1", got)
	}
}

func TestLogConcurrentUse(t *testing.T) {
	l := NewLog(50)
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				l.Add(rec(fmt.Sprintf("w%d-%d", w, i), Allow, 1))
				_ = l.Stats()
				_ = l.Recent(Query{Limit: 5})
			}
		}(w)
	}
	wg.Wait()
	if got := l.Stats().Total; got != 800 {
		t.Fatalf("total = %d, want 800", got)
	}
}

func TestMatchesFromAndEventData(t *testing.T) {
	m := MatchesFrom([]pkg.Violation{{RuleID: "pi-1", Severity: "CRITICAL", Category: "injection", ResponseAction: "block", Message: "raw", ResponseMessage: "blocked"}})
	if len(m) != 1 || m[0].Action != "block" || m[0].Message != "blocked" {
		t.Fatalf("unexpected matches: %+v", m)
	}
	r := Record{RequestID: "x", Decision: Deny, Reason: "pi-1", Violations: 1, Matches: m}
	data := r.EventData()
	for _, k := range []string{"request_id", "decision", "reason", "violations", "matches", "latency_ms"} {
		if _, ok := data[k]; !ok {
			t.Fatalf("event data missing %q", k)
		}
	}
	if data["violations"] != 1 {
		t.Fatalf("violations should stay a count, got %v", data["violations"])
	}
}
