package metrics

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestNDJSONWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	// Write multiple metrics
	metrics := []map[string]any{
		{"event": "scan", "files": 10},
		{"event": "violation", "severity": "HIGH"},
	}

	for _, m := range metrics {
		if err := w.Write(m); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Verify one JSON object per line
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != len(metrics) {
		t.Errorf("got %d lines, want %d", len(lines), len(metrics))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Errorf("line %d invalid JSON: %v", i, err)
		}
	}
}

func TestNDJSONWriterConcurrency(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	// Concurrent writes
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w.Write(map[string]any{"id": id})
		}(i)
	}
	wg.Wait()

	// Should have exactly 100 lines
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 100 {
		t.Errorf("got %d lines, want 100", len(lines))
	}
}
