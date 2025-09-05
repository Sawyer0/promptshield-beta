package api

import "strings"

// matchToolByQuery performs a minimal capability/tag query match against a tool-like object.
// Query syntax supported: words combined with AND/OR/NOT and simple qualifiers like "side:reversible".
// The tool argument must have fields: CapabilityTags []string, DataDomains []string, SideEffect string, AuthScope string.
func matchToolByQuery(tool interface {
	GetCapabilityTags() []string
	GetDataDomains() []string
	GetSideEffect() string
	GetAuthScope() string
}, query string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return true
	}
	// Normalize tokens
	tokens := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(q, "(", " "), ")", " "))
	// Build a simple set of attributes
	attrs := map[string]bool{}
	for _, c := range tool.GetCapabilityTags() {
		if s := strings.ToLower(strings.TrimSpace(c)); s != "" {
			attrs[s] = true
		}
	}
	for _, d := range tool.GetDataDomains() {
		if s := strings.ToLower(strings.TrimSpace(d)); s != "" {
			attrs[s] = true
		}
	}
	if s := strings.ToLower(strings.TrimSpace(tool.GetSideEffect())); s != "" {
		attrs["side:"+s] = true
	}
	if s := strings.ToLower(strings.TrimSpace(tool.GetAuthScope())); s != "" {
		attrs["auth:"+s] = true
	}

	// Evaluate simple left-to-right with AND precedence over OR; NOT is unary
	evalTerm := func(t string) bool {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" {
			return true
		}
		// Support key:value forms like side:reversible
		if strings.Contains(t, ":") {
			return attrs[t]
		}
		// Plain term must be present in capability/data domain
		return attrs[t]
	}

	// Split by OR
	orParts := splitOn(tokens, "or")
	for _, part := range orParts {
		// Within each OR segment, all AND terms must pass
		passed := true
		i := 0
		for i < len(part) {
			t := strings.ToLower(part[i])
			neg := false
			if t == "not" {
				neg = true
				i++
				if i >= len(part) {
					break
				}
				t = strings.ToLower(part[i])
			}
			if t == "and" {
				i++
				continue
			}
			ok := evalTerm(t)
			if neg {
				ok = !ok
			}
			if !ok {
				passed = false
				break
			}
			i++
		}
		if passed {
			return true
		}
	}
	return false
}

func splitOn(tokens []string, sep string) [][]string {
	var parts [][]string
	cur := []string{}
	for _, t := range tokens {
		if strings.EqualFold(t, sep) {
			if len(cur) > 0 {
				parts = append(parts, cur)
				cur = []string{}
			}
		} else {
			cur = append(cur, t)
		}
	}
	if len(cur) > 0 {
		parts = append(parts, cur)
	}
	return parts
}

// pgToolLike adapter implements the minimal interface for matchToolByQuery
func (t *pgToolLike) GetCapabilityTags() []string { return t.CapabilityTags }
func (t *pgToolLike) GetDataDomains() []string    { return t.DataDomains }
func (t *pgToolLike) GetSideEffect() string       { return t.SideEffect }
func (t *pgToolLike) GetAuthScope() string        { return t.AuthScope }
