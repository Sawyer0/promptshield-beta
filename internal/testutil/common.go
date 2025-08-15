package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TempFiles creates temporary files for testing
type TempFiles struct {
	t    *testing.T
	root string
}

// NewTempFiles creates a temp directory with helper methods
func NewTempFiles(t *testing.T) *TempFiles {
	t.Helper()
	return &TempFiles{
		t:    t,
		root: t.TempDir(),
	}
}

// Root returns the temp directory path
func (tf *TempFiles) Root() string {
	return tf.root
}

// WriteFile creates a file with content
func (tf *TempFiles) WriteFile(relPath, content string) string {
	tf.t.Helper()
	fullPath := filepath.Join(tf.root, relPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		tf.t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
		tf.t.Fatal(err)
	}
	return fullPath
}

// Path returns full path for a relative path
func (tf *TempFiles) Path(relPath string) string {
	return filepath.Join(tf.root, relPath)
}

// AssertCount checks expected result count
func AssertCount(t *testing.T, got, want int, what string) {
	t.Helper()
	if got != want {
		t.Errorf("got %d %s, want %d", got, what, want)
	}
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

// AssertError checks if error occurred when expected
func AssertError(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Errorf("error = %v, wantErr = %v", err, wantErr)
	}
}

// contains is a helper for string containment
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
