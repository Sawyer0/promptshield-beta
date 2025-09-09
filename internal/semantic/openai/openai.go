package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"unicode"

	"github.com/hashicorp/go-retryablehttp"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/promptshield/promptshield/internal/rules"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/time/rate"
)

// Analyzer implements scanner.SemanticAnalyzer using OpenAI's omni-moderation API.
// It is safe-by-default: bounded timeouts, small concurrency, caching, and rate limiting.
type Analyzer struct {
	client  openai.Client
	limiter *rate.Limiter

	// raw HTTP for features not yet supported in SDK (e.g., multimodal moderation)
	httpClient *http.Client
	apiKey     string
	baseURL    string

	// concurrency guard
	sem chan struct{}

	// LRU cache with TTL managed in values
	mu    sync.Mutex
	cache *lru.Cache[string, cacheEntry]
	ttl   time.Duration

	// optional structured logger
	logger *slog.Logger
}

type cacheEntry struct {
	ok        bool
	conf      float64
	expiresAt time.Time
}

type Options struct {
	APIKey         string
	BaseURL        string
	MaxConcurrency int
	CacheSize      int
	CacheTTL       time.Duration
	HTTPClient     *http.Client
	Logger         *slog.Logger
	// Rate limiting
	RequestsPerSecond float64
	BurstSize         int
}

func New(opts Options) *Analyzer {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.openai.com/v1"
	}
	if opts.MaxConcurrency <= 0 {
		opts.MaxConcurrency = 2
	}
	if opts.CacheSize <= 0 {
		opts.CacheSize = 1000
	}
	if opts.CacheTTL <= 0 {
		opts.CacheTTL = 15 * time.Minute
	}
	if opts.RequestsPerSecond <= 0 {
		opts.RequestsPerSecond = 10
	}
	if opts.BurstSize <= 0 {
		opts.BurstSize = 20
	}

	// Create retryable HTTP client with OTel instrumentation
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 10 * time.Second
	retryClient.Backoff = retryablehttp.DefaultBackoff
	retryClient.CheckRetry = retryablehttp.DefaultRetryPolicy
	retryClient.Logger = nil // Disable retryablehttp's default logging

	// Wrap with OpenTelemetry instrumentation
	httpClient := retryClient.StandardClient()
	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)
	httpClient.Timeout = 30 * time.Second

	// Override with custom client if provided
	if opts.HTTPClient != nil {
		httpClient = opts.HTTPClient
	}

	// Create OpenAI client with official SDK
	clientOpts := []option.RequestOption{
		option.WithAPIKey(opts.APIKey),
		option.WithHTTPClient(httpClient),
	}
	if opts.BaseURL != "" && opts.BaseURL != "https://api.openai.com/v1" {
		clientOpts = append(clientOpts, option.WithBaseURL(opts.BaseURL))
	}

	client := openai.NewClient(clientOpts...)

	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(opts.RequestsPerSecond), opts.BurstSize)

	// LRU with manual TTL
	lruCache, _ := lru.New[string, cacheEntry](opts.CacheSize)
	return &Analyzer{
		client:     client,
		limiter:    limiter,
		sem:        make(chan struct{}, opts.MaxConcurrency),
		cache:      lruCache,
		ttl:        opts.CacheTTL,
		logger:     opts.Logger,
		httpClient: httpClient,
		apiKey:     opts.APIKey,
		baseURL:    opts.BaseURL,
	}
}

// ModerationInput represents input for moderation, supporting both text and images
type ModerationInput struct {
	Text     string
	ImageURL string
}

// ModerationResult contains the analyzed results from omni-moderation
type ModerationResult struct {
	Flagged    bool
	Categories map[string]float64
	Decision   string
	Confidence float64
	Reason     string
}

// AnalyzeWithModeration uses OpenAI's omni-moderation-latest model for Level 3 semantic analysis
// This is FREE and supports multimodal (text + image) content
func (a *Analyzer) AnalyzeWithModeration(ctx context.Context, input ModerationInput, cfg rules.Semantic) (*ModerationResult, error) {
	// Pre-process content to reduce common evasion tactics and improve cache hits
	processed := sanitizeForModeration(input.Text)

	// Build cache key
	cacheKey := fmt.Sprintf("mod:%s:%s", normalizeForCache(processed), input.ImageURL)

	// Check cache first
	if cached, conf, found := a.getCache(cacheKey); found {
		if a.logger != nil {
			a.logger.Debug("moderation cache hit", "flagged", cached, "confidence", conf)
		}
		return &ModerationResult{
			Flagged:    cached,
			Confidence: conf,
			Decision:   getDecision(cached),
		}, nil
	}

	// Rate limiting
	if err := a.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Concurrency limiting
	select {
	case a.sem <- struct{}{}:
		defer func() { <-a.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// If image provided, use raw HTTP request to multimodal Moderations API
	if input.ImageURL != "" {
		mmResult, err := a.analyzeMultimodal(ctx, processed, input.ImageURL)
		if err != nil {
			return nil, err
		}
		// Cache and log for multimodal path to keep behavior consistent
		a.putCache(cacheKey, mmResult.Flagged, mmResult.Confidence)
		if a.logger != nil {
			a.logger.Info("moderation analysis complete",
				"flagged", mmResult.Flagged,
				"confidence", mmResult.Confidence,
				"category", mmResult.Reason,
				"multimodal", true,
			)
		}
		return mmResult, nil
	}

	// Text-only path via official SDK
	moderationReq := openai.ModerationNewParams{
		Input: openai.ModerationNewParamsInputUnion{OfString: openai.String(processed)},
		Model: openai.ModerationModelOmniModerationLatest,
	}
	resp, err := a.client.Moderations.New(ctx, moderationReq)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("moderation api error", "error", err)
		}
		return nil, fmt.Errorf("moderation api error: %w", err)
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no moderation results returned")
	}
	result := resp.Results[0]

	// Map categories to scores
	categories := make(map[string]float64)

	// Map available categories from the SDK
	if result.Categories.Harassment {
		categories["harassment"] = result.CategoryScores.Harassment
	}
	if result.Categories.HarassmentThreatening {
		categories["harassment_threatening"] = result.CategoryScores.HarassmentThreatening
	}
	if result.Categories.Hate {
		categories["hate"] = result.CategoryScores.Hate
	}
	if result.Categories.HateThreatening {
		categories["hate_threatening"] = result.CategoryScores.HateThreatening
	}
	if result.Categories.SelfHarm {
		categories["self_harm"] = result.CategoryScores.SelfHarm
	}
	if result.Categories.SelfHarmIntent {
		categories["self_harm_intent"] = result.CategoryScores.SelfHarmIntent
	}
	if result.Categories.SelfHarmInstructions {
		categories["self_harm_instructions"] = result.CategoryScores.SelfHarmInstructions
	}
	if result.Categories.Sexual {
		categories["sexual"] = result.CategoryScores.Sexual
	}
	if result.Categories.SexualMinors {
		categories["sexual_minors"] = result.CategoryScores.SexualMinors
	}
	if result.Categories.Violence {
		categories["violence"] = result.CategoryScores.Violence
	}
	if result.Categories.ViolenceGraphic {
		categories["violence_graphic"] = result.CategoryScores.ViolenceGraphic
	}

	// Check for Illicit categories (new in omni-moderation)
	// Note: These fields may not be in SDK v1.12.0 yet
	// Attempt to access them if available
	if result.Categories.Illicit {
		categories["illicit"] = result.CategoryScores.Illicit
	}
	if result.Categories.IllicitViolent {
		categories["illicit_violent"] = result.CategoryScores.IllicitViolent
	}

	// If the rule restricts categories, filter now
	if len(cfg.AllowedCategories) > 0 {
		allowed := make(map[string]struct{}, len(cfg.AllowedCategories))
		for _, c := range cfg.AllowedCategories {
			// normalize slashes to underscores to match keys above
			k := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(c)), "/", "_")
			allowed[k] = struct{}{}
		}
		filtered := make(map[string]float64)
		for k, v := range categories {
			if _, ok := allowed[k]; ok {
				filtered[k] = v
			}
		}
		categories = filtered
	}

	// Determine if content is flagged based on thresholds
	flagged := result.Flagged
	maxScore := 0.0
	highestCategory := ""

	// Check for prompt injection patterns (illicit category often catches these)
	if illicitScore, ok := categories["illicit"]; ok && illicitScore > 0.5 {
		highestCategory = "prompt_injection"
		maxScore = illicitScore
	}

	// Find highest scoring category
	for cat, score := range categories {
		if score > maxScore {
			maxScore = score
			highestCategory = cat
		}
	}

	// Override flagged if any category exceeds threshold
	threshold := cfg.ConfidenceThreshold
	if threshold == 0 {
		threshold = 0.7 // Default threshold
	}
	// If the text is likely non‑English, allow configurable adjustment to compensate
	if !isLikelyEnglish(processed) {
		// Nudge threshold down slightly to compensate for multilingual variance
		// but never below 0.5
		if threshold > 0.55 {
			threshold = threshold - 0.1
		}
	}

	if maxScore >= threshold {
		flagged = true
	}

	// Determine reason for flagging
	reason := ""
	if flagged {
		if highestCategory == "prompt_injection" {
			reason = "Potential prompt injection detected"
		} else if strings.Contains(highestCategory, "illicit") {
			reason = "Instructions for wrongdoing detected"
		} else if strings.Contains(highestCategory, "harassment") {
			reason = "Harassment content detected"
		} else if strings.Contains(highestCategory, "violence") {
			reason = "Violence content detected"
		} else if highestCategory != "" {
			reason = fmt.Sprintf("Content violates %s policy", strings.ReplaceAll(highestCategory, "_", " "))
		} else {
			reason = "Content flagged by moderation"
		}
	}

	modResult := &ModerationResult{
		Flagged:    flagged,
		Categories: categories,
		Decision:   getDecision(flagged),
		Confidence: maxScore,
		Reason:     reason,
	}

	// Cache the result
	a.putCache(cacheKey, flagged, maxScore)

	if a.logger != nil {
		a.logger.Info("moderation analysis complete",
			"flagged", flagged,
			"confidence", maxScore,
			"category", highestCategory,
			"multimodal", input.ImageURL != "",
		)
	}

	return modResult, nil
}

// analyzeMultimodal posts a mixed text+image payload to /moderations and parses the response.
func (a *Analyzer) analyzeMultimodal(ctx context.Context, text string, imageURL string) (*ModerationResult, error) {
	endpoint := strings.TrimRight(a.baseURL, "/") + "/moderations"
	body := map[string]any{
		"model": "omni-moderation-latest",
		"input": []any{
			map[string]any{"type": "text", "text": text},
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": imageURL}},
		},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("moderation api error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("moderation api http %d", resp.StatusCode)
	}
	var parsed struct {
		Results []struct {
			Flagged        bool               `json:"flagged"`
			Categories     map[string]bool    `json:"categories"`
			CategoryScores map[string]float64 `json:"category_scores"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Results) == 0 {
		return nil, fmt.Errorf("no moderation results returned")
	}
	r := parsed.Results[0]
	// Convert to ModerationResult shape
	maxScore := 0.0
	highest := ""
	for k, v := range r.CategoryScores {
		if v > maxScore {
			maxScore = v
			highest = k
		}
	}
	reason := ""
	if r.Flagged {
		if highest != "" {
			reason = "Content violates " + strings.ReplaceAll(highest, "_", " ")
		} else {
			reason = "Content flagged by moderation"
		}
	}
	return &ModerationResult{
		Flagged:    r.Flagged,
		Categories: r.CategoryScores,
		Decision:   getDecision(r.Flagged),
		Confidence: maxScore,
		Reason:     reason,
	}, nil
}

// AnalyzeModeration is the main implementation using OpenAI's FREE moderation API
func (a *Analyzer) AnalyzeModeration(ctx context.Context, input string, cfg rules.Semantic) (bool, float64, error) {
	// Always use omni-moderation-latest for best results
	// Override any model specified in config since we're only using moderation API
	cfg.Model = "omni-moderation-latest"

	result, err := a.AnalyzeWithModeration(ctx, ModerationInput{Text: input}, cfg)
	if err != nil {
		return false, 0, err
	}

	return result.Flagged, result.Confidence, nil
}

func getDecision(flagged bool) string {
	if flagged {
		return "block"
	}
	return "allow"
}

// Analyze uses the OpenAI omni-moderation API for semantic analysis
// This API is FREE and doesn't require analysis prompts
func (a *Analyzer) Analyze(ctx context.Context, input string, cfg rules.Semantic) (bool, float64, error) {
	return a.AnalyzeModeration(ctx, input, cfg)
}

func (a *Analyzer) getCache(key string) (bool, float64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cache == nil {
		return false, 0, false
	}
	if e, ok := a.cache.Get(key); ok {
		if time.Now().Before(e.expiresAt) {
			return e.ok, e.conf, true
		}
		a.cache.Remove(key)
	}
	return false, 0, false
}

func (a *Analyzer) putCache(key string, ok bool, conf float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cache == nil {
		return
	}
	a.cache.Add(key, cacheEntry{
		ok:        ok,
		conf:      conf,
		expiresAt: time.Now().Add(a.ttl),
	})
}

// normalizeForCache removes excess whitespace and normalizes input for caching
func normalizeForCache(s string) string {
	// Remove leading/trailing whitespace
	s = strings.TrimSpace(s)
	// Normalize internal whitespace
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	// Truncate if too long
	if len(s) > 500 {
		s = s[:500]
	}
	return strings.ToLower(s)
}

// sanitizeForModeration applies normalization intended to counter trivial obfuscations
// while preserving meaning for moderation (lowercasing, Unicode NFC, width folding,
// removal of zero‑width characters, and basic leetspeak deobfuscation).
func sanitizeForModeration(s string) string {
	if s == "" {
		return s
	}
	// Normalize Unicode to NFC and fold width (e.g., full‑width characters)
	// Remove zero‑width and control characters
	removeZWS := func(r rune) rune {
		if r == 0x200B || r == 0x200C || r == 0x200D || r == 0xFEFF {
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}
	b := &strings.Builder{}
	for _, r := range s {
		if rr := removeZWS(r); rr != -1 {
			b.WriteRune(rr)
		}
	}
	clean := b.String()
	// Unicode canonical composition
	clean, _, _ = transform.String(norm.NFC, clean)
	clean = strings.ToLower(clean)
	// Basic deobfuscation for common leetspeak
	replacer := strings.NewReplacer(
		"0", "o",
		"1", "i",
		"3", "e",
		"4", "a",
		"5", "s",
		"7", "t",
		"@", "a",
		"$", "s",
	)
	clean = replacer.Replace(clean)
	return clean
}

// isLikelyEnglish returns true if the majority of letters belong to the Latin script.
func isLikelyEnglish(s string) bool {
	letters := 0
	nonLatin := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if !unicode.Is(unicode.Latin, r) {
				nonLatin++
			}
		}
	}
	if letters == 0 {
		return true
	}
	return float64(nonLatin) <= 0.3*float64(letters)
}
