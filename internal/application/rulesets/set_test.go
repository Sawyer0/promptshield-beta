package rulesets

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/promptshield/promptshield/internal/rules"
)

const packYAML = `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: demo
  version: 1.2.0
  description: demo pack
rules:
  - id: pi-ignore
    name: Ignore instructions
    level: 1
    severity: CRITICAL
    keywords: ["ignore previous instructions", "disregard previous"]
    response:
      action: block
  - id: pii-ssn
    name: SSN
    level: 2
    severity: HIGH
    enabled: false
    patterns:
      - name: ssn
        regex: '\b\d{3}-\d{2}-\d{4}\b'
`

func writePack(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pack.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNewSetLoadsAndSummarizes(t *testing.T) {
	s, err := NewSet(writePack(t, packYAML))
	if err != nil {
		t.Fatalf("NewSet: %v", err)
	}
	snap := s.Snapshot()
	if snap.Generation != 1 || snap.RuleCount != 2 || len(snap.Packs) != 1 || snap.LoadedAt.IsZero() {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
	p := snap.Packs[0]
	if p.Name != "demo" || p.Version != "1.2.0" {
		t.Fatalf("unexpected pack meta: %+v", p)
	}
	first, second := p.Rules[0], p.Rules[1]
	if first.ID != "pi-ignore" || first.Action != "block" || !first.Enabled || first.Keywords != 2 {
		t.Fatalf("unexpected first rule: %+v", first)
	}
	if second.Enabled || second.Patterns != 1 || second.Action != "" {
		t.Fatalf("unexpected second rule: %+v", second)
	}
	if len(s.Packs()) != 1 {
		t.Fatalf("Packs() should return loaded packs")
	}
}

func TestReloadPicksUpChangesAndNotifies(t *testing.T) {
	path := writePack(t, packYAML)
	s, err := NewSet(path)
	if err != nil {
		t.Fatal(err)
	}
	var notified []rules.RulePack
	s.OnChange(func(p []rules.RulePack) { notified = p })

	updated := packYAML + `  - id: extra
    name: Extra
    level: 1
    severity: LOW
    keywords: ["x"]
`
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := s.Reload()
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if snap.Generation != 2 || snap.RuleCount != 3 {
		t.Fatalf("unexpected snapshot after reload: %+v", snap)
	}
	if len(notified) != 1 || len(notified[0].Rules) != 3 {
		t.Fatalf("listener not called with new packs: %+v", notified)
	}
}

func TestReloadFailureKeepsActivePacks(t *testing.T) {
	calls := 0
	load := func(string) ([]rules.RulePack, error) {
		calls++
		if calls > 1 {
			return nil, errors.New("boom")
		}
		return []rules.RulePack{{Metadata: rules.Metadata{Name: "kept"}}}, nil
	}
	s, err := NewSetWithLoader("rules.yaml", load)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.Reload()
	if err == nil {
		t.Fatal("want reload error")
	}
	if snap.Generation != 1 || snap.Packs[0].Name != "kept" {
		t.Fatalf("active packs should be kept: %+v", snap)
	}
}

func TestReloadWithoutSource(t *testing.T) {
	s, err := NewSet("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reload(); !errors.Is(err, ErrNoSource) {
		t.Fatalf("want ErrNoSource, got %v", err)
	}
	if snap := s.Snapshot(); snap.Generation != 0 || len(snap.Packs) != 0 {
		t.Fatalf("empty set expected: %+v", snap)
	}
}

func TestNewSetReportsLoadError(t *testing.T) {
	s, err := NewSet(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("want error for missing path")
	}
	if s == nil || len(s.Packs()) != 0 {
		t.Fatal("set should be usable and empty")
	}
}
