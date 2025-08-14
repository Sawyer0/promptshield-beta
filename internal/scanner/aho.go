package scanner

import (
	"sort"
	"strings"

	"github.com/cloudflare/ahocorasick"
)

// Aho wraps the Cloudflare Aho-Corasick matcher and preserves the local API.
type Aho struct {
	patterns []string
	matcher  *ahocorasick.Matcher
}

// Match represents a pattern occurrence at position idx (start index in haystack).
type Match struct {
	PatternIndex int
	Index        int
}

// NewAho builds a matcher for non-empty patterns. Duplicates are coalesced.
func NewAho(patterns []string) *Aho {
	seen := make(map[string]struct{}, len(patterns))
	uniq := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		uniq = append(uniq, p)
	}
	return &Aho{
		patterns: uniq,
		matcher:  ahocorasick.NewStringMatcher(uniq),
	}
}

// FindAll returns all matches with start indices. Order is ascending by index.
func (a *Aho) FindAll(hay []byte) []Match {
	if a == nil || len(a.patterns) == 0 {
		return nil
	}
	ids := a.matcher.Match(hay)
	if len(ids) == 0 {
		return nil
	}
	text := string(hay)
	matches := make([]Match, 0, len(ids))
	for _, pid := range ids {
		pat := a.patterns[pid]
		// enumerate all occurrences for this pattern
		for pos := 0; ; {
			idx := strings.Index(text[pos:], pat)
			if idx < 0 {
				break
			}
			start := pos + idx
			matches = append(matches, Match{PatternIndex: pid, Index: start})
			pos = start + 1
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Index < matches[j].Index })
	return matches
}

// Patterns returns the pattern list.
func (a *Aho) Patterns() []string { return a.patterns }
