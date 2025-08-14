package discovery

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/promptshield/promptshield/internal/testutil"
)

func TestDiscoverPaths(t *testing.T) {
	tf := testutil.NewTempFiles(t)

	// Setup test files
	tf.WriteFile("a/x.txt", "content")
	tf.WriteFile("b/y.json", "content")

	tests := []struct {
		name    string
		paths   []string
		want    int
		wantErr bool
	}{
		{"single file", []string{tf.Path("a/x.txt")}, 1, false},
		{"directory", []string{tf.Path("a")}, 1, false},
		{"glob", []string{tf.Path("*/*.txt")}, 1, false},
		{"recursive glob", []string{tf.Path("**/*.json")}, 1, false},
		{"mixed", []string{tf.Path("a/x.txt"), tf.Path("b")}, 2, false},
		{"nonexistent", []string{"/nonexistent/path"}, 0, true},
		{"empty", []string{}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DiscoverPaths(tt.paths)
			testutil.AssertError(t, err, tt.wantErr)
			testutil.AssertCount(t, len(got), tt.want, "files")
		})
	}
}

func TestVendorSkip(t *testing.T) {
	tf := testutil.NewTempFiles(t)

	// Create vendor dirs with files
	vendorDirs := []string{".git", "node_modules", "vendor", "dist", "build", "target"}
	for _, dir := range vendorDirs {
		tf.WriteFile(dir+"/file.txt", "skip")
	}

	// Create normal file
	tf.WriteFile("keep.txt", "keep")

	got, err := DiscoverPaths([]string{tf.Root()})
	if err != nil {
		t.Fatal(err)
	}

	testutil.AssertCount(t, len(got), 1, "files (vendor dirs should be skipped)")
}

func TestDeterministicOrder(t *testing.T) {
	tf := testutil.NewTempFiles(t)

	// Create files in random order
	for _, name := range []string{"z.txt", "a.txt", "m.txt"} {
		tf.WriteFile(name, "test")
	}

	// Get results multiple times
	var first []string
	for i := 0; i < 3; i++ {
		got, _ := DiscoverPaths([]string{tf.Root()})
		if i == 0 {
			first = got
		} else if !reflect.DeepEqual(got, first) {
			t.Error("results not deterministic")
		}
	}

	// Verify sorted
	if !sort.StringsAreSorted(first) {
		t.Error("results not sorted")
	}
}

func TestAllowDenyPathsAndGlobLimit(t *testing.T) {
	tf := testutil.NewTempFiles(t)
	tf.WriteFile("keep/a.txt", "x")
	tf.WriteFile("deny/a.txt", "x")
	t.Setenv("PS_ALLOW_PATHS", tf.Path("keep"))
	t.Setenv("PS_DENY_PATHS", tf.Path("deny"))

	// Allowed path should resolve
	got, err := DiscoverPaths([]string{tf.Path("keep/a.txt"), tf.Path("deny/a.txt")})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 allowed file, got %d", len(got))
	}

	// Overly broad glob should error (heuristic)
	_, err = DiscoverPaths([]string{"**/*/*/*/*/*/*/*/*/*/*"})
	if err == nil {
		t.Fatal("expected error for overly broad glob pattern")
	}
}

func TestGitignoreSemantics(t *testing.T) {
	tf := testutil.NewTempFiles(t)

	// Create .gitignore with file and directory patterns and a negation
	tf.WriteFile(".gitignore", strings.Join([]string{
		"skip.txt",
		"skipped/",
		"!skipped/keep.txt",
		"", // trailing newline
	}, "\n"))

	// Files to test
	tf.WriteFile("skip.txt", "x")
	tf.WriteFile("kept.txt", "x")
	tf.WriteFile("skipped/a.txt", "x")
	tf.WriteFile("skipped/keep.txt", "x")

	got, err := DiscoverPaths([]string{tf.Root()})
	if err != nil {
		t.Fatal(err)
	}

	// Build a set for quick lookup
	have := map[string]struct{}{}
	for _, p := range got {
		have[p] = struct{}{}
	}

	// According to gitignore semantics, files inside an ignored directory cannot be re-included.
	mustContain := []string{
		tf.Path("kept.txt"),
	}
	mustNotContain := []string{
		tf.Path("skip.txt"),
		tf.Path("skipped/a.txt"),
		tf.Path("skipped/keep.txt"),
	}

	for _, p := range mustContain {
		if _, ok := have[p]; !ok {
			t.Errorf("expected to include %s, but it was filtered", p)
		}
	}
	for _, p := range mustNotContain {
		if _, ok := have[p]; ok {
			t.Errorf("expected to exclude %s, but it was included", p)
		}
	}
}
