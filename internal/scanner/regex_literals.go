package scanner

import (
	"regexp"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

// globalRegexCache caches compiled regexes by pattern with flags using a bounded LRU.
var globalRegexCache = struct {
	sync.Mutex
	cache *lru.Cache[string, *regexp.Regexp]
}{
	// Initialized in init(); keep zero-value until then
}

const defaultRegexLRUSize = 4096

func init() {
	// Best-effort init; size is constant and valid so error is nil.
	if c, err := lru.New[string, *regexp.Regexp](defaultRegexLRUSize); err == nil {
		globalRegexCache.cache = c
	}
}

func cacheKey(expr string, flags []string) string {
	if len(flags) == 0 {
		return expr
	}
	// deterministic key
	b := strings.Builder{}
	b.WriteString(expr)
	b.WriteString("::")
	// flags are simple small slices; order from source is stable; we can sort if needed
	for _, f := range flags {
		b.WriteString(strings.ToLower(f))
		b.WriteByte(',')
	}
	return b.String()
}

// extractLiteralTokensFromRegex extracts conservative literal tokens (lowercase) from a regex string.
// It returns contiguous runs of [A-Za-z0-9_-] of length >= 4 found outside of character classes and not following escapes.
var minTokenLen = 3
var globalMaxPatternLength = 0

func extractLiteralTokensFromRegex(expr string) []string {
	tokens := make([]string, 0)
	inClass := false
	escaped := false
	start := -1
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if escaped {
			// escaped char is treated as non-literal boundary
			if start >= 0 {
				if i-start >= minTokenLen {
					tokens = append(tokens, toLower(expr[start:i]))
				}
				start = -1
			}
			escaped = false
			continue
		}
		if c == '\\' { // escape
			escaped = true
			continue
		}
		if c == '[' {
			inClass = true
			if start >= 0 {
				if i-start >= 4 {
					tokens = append(tokens, toLower(expr[start:i]))
				}
				start = -1
			}
			continue
		}
		if c == ']' && inClass {
			inClass = false
			continue
		}
		if inClass {
			continue
		}
		// metacharacters terminate token runs, but keep a short literal prefix like 'sk-'
		switch c {
		case '^', '$', '.', '*', '+', '?', '{', '}', '(', ')', '|':
			if start >= 0 {
				if i-start >= minTokenLen {
					tokens = append(tokens, toLower(expr[start:i]))
				}
				start = -1
			}
			continue
		}
		// consider token chars and literal hyphen patterns like 'sk-'
		if isTokenChar(c) {
			if start < 0 {
				start = i
			}
		} else {
			// allow dash continuation if current char is '-' and a token run is ongoing
			if c == '-' && start >= 0 {
				// keep run going to include '-'
				continue
			}
			if start >= 0 {
				if i-start >= minTokenLen {
					tokens = append(tokens, toLower(expr[start:i]))
				}
				start = -1
			}
		}
	}
	if start >= 0 {
		if len(expr)-start >= minTokenLen {
			tokens = append(tokens, toLower(expr[start:]))
		}
	}
	return dedupe(tokens)
}

func isTokenChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

func toLower(s string) string {
	// fast ASCII lower
	bs := []byte(s)
	for i := range bs {
		if bs[i] >= 'A' && bs[i] <= 'Z' {
			bs[i] = bs[i] + 32
		}
	}
	return string(bs)
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return in
	}
	m := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := m[s]; !ok {
			m[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
