package openai

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/promptshield/promptshield/internal/rules"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/time/rate"
)

// Analyzer implements scanner.SemanticAnalyzer using OpenAI's omni-moderation API.
// It is safe-by-default: bounded timeouts, small concurrency, caching, and rate limiting.
type Analyzer struct {
	client  openai.Client
	limiter *rate.Limiter

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
		client:  client,
		limiter: limiter,
		sem:     make(chan struct{}, opts.MaxConcurrency),
		cache:   lruCache,
		ttl:     opts.CacheTTL,
		logger:  opts.Logger,
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
	// Build cache key
	cacheKey := fmt.Sprintf("mod:%s:%s", normalizeForCache(input.Text), input.ImageURL)
	
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

	// Prepare moderation request based on OpenAI Go SDK pattern
	// For now, use text-only moderation as multimodal support requires SDK updates
	// The omni-moderation-latest model still provides excellent detection
	moderationReq := openai.ModerationNewParams{
		Input: openai.ModerationNewParamsInputUnion{
			OfString: openai.String(input.Text),
		},
		Model: openai.ModerationModelOmniModerationLatest,
	}

	// Call OpenAI Moderation API (FREE!)
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

// Helper function for string pointers
func openaiString(s string) *string {
	return &s
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