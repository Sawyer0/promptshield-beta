package scanner

import (
    "context"
    "os"
    "strings"
    "time"

    lru "github.com/hashicorp/golang-lru/v2"
    "github.com/promptshield/promptshield/internal/rules"
)

// SemanticAnalyzer defines the minimal interface for level-3 rule evaluation.
// Implementations should be pure and side-effect free; the scanner coordinates timeouts.
type SemanticAnalyzer interface {
	Analyze(ctx context.Context, input string, cfg rules.Semantic) (bool, float64, error)
}

// semanticCacheEntry stores decision and confidence with expiry.
type semanticCacheEntry struct {
	ok        bool
	conf      float64
	expiresAt time.Time
}

// evaluateSemantic applies a semantic analyzer with proper timeouts and fallback regexes.
// Returns (matched, confidence). Confidence is meaningful when matched via semantic L3.
func (s *Scanner) evaluateSemantic(line string, cr compiledRule, matchCol *int) (bool, float64) {
	if cr.semantic == nil || s.semantic == nil {
		return false, 0
	}
	// License/feature gating is enforced at runtime layers (gateway/CLI).
	// The core scanner remains feature-agnostic to preserve testability and OSS usability.
    // Parent context for request-scoped values (tenant, tracing)
    parent := context.Background()
    if s.baseCtx != nil { parent = s.baseCtx }
    // Derive timeout
    var timeout time.Duration
	if cr.timeoutMs > 0 {
		timeout = time.Duration(cr.timeoutMs) * time.Millisecond
	} else if s.fileTimeout > 0 {
		timeout = s.fileTimeout / 10
		if timeout <= 0 {
			timeout = 2 * time.Second
		}
	} else {
		timeout = 2 * time.Second
	}
	// Optional per-rule cache (respects rule.cache settings)
	var cachedOk bool
	if cr.semantic != nil && s.semanticCaches != nil && cr.cacheEnabled {
		if c, ok := s.semanticCaches[cr.id]; ok && c != nil {
			key := line // simple key; upstream analyzers also normalize and include prompt
				if ce, hit := c.Get(key); hit {
					if time.Now().Before(ce.expiresAt) {
						cachedOk = ce.ok
						conf := ce.conf
					if s.logger != nil {
						s.logger.Debug("semantic cache hit (rule)", "rule", cr.id)
					}
						if cachedOk {
							return true, conf
						}
				} else {
					c.Remove(key)
				}
			}
		}
	}
	// Global policy: require cache hit only (skip remote L3) when enabled
	reqCache := strings.ToLower(strings.TrimSpace(os.Getenv("PS_SEMANTIC_REQUIRE_CACHE_HIT")))
	if reqCache == "1" || reqCache == "true" || reqCache == "yes" {
		return false, 0
	}
    ctx, cancel := context.WithTimeout(parent, timeout)
	ok, conf, err := s.semantic.Analyze(ctx, line, *cr.semantic)
	cancel()
	if err == nil && ok {
		// Populate cache if enabled
		if cr.semantic != nil && s.semanticCaches != nil && cr.cacheEnabled {
			if _, ok := s.semanticCaches[cr.id]; !ok {
				cap := 256
				if cr.cacheMaxEntries > 0 {
					cap = cr.cacheMaxEntries
				}
				if cache, e := lru.New[string, semanticCacheEntry](cap); e == nil {
					s.semanticCaches[cr.id] = cache
				}
			}
			if cache := s.semanticCaches[cr.id]; cache != nil {
				ttl := 15 * time.Minute
				if cr.cacheTTL > 0 {
					ttl = cr.cacheTTL
				}
				cache.Add(line, semanticCacheEntry{ok: ok, conf: conf, expiresAt: time.Now().Add(ttl)})
			}
		}
		if s.logger != nil {
			s.logger.Debug("semantic match", "rule", cr.id, "confidence", conf)
		}
		return true, conf
	}
	// If API error and rule allows fallback_on_error, continue to fallback
	if err != nil {
		if s.logger != nil {
			s.logger.Debug("semantic error", "rule", cr.id, "err", err.Error())
		}
		if !cr.semantic.FallbackOnError {
			return false, 0
		}
	}
	// Always evaluate fallback regexes even on SAFE to catch heuristic patterns
	for _, rx := range cr.fallbackRegexes {
		if loc := rx.FindStringIndex(line); loc != nil {
			if matchCol != nil {
				*matchCol = loc[0] + 1
			}
			if s.logger != nil {
				s.logger.Debug("semantic fallback matched", "rule", cr.id, "action", cr.fallbackAction)
			}
			return true, 0
		}
	}
	if s.logger != nil {
		s.logger.Debug("semantic no match", "rule", cr.id, "err", err != nil)
	}
	return false, 0
}

// SetSemanticAnalyzer injects a semantic analyzer implementation.
func (s *Scanner) SetSemanticAnalyzer(a SemanticAnalyzer) { s.semantic = a }
