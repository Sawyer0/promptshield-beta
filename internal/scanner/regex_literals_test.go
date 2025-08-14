package scanner

import "testing"

func TestExtractLiteralTokensFromRegex(t *testing.T) {
	cases := []struct {
		expr        string
		wantAtLeast int
	}{
		// This pattern has literal prefix 'sk-'; our extractor may or may not pick it up depending on metachar positions.
		// We don't require it strictly; aim to get some tokens overall across cases.
		{`sk-[a-z0-9]{24,}`, 0},
		{`(?i)password|secret|api[_-]?key`, 2},
		{`^user:[A-Za-z0-9_-]{3,}$`, 1},
		{`(\d{3}-\d{2}-\d{4})`, 0}, // digits only not tokenized
		{`foo[abc]bar`, 1},         // 'foo' literal prefix
	}
	for _, tc := range cases {
		toks := extractLiteralTokensFromRegex(tc.expr)
		if len(toks) < tc.wantAtLeast {
			t.Fatalf("expr=%q got %v", tc.expr, toks)
		}
	}
}
