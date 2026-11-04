// Package rulesets holds the RulePacks the gateway enforces at runtime and
// lets them be reloaded from disk without restarting the process.
package rulesets

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
)

// ErrNoSource is returned by Reload when the set was created without a path.
var ErrNoSource = errors.New("no rulepack path configured")

// Loader reads RulePacks from a file or directory.
type Loader func(path string) ([]rules.RulePack, error)

// Set is the active collection of RulePacks. It is safe for concurrent use.
type Set struct {
	mu         sync.RWMutex
	path       string
	packs      []rules.RulePack
	loadedAt   time.Time
	generation int
	listeners  []func([]rules.RulePack)
	load       Loader
}

// NewSet loads the packs at path (when non-empty) using rules.LoadPacks.
// A load failure is returned alongside an empty, usable set.
func NewSet(path string) (*Set, error) {
	return NewSetWithLoader(path, rules.LoadPacks)
}

// NewSetWithLoader is NewSet with an injectable loader.
func NewSetWithLoader(path string, load Loader) (*Set, error) {
	s := &Set{path: path, load: load}
	if path == "" {
		return s, nil
	}
	if _, err := s.Reload(); err != nil {
		return s, err
	}
	return s, nil
}

// Packs returns the active packs. Callers must not modify the result.
func (s *Set) Packs() []rules.RulePack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.packs
}

// OnChange registers fn to run with the new packs after every successful reload.
func (s *Set) OnChange(fn func([]rules.RulePack)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

// Reload re-reads the configured path. On failure the active packs are kept.
func (s *Set) Reload() (Snapshot, error) {
	if s.path == "" {
		return s.Snapshot(), ErrNoSource
	}
	packs, err := s.load(s.path)
	if err != nil {
		return s.Snapshot(), fmt.Errorf("load %s: %w", s.path, err)
	}
	s.mu.Lock()
	s.packs = packs
	s.loadedAt = time.Now().UTC()
	s.generation++
	listeners := append([]func([]rules.RulePack){}, s.listeners...)
	s.mu.Unlock()
	for _, fn := range listeners {
		fn(packs)
	}
	return s.Snapshot(), nil
}

// Snapshot describes the active packs for display.
func (s *Set) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := Snapshot{
		Source:     s.path,
		LoadedAt:   s.loadedAt,
		Generation: s.generation,
		Packs:      make([]Pack, 0, len(s.packs)),
	}
	for _, p := range s.packs {
		pack := summarizePack(p)
		snap.RuleCount += len(pack.Rules)
		snap.Packs = append(snap.Packs, pack)
	}
	return snap
}
