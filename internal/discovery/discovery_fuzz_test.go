//go:build go1.18
// +build go1.18

package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzDiscoverPaths(f *testing.F) {
	// Add seed corpus with various path patterns
	f.Add("file.txt")
	f.Add("*.go")
	f.Add("**/*.txt")
	f.Add("../test")
	f.Add("./subdir/")
	f.Add("path with spaces/file.txt")
	f.Add("path\\with\\backslashes")
	f.Add("~/.config/test")
	f.Add("/absolute/path/file")
	f.Add("C:\\Windows\\Path")
	f.Add("../../../../../../etc/passwd")
	f.Add("file\x00null")
	f.Add("file\nwith\nnewlines")
	f.Add("😀emoji🎉path/file.txt")

	f.Fuzz(func(t *testing.T, pathInput string) {
		// Skip certain problematic inputs
		if strings.Contains(pathInput, "\x00") {
			t.Skip("Null bytes in paths are not supported")
		}

		// Create a temporary directory for testing
		root := t.TempDir()

		// Try to use the input as a path (kept for clarity; not used directly)
		_ = filepath.Join(root, "test_"+sanitizeForFilename(pathInput))

		// Create a test file
		testFile := filepath.Join(root, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0o644)
		if err != nil {
			t.Skip("Could not create test file")
		}

		// Test various path inputs
		paths := []string{pathInput}

		// DiscoverPaths should handle any input gracefully
		result, err := DiscoverPaths(paths)

		// The function should either:
		// 1. Return an error for invalid paths
		// 2. Return an empty result for non-matching patterns
		// 3. Return found files for valid patterns

		// It should never panic
		if err == nil && result == nil {
			t.Error("DiscoverPaths returned nil result without error")
		}

		// If the path looks absolute and doesn't exist, should get an error
		if filepath.IsAbs(pathInput) && !strings.Contains(pathInput, "*") && !strings.Contains(pathInput, "?") {
			if _, statErr := os.Stat(pathInput); os.IsNotExist(statErr) && err == nil {
				t.Errorf("Expected error for non-existent absolute path %q", pathInput)
			}
		}
	})
}

func FuzzGlobPatterns(f *testing.F) {
	// Add glob pattern seeds
	f.Add("*")
	f.Add("**")
	f.Add("?")
	f.Add("[abc]")
	f.Add("[!abc]")
	f.Add("[a-z]")
	f.Add("{a,b,c}")
	f.Add("*.{go,txt}")
	f.Add("**/[!.]*.go")
	f.Add("a[b")  // Invalid bracket
	f.Add("a[b-") // Invalid range

	f.Fuzz(func(t *testing.T, pattern string) {
		root := t.TempDir()

		// Create some test files
		testFiles := []string{
			"test.go",
			"test.txt",
			"a.go",
			"b.txt",
			"subdir/test.go",
		}

		for _, file := range testFiles {
			path := filepath.Join(root, file)
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				continue
			}
			if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
				continue
			}
		}

		// Test the glob pattern
		fullPattern := filepath.Join(root, pattern)

		// Should handle any glob pattern without panicking
		result, err := DiscoverPaths([]string{fullPattern})

		// Invalid glob patterns should either:
		// 1. Return an error
		// 2. Return empty results
		// 3. Be treated as literal paths
		_ = result
		_ = err

		// The key is that it doesn't panic
	})
}

// sanitizeForFilename removes characters that are problematic in filenames
func sanitizeForFilename(s string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", "_",
		"\r", "_",
		"\t", "_",
	)

	s = replacer.Replace(s)

	// Limit length
	if len(s) > 50 {
		s = s[:50]
	}

	// Ensure non-empty
	if s == "" {
		s = "empty"
	}

	return s
}
