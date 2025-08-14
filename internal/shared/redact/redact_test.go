package redact

import "testing"

func TestRedact_Patterns(t *testing.T) {
	cases := []struct {
		in      string
		wantSub string
	}{
		{"key sk-abcdefghijklmnopqrstuvwxyzabcdef", "[REDACTED]"},
		{"x anthropic-abc123def456", "[REDACTED]"},
		{"api_key=SECRETVALUE1234567890", "[REDACTED]"},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "[REDACTED]"},
		{"AKIAABCDEFGHIJKLMNOP", "[REDACTED]"},
		{"AIzaSyA-abcdefghijklmnopqrstuvwxyz012345", "[REDACTED]"},
		{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", "[REDACTED]"},
	}
	for _, c := range cases {
		out := Redact(c.in)
		if out == c.in || !contains(out, c.wantSub) {
			t.Fatalf("expected redaction in %q; got %q", c.in, out)
		}
	}
}

func TestRedact_Toggle(t *testing.T) {
	SetEnabled(false)
	in := "sk-abcdefghijklmnopqrstuvwxyzabcdef"
	if out := Redact(in); out != in {
		t.Fatalf("expected no redaction when disabled; got %q", out)
	}
	SetEnabled(true)
	if out := Redact(in); out == in {
		t.Fatalf("expected redaction when enabled; got %q", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestRedact_LuhnVerifier(t *testing.T) {
	// Valid Visa (test number): 4111 1111 1111 1111
	in := "my card 4111 1111 1111 1111 please"
	out := Redact(in)
	if out == in {
		t.Fatalf("expected redaction for valid Luhn card; got %q", out)
	}
	// Invalid number with same shape should not redact
	in2 := "my card 4111 1111 1111 1112 please"
	out2 := Redact(in2)
	if out2 != in2 {
		t.Fatalf("did not expect redaction for invalid Luhn card; got %q", out2)
	}
}
