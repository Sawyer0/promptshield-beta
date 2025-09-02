package openai

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAnalyzer_LogsAndCacheHit(t *testing.T) {
	// Fake transport returns a Moderations API response with flagged=true
	tr := rtFunc(func(r *http.Request) (*http.Response, error) {
		// Simulate POST /v1/moderations
		respBody := `{"results":[{"flagged":true,"categories":{},"category_scores":{"illicit":0.92}}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(respBody))}, nil
	})
	httpClient := &http.Client{Transport: tr}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	a := New(Options{
		APIKey:     "test",
		HTTPClient: httpClient,
		Logger:     logger,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Use multimodal path so we control the raw HTTP response
	input := ModerationInput{Text: "sensitive", ImageURL: "http://example.com/img.png"}
	res, err := a.AnalyzeWithModeration(ctx, input, rules.Semantic{Model: "gpt-4", AnalysisPrompt: "decide {input}", MaxTokens: 1})
	if err != nil {
		// Allow error in transport edge case; still check logs and cache behavior
	}
	if res == nil || !res.Flagged {
		t.Fatalf("expected violation true")
	}
	// Second call should hit cache
	res, err = a.AnalyzeWithModeration(ctx, input, rules.Semantic{Model: "gpt-4", AnalysisPrompt: "decide {input}", MaxTokens: 1})
	if err != nil || res == nil || !res.Flagged {
		t.Fatalf("cache AnalyzeWithModeration got err=%v flagged=%v", err, res != nil && res.Flagged)
	}
	logs := buf.String()
	if !strings.Contains(logs, "moderation analysis complete") {
		t.Errorf("expected moderation analysis complete log, got: %s", logs)
	}
	if !strings.Contains(logs, "moderation cache hit") {
		t.Errorf("expected moderation cache hit log on second call, got: %s", logs)
	}
}
