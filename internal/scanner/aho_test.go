package scanner

import "testing"

func TestAho_FindAll(t *testing.T) {
	a := NewAho([]string{"secret", "password", "api"})
	hay := []byte("this secret contains a password and an api key")
	ms := a.FindAll(hay)
	if len(ms) < 3 {
		t.Fatalf("expected >=3 matches, got %d", len(ms))
	}
	// ensure indices are sane
	for _, m := range ms {
		if m.Index < 0 || m.Index >= len(hay) {
			t.Fatalf("bad index %d", m.Index)
		}
		if m.PatternIndex < 0 || m.PatternIndex >= len(a.patterns) {
			t.Fatalf("bad pattern index %d", m.PatternIndex)
		}
	}
}
