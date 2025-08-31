package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
)

// KeyResolver returns (apiKey, baseURL, error) for the given request context and rule config.
type KeyResolver func(ctx context.Context, cfg rules.Semantic) (string, string, error)

type Analyzer struct {
	resolve KeyResolver
	timeout time.Duration
	http    *http.Client
}

func New(resolver KeyResolver) *Analyzer {
	return &Analyzer{resolve: resolver, timeout: 5 * time.Second, http: &http.Client{Timeout: 10 * time.Second}}
}

// Analyze calls a BYOK provider (OpenAI for first iteration) and expects a strict JSON response
// {"flagged": boolean, "confidence": number, "categories": [string]}
func (a *Analyzer) Analyze(ctx context.Context, input string, cfg rules.Semantic) (bool, float64, error) {
	if a.resolve == nil {
		return false, 0, errors.New("custom analyzer missing resolver")
	}
	apiKey, baseURL, err := a.resolve(ctx, cfg)
	if err != nil {
		return false, 0, err
	}
	if strings.TrimSpace(apiKey) == "" {
		return false, 0, errors.New("no api key")
	}

	if baseURL = strings.TrimSpace(baseURL); baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// System/user messages
	prompt := cfg.AnalysisPrompt
	if strings.TrimSpace(prompt) == "" {
		prompt = `You are a strict classifier. Return ONLY compact JSON with this shape: {"flagged": boolean, "confidence": number, "categories": string[]}`
	}
	userMsg := fmt.Sprintf("Input:\n%s\n\nReturn only the JSON.", input)

	// Build request body
	body := map[string]any{
		"model": valueOr(cfg.Model, "gpt-4o-mini"),
		"messages": []map[string]string{
			{"role": "system", "content": prompt},
			{"role": "user", "content": userMsg},
		},
	}
	if cfg.MaxTokens > 0 {
		body["max_tokens"] = cfg.MaxTokens
	}
	if cfg.Temperature != 0 {
		body["temperature"] = cfg.Temperature
	}

	data, _ := json.Marshal(body)

	// Child context with timeout
	to := a.timeout
	if dl, ok := ctx.Deadline(); ok {
		if left := time.Until(dl); left < to {
			to = left
		}
	}
	reqCtx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	if a.http == nil {
		a.http = &http.Client{Timeout: to}
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, 0, fmt.Errorf("chat api http %d", resp.StatusCode)
	}

	var outResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&outResp); err != nil {
		return false, 0, err
	}
	if len(outResp.Choices) == 0 || strings.TrimSpace(outResp.Choices[0].Message.Content) == "" {
		return false, 0, errors.New("empty custom response")
	}

	raw := strings.TrimSpace(outResp.Choices[0].Message.Content)
	if i := strings.Index(raw, "{"); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndex(raw, "}"); j > 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}

	var parsed struct {
		Flagged    bool     `json:"flagged"`
		Confidence float64  `json:"confidence"`
		Categories []string `json:"categories"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return false, 0, fmt.Errorf("custom JSON parse failed: %w", err)
	}
	thr := cfg.ConfidenceThreshold
	if thr == 0 {
		thr = 0.7
	}
	flagged := parsed.Flagged || parsed.Confidence >= thr
	return flagged, parsed.Confidence, nil
}

func valueOr(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}
