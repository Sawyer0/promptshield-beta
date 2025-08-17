package mocks

import (
	"context"
	"io"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/pkg/types"
	"github.com/stretchr/testify/mock"
)

// MockScanner mocks the Scanner interface for testing
type MockScanner struct {
	mock.Mock
}

func (m *MockScanner) LoadRulePacks(packs []interface{}) error {
	args := m.Called(packs)
	return args.Error(0)
}

func (m *MockScanner) HasSemanticAnalyzer() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockScanner) SetRuleDefaults(defaults map[string]interface{}) {
	m.Called(defaults)
}

func (m *MockScanner) SetCompositionStrategy(strategy string) {
	m.Called(strategy)
}

func (m *MockScanner) SetRuntimeContext(ctx map[string]interface{}) {
	m.Called(ctx)
}

func (m *MockScanner) ScanPathsOrdered(ctx context.Context, paths []string, workers int) ([]types.ScanResult, error) {
	args := m.Called(ctx, paths, workers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.ScanResult), args.Error(1)
}

func (m *MockScanner) ScanReaderOrdered(ctx context.Context, readers []io.Reader, names []string, workers int) ([]types.ScanResult, error) {
	args := m.Called(ctx, readers, names, workers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.ScanResult), args.Error(1)
}

func (m *MockScanner) ScanPathsStream(ctx context.Context, paths []string, workers int, resultChan chan<- types.ScanResult) error {
	args := m.Called(ctx, paths, workers, resultChan)
	return args.Error(0)
}

func (m *MockScanner) ScanReaderStream(ctx context.Context, readers []io.Reader, names []string, workers int, resultChan chan<- types.ScanResult) error {
	args := m.Called(ctx, readers, names, workers, resultChan)
	return args.Error(0)
}

// FakeScanner is a lightweight test double that captures calls
type FakeScanner struct {
	*scanner.Scanner
	// Captured values
	LoadedPacks         []rules.RulePack
	RuleDefaults        map[string]interface{}
	CompositionStrategy string
	RuntimeContext      map[string]string
	LastWorkerCount     int

	// Control behavior
	HasSemantic   bool
	ScanResults   []types.ScanResult
	ScanError     error
	SleepDuration int // milliseconds to sleep in scan methods
}

func NewFakeScanner() *FakeScanner {
	return &FakeScanner{
		Scanner:     scanner.New(1024 * 1024),
		ScanResults: []types.ScanResult{},
	}
}

// AsScanner returns the embedded *scanner.Scanner for wiring into services.
func (f *FakeScanner) AsScanner() *scanner.Scanner { return f.Scanner }

func (f *FakeScanner) LoadRulePacks(packs []rules.RulePack) {
	f.LoadedPacks = packs
}

func (f *FakeScanner) HasSemanticAnalyzer() bool {
	return f.HasSemantic
}

func (f *FakeScanner) SetRuleDefaults(ms int64, cs, ww bool) {
	f.RuleDefaults = map[string]interface{}{"perRuleMs": ms, "caseSensitive": cs, "wholeWord": ww}
}

func (f *FakeScanner) SetCompositionStrategy(strategy string) {
	f.CompositionStrategy = strategy
}

func (f *FakeScanner) SetRuntimeContext(ctx map[string]string) {
	f.RuntimeContext = ctx
}

func (f *FakeScanner) SetFileSizeLimit(int64)           {}
func (f *FakeScanner) SetMaxPatternLength(int)          {}
func (f *FakeScanner) SetTotalScanBudget(time.Duration) {}
func (f *FakeScanner) SetBufferBytes(int)               {}
func (f *FakeScanner) SetChunkOverlap(int)              {}

// ScanPathsOrdered matches the current scanner.Scanner signature with window & includeBinary flags.
func (f *FakeScanner) ScanPathsOrdered(ctx context.Context, paths []string, workers, window int, includeBinary bool) ([]types.ScanResult, error) {
	f.LastWorkerCount = workers

	if f.SleepDuration > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(f.SleepDuration) * time.Millisecond):
		}
	}

	return f.ScanResults, f.ScanError
}

// ScanReaderOrdered updated signature.
func (f *FakeScanner) ScanReaderOrdered(ctx context.Context, readers []io.Reader, names []string, workers, window int, includeBinary bool) ([]types.ScanResult, error) {
	f.LastWorkerCount = workers

	if f.SleepDuration > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(f.SleepDuration) * time.Millisecond):
		}
	}

	return f.ScanResults, f.ScanError
}

func (f *FakeScanner) ScanPathsStream(ctx context.Context, paths []string, workers int, resultChan chan<- types.ScanResult) error {
	f.LastWorkerCount = workers

	if f.SleepDuration > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(f.SleepDuration) * time.Millisecond):
		}
	}

	// Send results to channel
	for _, result := range f.ScanResults {
		select {
		case resultChan <- result:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return f.ScanError
}

func (f *FakeScanner) ScanReaderStream(ctx context.Context, readers []io.Reader, names []string, workers int, resultChan chan<- types.ScanResult) error {
	f.LastWorkerCount = workers

	if f.SleepDuration > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(f.SleepDuration) * time.Millisecond):
		}
	}

	// Send results to channel
	for _, result := range f.ScanResults {
		select {
		case resultChan <- result:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return f.ScanError
}

// ScanPathsOrderedStream streams results via callback in deterministic order.
func (f *FakeScanner) ScanPathsOrderedStream(ctx context.Context, paths []string, workers, window int, includeBinary bool, emit func(types.ScanResult) error) error {
	f.LastWorkerCount = workers
	if f.SleepDuration > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(f.SleepDuration) * time.Millisecond):
		}
	}
	for _, r := range f.ScanResults {
		if err := emit(r); err != nil {
			return err
		}
	}
	return f.ScanError
}
