package deberta

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

    lru "github.com/hashicorp/golang-lru/v2"
    pmetrics "github.com/promptshield/promptshield/internal/observability/metrics"
    ckeys "github.com/promptshield/promptshield/internal/shared/contextkeys"
    "github.com/promptshield/promptshield/internal/rules"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Analyzer implements scanner.SemanticAnalyzer using a DeBERTa v3-based
// prompt-injection classifier (e.g., ProtectAI deberta-v3-base-prompt-injection-v2).
// It calls a configurable HTTP inference endpoint and converts the model output
// into a (flagged, confidence) pair. No rationales are produced or returned.
// All logs are redacted to avoid leaking model internals or justifications.
//
// Expected JSON response formats supported (first match wins):
// 1) HuggingFace-style: [{"label":"PROMPT_INJECTION","score":0.98}, ...]
// 2) Generic: {"risk_score":0.92, "label":"malicious"}
// 3) Text-classification: {"labels":["PROMPT_INJECTION","SAFE"], "scores":[0.98,0.02]}
//
// Configure via env or options:
//  - Endpoint (required): e.g. http://localhost:8089/infer or a HF Inference endpoint
//  - Timeout, cache, rate limits are kept small by default
//
// IMPORTANT: This analyzer NEVER returns chain-of-thought or rationales.
// It exposes only a boolean and numeric confidence. Logs redact details.

type Analyzer struct {
    endpoint string
    apiKey   string // optional (e.g., HF token); not required for local endpoints

    // HTTP
    http    *http.Client
    timeout time.Duration

    // cache (manual TTL)
    mu    sync.Mutex
    cache *lru.Cache[string, cacheEntry]
    ttl   time.Duration

    // logger (optional)
    logger *slog.Logger
}

type Options struct {
    Endpoint string
    APIKey   string // optional

    Timeout   time.Duration
    CacheSize int
    CacheTTL  time.Duration
    HTTP      *http.Client
    Logger    *slog.Logger
}

type cacheEntry struct {
    ok        bool
    conf      float64
    expiresAt time.Time
}

func New(opts Options) *Analyzer {
    if opts.Timeout <= 0 {
        opts.Timeout = 2500 * time.Millisecond
    }
    if opts.CacheSize <= 0 {
        opts.CacheSize = 1000
    }
    if opts.CacheTTL <= 0 {
        opts.CacheTTL = 15 * time.Minute
    }
    client := opts.HTTP
    if client == nil {
        client = &http.Client{Timeout: opts.Timeout, Transport: otelhttp.NewTransport(http.DefaultTransport)}
    }
    l, _ := lru.New[string, cacheEntry](opts.CacheSize)
    return &Analyzer{
        endpoint: strings.TrimSpace(opts.Endpoint),
        apiKey:   strings.TrimSpace(opts.APIKey),
        http:     client,
        timeout:  opts.Timeout,
        cache:    l,
        ttl:      opts.CacheTTL,
        logger:   opts.Logger,
    }
}

// Analyze implements scanner.SemanticAnalyzer.
// It posts the input to the configured endpoint and interprets the result.
func (a *Analyzer) Analyze(ctx context.Context, input string, cfg rules.Semantic) (bool, float64, error) {
    if a.endpoint == "" {
        return false, 0, fmt.Errorf("deberta endpoint not configured")
    }

    start := time.Now()
    provider := "protectai"
    model := "deberta"
    tenant := ""
    if v := ctx.Value(ckeys.TenantID); v != nil {
        tenant = strings.TrimSpace(fmt.Sprint(v))
    }

    // Normalize input to improve cache hits and reduce obvious obfuscation
    norm := normalizeForCache(input)

    // Estimate tokens using WordPiece tokenizer when available; fallback to basic splitting
    estTokens := estimateTokens(input)

    // Cache lookup
    if ok, conf, hit := a.getCache(norm); hit {
        if a.logger != nil {
            a.logger.Debug("deberta cache hit", "flagged", ok, "confidence", red(conf))
        }
        // Count as a successful request with near-zero latency
        if pmetrics.Enabled() {
            pmetrics.LLMLatency.WithLabelValues(provider, model).Observe(time.Since(start).Seconds())
            decision := "safe"
            if ok { decision = "flagged" }
            pmetrics.LLMRequestsTotal.WithLabelValues(provider, model, decision, tenant).Inc()
            if estTokens > 0 {
                pmetrics.TokensTotal.WithLabelValues(provider, model, "input", tenant).Add(float64(estTokens))
            }
        }
        return ok, conf, nil
    }

    // Build payload compatible with common text-classification servers
    // Primary preference: HF Inference style: {"inputs": "..."}
    body := map[string]any{
        "inputs": norm,
    }
    data, _ := json.Marshal(body)

    // Prepare request with context deadline
    to := a.timeout
    if dl, ok := ctx.Deadline(); ok {
        if left := time.Until(dl); left < to {
            to = left
        }
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(data))
    if err != nil {
        return false, 0, err
    }
    req.Header.Set("Content-Type", "application/json")
    if a.apiKey != "" {
        // Do not log or echo the API key
        req.Header.Set("Authorization", "Bearer "+a.apiKey)
    }

    // Execute
    resp, err := a.http.Do(req)
    if err != nil {
        if pmetrics.Enabled() {
            // Treat context deadline as timeout; otherwise generic request error
            retryable := "false"
            code := "request"
            if errors.Is(err, context.DeadlineExceeded) || (ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)) {
                code = "timeout"
                pmetrics.LLMTimeouts.WithLabelValues(provider, model).Inc()
                retryable = "true"
            }
            pmetrics.LLMErrors.WithLabelValues(provider, code, retryable).Inc()
        }
        return false, 0, err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        if pmetrics.Enabled() {
            pmetrics.LLMErrors.WithLabelValues(provider, fmt.Sprintf("http_%d", resp.StatusCode), "false").Inc()
        }
        return false, 0, fmt.Errorf("deberta inference http %d", resp.StatusCode)
    }

    // Parse any of the supported shapes
    var raw json.RawMessage
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
        if pmetrics.Enabled() {
            pmetrics.LLMErrors.WithLabelValues(provider, "decode", "false").Inc()
        }
        return false, 0, err
    }

    ok, conf := interpretOutput(raw)

    // Apply rule-level thresholding if present
    thr := cfg.ConfidenceThreshold
    if thr == 0 {
        thr = 0.70
    }
    flagged := conf >= thr && ok

    // Cache and redact logs
    a.putCache(norm, flagged, conf)
    if a.logger != nil {
        a.logger.Info("deberta classify", "flagged", flagged, "confidence", red(conf))
    }

    // Metrics: record latency and request outcome
    if pmetrics.Enabled() {
        pmetrics.LLMLatency.WithLabelValues(provider, model).Observe(time.Since(start).Seconds())
        decision := "safe"
        if flagged { decision = "flagged" }
        pmetrics.LLMRequestsTotal.WithLabelValues(provider, model, decision, tenant).Inc()
        if estTokens > 0 {
            pmetrics.TokensTotal.WithLabelValues(provider, model, "input", tenant).Add(float64(estTokens))
        }
    }

    return flagged, conf, nil
}

// interpretOutput attempts multiple known response formats and extracts
// (ok, confidence) where ok indicates the model saw injection/malicious intent.
func interpretOutput(raw json.RawMessage) (bool, float64) {
    // 1) HF: [{"label":"PROMPT_INJECTION","score":0.98}, ...]
    var arr []struct {
        Label string  `json:"label"`
        Score float64 `json:"score"`
    }
    if json.Unmarshal(raw, &arr) == nil && len(arr) > 0 {
        best := arr[0]
        for _, it := range arr {
            if it.Score > best.Score { best = it }
        }
        return labelIsMalicious(best.Label), best.Score
    }
    // 2) Generic: {"risk_score":0.92, "label":"malicious"}
    var obj map[string]any
    if json.Unmarshal(raw, &obj) == nil {
        label := strings.ToLower(fmt.Sprint(obj["label"]))
        if v, ok := obj["risk_score"].(float64); ok {
            return labelIsMalicious(label) || v >= 0.5, v
        }
        // 3) Text-classification: {labels:[], scores:[]}
        if ls, ok := obj["labels"].([]any); ok {
            if ss, ok2 := obj["scores"].([]any); ok2 && len(ls) == len(ss) && len(ls) > 0 {
                bestIdx := 0
                bestScore := toFloat(ss[0])
                for i := 1; i < len(ss); i++ {
                    if s := toFloat(ss[i]); s > bestScore { bestScore = s; bestIdx = i }
                }
                return labelIsMalicious(strings.ToLower(fmt.Sprint(ls[bestIdx]))), bestScore
            }
        }
    }
    // Fallback: unknown shape → SAFE with low confidence
    return false, 0.0
}

func toFloat(v any) float64 {
    switch t := v.(type) {
    case float64:
        return t
    case float32:
        return float64(t)
    case int:
        return float64(t)
    case json.Number:
        f, _ := t.Float64(); return f
    default:
        return 0
    }
}

var injLabelRe = regexp.MustCompile(`(?i)(injection|prompt_injection|malicious|attack|jailbreak)`) // best-effort

func labelIsMalicious(label string) bool {
    return injLabelRe.MatchString(strings.ToLower(strings.TrimSpace(label)))
}

// normalizeForCache trims, squeezes whitespace, lowercases, shortens long inputs.
func normalizeForCache(s string) string {
    s = strings.TrimSpace(s)
    s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
    if len(s) > 800 { s = s[:800] }
    return strings.ToLower(s)
}

func (a *Analyzer) getCache(key string) (bool, float64, bool) {
    a.mu.Lock(); defer a.mu.Unlock()
    if a.cache == nil { return false, 0, false }
    if e, ok := a.cache.Get(key); ok {
        if time.Now().Before(e.expiresAt) {
            return e.ok, e.conf, true
        }
        a.cache.Remove(key)
    }
    return false, 0, false
}

func (a *Analyzer) putCache(key string, ok bool, conf float64) {
    a.mu.Lock(); defer a.mu.Unlock()
    if a.cache == nil { return }
    a.cache.Add(key, cacheEntry{ ok: ok, conf: conf, expiresAt: time.Now().Add(a.ttl) })
}

// red returns a redacted version of floats for logs (no raw rationales or inputs)
func red(f float64) string { return fmt.Sprintf("%.3f", f) }

// estimateTokens tokenizes input using a minimal WordPiece-like approach.
// It expects a vocabulary embedded at build time; if unavailable, it falls back
// to a basic whitespace+punctuation split with subword prefixing rules.
// This is an approximation aimed at accounting; for exact parity, integrate
// your production tokenizer model.
// estimateTokens is implemented in tokenizer.go (prefers WordPiece tokenizer when available).
