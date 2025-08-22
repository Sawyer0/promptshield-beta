package mocks

import (
	"time"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
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


// FakeScanner is a lightweight test double that captures calls
type FakeScanner struct {
	*scanner.Scanner
	// Captured values
	LoadedPacks         []rules.RulePack
	RuleDefaults        map[string]interface{}
	CompositionStrategy string
	RuntimeContext      map[string]string

	// Control behavior
	HasSemantic bool
}

func NewFakeScanner() *FakeScanner {
	return &FakeScanner{
		Scanner: scanner.ScanEngineCstor(1024 * 1024),
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

