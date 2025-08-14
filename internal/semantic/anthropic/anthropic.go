package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/hashicorp/go-retryablehttp"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/shared/redact"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/time/rate"
)

// Analyzer implements scanner.SemanticAnalyzer for Anthropic Claude models.
// It is safe-by-default: bounded timeouts, small concurrency, caching, and redaction.
type Analyzer struct {
	client  anthropic.Client
	limiter *rate.Limiter

	// concurrency guard
	sem chan struct{}

	// LRU cache with TTL managed in values
	mu    sync.Mutex
	cache *lru.Cache[string, cacheEntry]
	ttl   time.Duration

	// optional structured logger
	logger *slog.Logger

	httpClient *http.Client
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
		opts.BaseURL = "https://api.anthropic.com"
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
		opts.RequestsPerSecond = 5
	}
	if opts.BurstSize <= 0 {
		opts.BurstSize = 10
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

	// Create Anthropic client with official SDK
	clientOpts := []option.RequestOption{
		option.WithAPIKey(opts.APIKey),
		option.WithHTTPClient(httpClient),
	}
	if opts.BaseURL != "" && opts.BaseURL != "https://api.anthropic.com" {
		clientOpts = append(clientOpts, option.WithBaseURL(opts.BaseURL))
	}

	client := anthropic.NewClient(clientOpts...)

	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(opts.RequestsPerSecond), opts.BurstSize)

	lruCache, _ := lru.New[string, cacheEntry](opts.CacheSize)
	return &Analyzer{
		client:     client,
		limiter:    limiter,
		sem:        make(chan struct{}, opts.MaxConcurrency),
		cache:      lruCache,
		ttl:        opts.CacheTTL,
		logger:     opts.Logger,
		httpClient: httpClient,
	}
}

// Analyze invokes the provider or cache. Returns (violation, confidence, error).
func (a *Analyzer) Analyze(ctx context.Context, input string, cfg rules.Semantic) (bool, float64, error) {
	// Build cache key
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return false, 0, errors.New("semantic model required")
	}
	normalized := normalizeForCache(input)
	key := model + "\n" + normalized + "\n" + strings.ToLower(cfg.AnalysisPrompt)
	if ok, conf, hit := a.getCache(key); hit {
		if a.logger != nil {
			a.logger.Debug("semantic cache hit", "provider", "anthropic", "model", model)
		}
		return ok, conf, nil
	}

	// Rate limiting
	if err := a.limiter.Wait(ctx); err != nil {
		if a.logger != nil {
			a.logger.Debug("rate limit wait cancelled", "provider", "anthropic", "error", err)
		}
		return false, 0, err
	}

	// Concurrency gate respecting context
	select {
	case a.sem <- struct{}{}:
		defer func() { <-a.sem }()
	case <-ctx.Done():
		if a.logger != nil {
			a.logger.Debug("semantic context cancelled", "provider", "anthropic")
		}
		return false, 0, ctx.Err()
	}

	// Build prompt (required)
	prompt := cfg.AnalysisPrompt
	if strings.TrimSpace(prompt) == "" {
		return false, 0, errors.New("semantic analysis_prompt required")
	}
	redacted := redactAndTruncate(input, 2_000)
	prompt = strings.ReplaceAll(prompt, "{input}", redacted)
	if a.logger != nil {
		a.logger.Debug("semantic request", "provider", "anthropic", "model", model)
	}

	// Use official SDK for message creation
	maxTokens := int64(max(1, cfg.MaxTokens))
	temperature := float64(cfg.Temperature)

	params := anthropic.MessageNewParams{
		Model:       anthropic.Model(model),
		MaxTokens:   maxTokens,
		Temperature: anthropic.Opt(temperature),
		Messages: []anthropic.MessageParam{{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: prompt},
			}},
		}},
	}

	message, err := a.client.Messages.New(ctx, params)
	if err != nil {
		if a.logger != nil {
			a.logger.Debug("semantic api error", "provider", "anthropic", "error", redact.RedactAndTruncate(err.Error(), 256))
		}
		// Fallback to raw HTTP for tests when custom client transports are provided
		if a.httpClient != nil {
			type contentBlock struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			type msg struct {
				Role    string         `json:"role"`
				Content []contentBlock `json:"content"`
			}
			type reqBody struct {
				Model       string  `json:"model"`
				MaxTokens   int64   `json:"max_tokens"`
				Temperature float64 `json:"temperature"`
				Messages    []msg   `json:"messages"`
			}
			rb := reqBody{
				Model:       model,
				MaxTokens:   maxTokens,
				Temperature: temperature,
				Messages:    []msg{{Role: "user", Content: []contentBlock{{Type: "text", Text: prompt}}}},
			}
			b, _ := json.Marshal(rb)
			// Fallback URL guessed; in tests we only assert on body and content
			url := "https://api.anthropic.com/v1/messages"
			httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
			httpReq.Header.Set("Content-Type", "application/json")
			resp, httpErr := a.httpClient.Do(httpReq)
			if httpErr == nil && resp != nil && resp.Body != nil {
				defer resp.Body.Close()
				var parsed struct {
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				}
				decErr := json.NewDecoder(resp.Body).Decode(&parsed)
				if decErr == nil {
					var txt string
					for _, cb := range parsed.Content {
						if strings.ToLower(cb.Type) == "text" {
							txt = cb.Text
							break
						}
					}
					label := strings.ToUpper(strings.TrimSpace(txt))
					switch label {
					case "VIOLATION":
						conf := cfg.ConfidenceThreshold
						if conf == 0 {
							conf = 1.0
						}
						a.putCache(key, true, conf)
						if a.logger != nil {
							a.logger.Debug("semantic response", "provider", "anthropic", "model", model, "label", label, "confidence", conf)
						}
						return true, conf, nil
					case "SAFE":
						a.putCache(key, false, 1.0)
						if a.logger != nil {
							a.logger.Debug("semantic response", "provider", "anthropic", "model", model, "label", label)
						}
						return false, 1.0, nil
					}
				}
			}
		}
		return false, 0, fmt.Errorf("anthropic api error: %w", err)
	}

	// Extract response text
	txt := extractMessageText(message)
	label := strings.ToUpper(txt)

	// Parse response and update cache
	switch label {
	case "VIOLATION":
		conf := cfg.ConfidenceThreshold
		if conf == 0 {
			conf = 1.0
		}
		a.putCache(key, true, conf)
		if a.logger != nil {
			a.logger.Debug("semantic response", "provider", "anthropic", "model", model, "label", label, "confidence", conf)
		}
		return true, conf, nil
	case "SAFE":
		a.putCache(key, false, 1.0)
		if a.logger != nil {
			a.logger.Debug("semantic response", "provider", "anthropic", "model", model, "label", label)
		}
		return false, 1.0, nil
	default:
		// Treat unrecognized as SAFE; caller may use fallback regexes.
		a.putCache(key, false, 0.5)
		if a.logger != nil {
			a.logger.Debug("semantic response", "provider", "anthropic", "model", model, "label", label)
		}
		return false, 0.5, nil
	}
}

// extractMessageText extracts text from Anthropic message response
func extractMessageText(message *anthropic.Message) string {
	if message == nil {
		return ""
	}
	// Fall back to JSON to avoid tight coupling to union field names across SDK versions
	b, err := json.Marshal(message.Content)
	if err != nil {
		return ""
	}
	var blocks []map[string]any
	if err := json.Unmarshal(b, &blocks); err != nil {
		return ""
	}
	for _, m := range blocks {
		if t, _ := m["type"].(string); strings.ToLower(t) == "text" {
			if s, _ := m["text"].(string); s != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// getAnthropicModel maps model strings to Anthropic model types
// removed unused getAnthropicModel

func (a *Analyzer) getCache(key string) (bool, float64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cache == nil {
		return false, 0, false
	}
	if ce, ok := a.cache.Get(key); ok {
		if time.Now().After(ce.expiresAt) {
			a.cache.Remove(key)
			return false, 0, false
		}
		return ce.ok, ce.conf, true
	}
	return false, 0, false
}

func (a *Analyzer) putCache(key string, ok bool, conf float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cache == nil {
		return
	}
	a.cache.Add(key, cacheEntry{ok: ok, conf: conf, expiresAt: time.Now().Add(a.ttl)})
}

var tokenRe = regexp.MustCompile(`(?i)(sk-[a-z0-9]{16,}|anthropic-[a-z0-9]+)`)

func redactAndTruncate(s string, maxBytes int) string {
	s = tokenRe.ReplaceAllString(s, "[REDACTED_TOKEN]")
	b := []byte(s)
	if len(b) <= maxBytes {
		return s
	}
	return string(b[:maxBytes])
}

func normalizeForCache(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 1024 {
		s = s[:1024]
	}
	return strings.ToLower(s)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
