package discovery

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	doublestar "github.com/bmatcuk/doublestar/v4"
	gitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/karrick/godirwalk"
)

// DiscoverPaths expands a list of input args into concrete file paths.
// Each arg may be a file, a directory (walked recursively), or a glob pattern.
func DiscoverPaths(args []string) ([]string, error) {
	// If no args provided, return empty without error (useful for piping/streaming modes)
	if len(args) == 0 {
		return []string{}, nil
	}

	// Prepare a repo-aware gitignore evaluator rooted at the repository root if found, otherwise CWD.
	cwd, _ := os.Getwd()
	gitRoot := findGitRoot(cwd)
	if gitRoot == "" {
		gitRoot = cwd
	}
	igCache := newGitignoreCache(gitRoot)

	// Additional hard skip directories regardless of .gitignore
	hardSkip := map[string]struct{}{".git": {}, "node_modules": {}, "vendor": {}, "dist": {}, "build": {}, "target": {}}

	found := make(map[string]struct{})
	resolvedArgCount := 0 // count args that exist (files/dirs) or are valid globs
	hadMissing := false   // at least one arg explicitly missing (non-glob path)

	// Basic input validation to prevent path traversal and null-byte injection
	for _, a := range args {
		if strings.Contains(a, "\x00") {
			return nil, fmt.Errorf("invalid input path: contains null byte")
		}
		// Normalize separators and check for parent directory references
		norm := strings.ReplaceAll(a, "\\", "/")
		if strings.HasPrefix(norm, "../") || strings.Contains(norm, "/../") || strings.HasSuffix(norm, "/..") || norm == ".." {
			return nil, fmt.Errorf("invalid input path: path traversal is not allowed (%s)", a)
		}
		if strings.HasPrefix(norm, "..\\") || strings.Contains(norm, "\\..\\") || strings.HasSuffix(norm, "\\..") {
			return nil, fmt.Errorf("invalid input path: path traversal is not allowed (%s)", a)
		}
		// Apply allow/deny filters if provided via env
		if !allowPath(a) {
			continue
		}
		if denyPath(a) {
			continue
		}
		// Try stat first (do not follow symlinks)
		if fi, err := os.Lstat(a); err == nil {
			resolvedArgCount++
			// Symlink policy: deny by default unless PS_ALLOW_SYMLINKS=true
			if fi.Mode()&os.ModeSymlink != 0 {
				if strings.ToLower(os.Getenv("PS_ALLOW_SYMLINKS")) != "true" {
					return nil, fmt.Errorf("invalid input path: symlinks are not allowed (%s)", a)
				}
				// If allowed, resolve once to absolute for consistent hashing/reporting, but do not traverse outside allowlist/denylist
				if target, rerr := filepath.EvalSymlinks(a); rerr == nil {
					a = target
					// refresh fi after resolution
					if nfi, nerr := os.Lstat(a); nerr == nil {
						fi = nfi
					}
				}
			}
			if fi.IsDir() {
				// Walk directory with godirwalk; apply hardSkip + spec-accurate .gitignore
				root := a
				// Prefer repo root when computing ignore semantics, but restrict walk to 'a'
				localIgnore := newGitignoreCache(gitRoot)
				// Preload ignore for the walk root directory to ensure patterns in that directory apply to its children
				if err := localIgnore.loadDirPatterns(root); err != nil {
					// Non-fatal: log warning but continue discovery
					log.Printf("Warning: failed to load ignore patterns for %s: %v", root, err)
				}
				_ = godirwalk.Walk(root, &godirwalk.Options{
					ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
						return godirwalk.SkipNode
					},
					Unsorted: true,
					Callback: func(path string, de *godirwalk.Dirent) error {
						name := de.Name()
						if de.IsDir() {
							if _, skip := hardSkip[name]; skip {
								return godirwalk.SkipThis
							}
							// Load .gitignore patterns in this directory before visiting children
							if err := localIgnore.loadDirPatterns(path); err != nil {
								return nil
							}
						}
						// Enforce symlink policy within directory walks as well
						if de.ModeType()&os.ModeSymlink != 0 {
							if strings.ToLower(os.Getenv("PS_ALLOW_SYMLINKS")) != "true" {
								return nil
							}
						}
						// Respect gitignore semantics relative to gitRoot
						if localIgnore.isIgnored(path, de.IsDir()) {
							if de.IsDir() {
								return godirwalk.SkipThis
							}
							return nil
						}
						if !de.IsDir() {
							abs, _ := filepath.Abs(path)
							found[abs] = struct{}{}
						}
						return nil
					},
				})
				continue
			}
			abs, _ := filepath.Abs(a)
			found[abs] = struct{}{}
			continue
		} else {
			// If the arg does not exist, decide if it's a glob
			isGlob := strings.ContainsAny(a, "*?[") || strings.Contains(a, "**")
			if !isGlob {
				// Missing plain path should contribute to an error
				hadMissing = true
			}

			// Guard excessive recursive glob depth and fan-out
			if strings.Contains(a, "**") {
				if tooManyGlobs(a) {
					return []string{}, fmt.Errorf("glob pattern too broad: %s", a)
				}
				resolvedArgCount++
				matches, _ := doublestar.FilepathGlob(a)
				for _, full := range matches {
					if st, e := os.Lstat(full); e == nil && !st.IsDir() {
						// Enforce symlink policy for globbed files
						if st.Mode()&os.ModeSymlink != 0 {
							if strings.ToLower(os.Getenv("PS_ALLOW_SYMLINKS")) != "true" {
								continue
							}
						}
						// Filter via gitignore semantics
						if igCache.isIgnored(full, false) {
							continue
						}
						abs, _ := filepath.Abs(full)
						found[abs] = struct{}{}
					}
				}
				continue
			}

			// Treat as normal glob (non-recursive)
			if isGlob {
				if tooManyGlobs(a) {
					return []string{}, fmt.Errorf("glob pattern too broad: %s", a)
				}
				resolvedArgCount++
				matches, _ := doublestar.FilepathGlob(a)
				for _, full := range matches {
					if st, e := os.Lstat(full); e == nil && !st.IsDir() {
						if st.Mode()&os.ModeSymlink != 0 {
							if strings.ToLower(os.Getenv("PS_ALLOW_SYMLINKS")) != "true" {
								continue
							}
						}
						if igCache.isIgnored(full, false) {
							continue
						}
						abs, _ := filepath.Abs(full)
						found[abs] = struct{}{}
					}
				}
			}
		}
	}

	// If no args resolved to anything but args were provided, decide erroring
	if resolvedArgCount == 0 {
		return []string{}, fmt.Errorf("%w", ErrNoInputFiles)
	}

	// Deduplicate and sort for determinism
	out := make([]string, 0, len(found))
	for p := range found {
		out = append(out, p)
	}
	sort.Strings(out)

	if hadMissing {
		return out, fmt.Errorf("one or more input paths not found")
	}
	return out, nil
}

// allowPath checks PS_ALLOW_PATHS (comma-separated prefixes). If unset, allow all.
func allowPath(p string) bool {
	s := os.Getenv("PS_ALLOW_PATHS")
	if s == "" {
		return true
	}
	parts := strings.Split(s, ",")
	for _, pref := range parts {
		pref = strings.TrimSpace(pref)
		if pref == "" {
			continue
		}
		if strings.HasPrefix(filepath.Clean(p), filepath.Clean(pref)) {
			return true
		}
	}
	return false
}

// denyPath checks PS_DENY_PATHS (comma-separated prefixes). If path starts with any, deny.
func denyPath(p string) bool {
	s := os.Getenv("PS_DENY_PATHS")
	if s == "" {
		return false
	}
	parts := strings.Split(s, ",")
	for _, pref := range parts {
		pref = strings.TrimSpace(pref)
		if pref == "" {
			continue
		}
		if strings.HasPrefix(filepath.Clean(p), filepath.Clean(pref)) {
			return true
		}
	}
	return false
}

// tooManyGlobs enforces a simple heuristic to prevent explosive patterns.
func tooManyGlobs(p string) bool {
	// Count wildcards and restrict combined to a small number
	wc := strings.Count(p, "*") + strings.Count(p, "?")
	return wc > 10
}

// ErrNoInputFiles indicates that none of the provided arguments resolved to files.
var ErrNoInputFiles = errors.New("no input files found")

// gitignoreCache maintains compiled gitignore matchers with directory pattern loading.
type gitignoreCache struct {
	root        string
	patterns    []gitignore.Pattern
	matcher     gitignore.Matcher
	loadedDirs  map[string]struct{}
	lastPatSize int
}

func newGitignoreCache(root string) *gitignoreCache {
	return &gitignoreCache{
		root:       root,
		patterns:   make([]gitignore.Pattern, 0, 16),
		loadedDirs: make(map[string]struct{}),
	}
}

// loadDirPatterns loads and compiles patterns for the provided absolute directory if not already loaded.
func (g *gitignoreCache) loadDirPatterns(absDir string) error {
	if g == nil {
		return nil
	}
	absDir = filepath.Clean(absDir)
	if _, ok := g.loadedDirs[absDir]; ok {
		return nil
	}
	// mark as checked regardless of existence to avoid repeated stats
	g.loadedDirs[absDir] = struct{}{}
	giPath := filepath.Join(absDir, ".gitignore")
	if fi, err := os.Stat(giPath); err == nil && !fi.IsDir() {
		data, err := os.ReadFile(giPath)
		if err == nil {
			baseRel, _ := filepath.Rel(g.root, absDir)
			baseRel = filepath.ToSlash(baseRel)
			if baseRel == "." {
				baseRel = ""
			}
			lines := strings.Split(string(data), "\n")
			for _, ln := range lines {
				l := strings.TrimSpace(ln)
				if l == "" || strings.HasPrefix(l, "#") {
					continue
				}
				var base []string
				if baseRel != "" {
					base = strings.Split(filepath.ToSlash(baseRel), "/")
				}
				p := gitignore.ParsePattern(l, base)
				g.patterns = append(g.patterns, p)
			}
		}
	}
	// Rebuild matcher only if pattern list changed
	if len(g.patterns) != g.lastPatSize {
		g.matcher = gitignore.NewMatcher(g.patterns)
		g.lastPatSize = len(g.patterns)
	}
	return nil
}

// isIgnored reports whether the absolute path should be ignored according to loaded patterns.
func (g *gitignoreCache) isIgnored(absPath string, isDir bool) bool {
	if g == nil {
		return false
	}
	// Ensure patterns are loaded for all directories along the path
	absPath = filepath.Clean(absPath)
	dir := absPath
	if !isDir {
		dir = filepath.Dir(absPath)
	}
	g.ensureAncestryLoaded(dir)
	if g.matcher == nil {
		return false
	}
	rel, err := filepath.Rel(g.root, absPath)
	if err != nil {
		return false
	}
	segments := strings.Split(filepath.ToSlash(rel), "/")
	return g.matcher.Match(segments, isDir)
}

func (g *gitignoreCache) ensureAncestryLoaded(absDir string) {
	if g == nil {
		return
	}
	absDir = filepath.Clean(absDir)
	root := filepath.Clean(g.root)
	if !strings.HasPrefix(absDir, root) {
		return
	}
	// load patterns for each directory from root to absDir
	rel, err := filepath.Rel(root, absDir)
	if err != nil {
		return
	}
	if rel == "." {
		_ = g.loadDirPatterns(root)
		return
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	cur := root
	for _, part := range parts {
		cur = filepath.Join(cur, part)
		_ = g.loadDirPatterns(cur)
	}
}

// findGitRoot walks up from start to locate the nearest directory containing a .git directory.
func findGitRoot(start string) string {
	d := filepath.Clean(start)
	for {
		if fi, err := os.Stat(filepath.Join(d, ".git")); err == nil && fi.IsDir() {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}
