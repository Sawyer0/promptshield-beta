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
	// Fake transport simulates a successful moderation response with flagged=true
	tr := rtFunc(func(r *http.Request) (*http.Response, error) {
		body := `{
			"id": "modr-test",
			"model": "omni-moderation-latest",
			"results": [
				{
					"flagged": true,
					"categories": {"harassment": true},
					"category_scores": {"harassment": 0.9}
				}
			]
		}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(body))}, nil
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

	ok, _, err := a.Analyze(ctx, "sensitive", rules.Semantic{Model: "gpt-4", AnalysisPrompt: "decide {input}", MaxTokens: 1})
	if err != nil {
		t.Fatalf("Analyze returned unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected violation true")
	}
	// Second call should hit cache
	ok, _, err = a.Analyze(ctx, "sensitive", rules.Semantic{Model: "gpt-4", AnalysisPrompt: "decide {input}", MaxTokens: 1})
	if err != nil || !ok {
		t.Fatalf("cache Analyze got err=%v ok=%v", err, ok)
	}
	logs := buf.String()
	if !strings.Contains(logs, "moderation analysis complete") {
		t.Errorf("expected moderation analysis complete log, got: %s", logs)
	}
	if !strings.Contains(logs, "moderation cache hit") {
		t.Errorf("expected cache hit log on second call, got: %s", logs)
	}
}
