package audit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileLogger_HashChain(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLogger(&buf)
	if err := logger.Log(Event{Type: "a", Data: map[string]any{"x": 1}}); err != nil {
		t.Fatal(err)
	}
	if err := logger.Log(Event{Type: "b", Data: map[string]any{"y": 2}}); err != nil {
		t.Fatal(err)
	}
	// Decode last line and ensure PrevHash set
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	var last Event
	if err := json.Unmarshal(lines[len(lines)-1], &last); err != nil {
		t.Fatal(err)
	}
	if last.PrevHash == "" || last.Hash == "" {
		t.Fatalf("expected hash/prev-hash populated: %+v", last)
	}
}

func TestRotatingFileLogger_DailyRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit")
	logger, err := NewDailyRotatingLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	if err := logger.Log(Event{Type: "scan_start"}); err != nil {
		t.Fatal(err)
	}
	// Should create today's file
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
}
