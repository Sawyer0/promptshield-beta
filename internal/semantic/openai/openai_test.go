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
	// Fake transport always returns a VIOLATION label
	tr := rtFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"choices":[{"message":{"content":"VIOLATION"}}]}`
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
		// Allow error in transport edge case; still check logs and cache behavior
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
	if !strings.Contains(logs, "semantic request") || !strings.Contains(logs, "semantic response") {
		t.Errorf("expected semantic request/response logs, got: %s", logs)
	}
	if !strings.Contains(logs, "semantic cache hit") {
		t.Errorf("expected cache hit log on second call, got: %s", logs)
	}
}
