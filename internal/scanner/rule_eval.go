package scanner

import (
	"regexp"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
)

type compiledRule struct {
	id              string
	message         string
	severity        string
	category        string
	level           int
	keywords        []string
	regexes         []*regexp.Regexp
	verifiers       []string
	semantic        *rules.Semantic
	fallbackRegexes []*regexp.Regexp
	when            *rules.Condition
	unless          *rules.Condition
	timeoutMs       int64
	// Rule matching logic (any=default, all=require all elements within a level)
	requireAll bool
	// Keyword matching options
	caseSensitive bool
	wholeWord     bool
	// Literal tokens extracted from regex patterns for gating
	literalTokens []string
	// Response mapping
	response *rules.Response
	// Fallback action
	fallbackAction string
	// Rule-level cache controls for semantic results
	cacheEnabled    bool
	cacheTTL        time.Duration
	cacheMaxEntries int
}

func compileRules(rulesList []rules.Rule, defaultRuleTimeoutMs int64, defaultCaseSensitive bool, defaultWholeWord bool) []compiledRule {
	var out []compiledRule
	for _, r := range rulesList {
		// Guard overly long regex patterns to prevent catastrophic complexity.
		const maxPatternLength = 1000
		cr := compiledRule{
			id:       r.ID,
			message:  coalesce(r.Name, "rule violation"),
			severity: coalesce(r.Severity, "INFO"),
			category: r.Category,
			level:    r.Level,
			when:     r.When,
			unless:   r.Unless,
		}
		if strings.EqualFold(r.Logic, "all") {
			cr.requireAll = true
		}
		if r.Timeout != "" {
			if d, err := time.ParseDuration(r.Timeout); err == nil {
				cr.timeoutMs = d.Milliseconds()
			}
		}
		if cr.timeoutMs == 0 && defaultRuleTimeoutMs > 0 {
			cr.timeoutMs = defaultRuleTimeoutMs
		}
		// Keyword options: rule-level overrides take precedence; otherwise use global defaults
		cr.caseSensitive = defaultCaseSensitive
		cr.wholeWord = defaultWholeWord
		if r.Options.CaseSensitive {
			cr.caseSensitive = true
		}
		if r.Options.WholeWord {
			cr.wholeWord = true
		}
		for _, kw := range r.Keywords {
			if cr.caseSensitive {
				cr.keywords = append(cr.keywords, kw)
			} else {
				cr.keywords = append(cr.keywords, strings.ToLower(kw))
			}
		}
		for _, pat := range r.Patterns {
			if len(pat.Regex) > maxPatternLength {
				continue
			}
			// Enforce complexity limits to avoid pathological regexes
			if err := rules.CheckRegexComplexity(pat.Regex, pat.Flags); err != nil {
				continue
			}
			// Touch coarse complexity scorer to keep symbol live (future tuning hook)
			_ = regexComplexityScore(pat.Regex)
			rx := compileRegex(pat.Regex, pat.Flags)
			if rx != nil {
				cr.regexes = append(cr.regexes, rx)
				cr.verifiers = append(cr.verifiers, strings.ToLower(strings.TrimSpace(pat.Verifier)))
			}
			// extract literal tokens for gating
			if toks := extractLiteralTokensFromRegex(pat.Regex); len(toks) > 0 {
				cr.literalTokens = append(cr.literalTokens, toks...)
			}
		}
		if r.Semantic != nil {
			// Shallow copy semantic config
			sc := *r.Semantic
			cr.semantic = &sc
		}
		if r.Fallback != nil {
			for _, p := range r.Fallback.Patterns {
				if rx := compileRegex(p.Regex, p.Flags); rx != nil {
					cr.fallbackRegexes = append(cr.fallbackRegexes, rx)
				}
			}
			cr.fallbackAction = r.Fallback.Action
		}
		if r.Response != nil {
			rr := *r.Response
			cr.response = &rr
		}
		if r.Cache != nil {
			cr.cacheEnabled = r.Cache.Enabled
			if r.Cache.MaxEntries > 0 {
				cr.cacheMaxEntries = r.Cache.MaxEntries
			}
			if ttl := strings.TrimSpace(r.Cache.TTL); ttl != "" {
				if d, err := time.ParseDuration(ttl); err == nil {
					cr.cacheTTL = d
				}
			}
		}
		out = append(out, cr)
	}
	return out
}

func coalesce(v string, def string) string {
	if v == "" {
		return def
	}
	return v
}

func compileRegex(expr string, flags []string) *regexp.Regexp {
	if expr == "" {
		return nil
	}
	if globalMaxPatternLength > 0 && len(expr) > globalMaxPatternLength {
		return nil
	}
	prefix := ""
	for _, f := range flags {
		switch strings.ToLower(f) {
		case "ignorecase", "i":
			prefix = "(?i)" + prefix
		case "multiline", "m":
			prefix = "(?m)" + prefix
		}
	}
	// Global cache lookup
	key := cacheKey(prefix+expr, flags)
	globalRegexCache.Lock()
	if globalRegexCache.cache != nil {
		if rx, ok := globalRegexCache.cache.Get(key); ok {
			globalRegexCache.Unlock()
			return rx
		}
	}
	globalRegexCache.Unlock()
	rx, err := regexp.Compile(prefix + expr)
	if err != nil {
		return nil
	}
	globalRegexCache.Lock()
	if globalRegexCache.cache != nil {
		globalRegexCache.cache.Add(key, rx)
	}
	globalRegexCache.Unlock()
	return rx
}

// contextMatches evaluates optional when/unless conditions against a runtime context map.
// If both are nil, returns true. If when is set, all its keys must match; if unless
// is set and matches, the rule is skipped.
func contextMatches(when *rules.Condition, unless *rules.Condition, ctx map[string]string) bool {
	if unless != nil && conditionMatches(unless, ctx) {
		return false
	}
	if when == nil {
		return true
	}
	return conditionMatches(when, ctx)
}

func conditionMatches(c *rules.Condition, ctx map[string]string) bool {
	if c == nil {
		return true
	}
	if len(c.Match) == 0 {
		return true
	}
	if ctx == nil {
		return false
	}
	for k, vals := range c.Match {
		got, ok := ctx[k]
		if !ok {
			return false
		}
		match := false
		for _, v := range vals {
			if strings.EqualFold(got, v) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}
