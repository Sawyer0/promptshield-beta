package rules

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestLoadPacks_LocalImportsAndGlobs(t *testing.T) {
	dir := t.TempDir()
	// Base pack imports a file and a glob
	_ = writeFile(t, dir, "dep/a.yaml", `apiVersion: promptshield.io/v1
kind: RulePack
metadata: {name: dep-a}
rules: [{id: a1, level: 1, severity: INFO, keywords: [hello]}]
`)
	_ = writeFile(t, dir, "dep/b.yaml", `apiVersion: promptshield.io/v1
kind: RulePack
metadata: {name: dep-b}
rules: [{id: b1, level: 1, severity: INFO, keywords: [world]}]
`)
	basePath := writeFile(t, dir, "base.yaml", `apiVersion: promptshield.io/v1
kind: RulePack
metadata: {name: base}
imports: ["dep/a.yaml", "dep/*.yaml"]
rules: []
`)
	packs, err := LoadPacks(basePath)
	if err != nil {
		t.Fatalf("LoadPacks: %v", err)
	}
	// Expect base + dep-a + dep-b (no duplicates)
	names := make(map[string]bool)
	for _, p := range packs {
		names[p.Metadata.Name] = true
	}
	for _, want := range []string{"base", "dep-a", "dep-b"} {
		if !names[want] {
			t.Fatalf("missing pack %s; got %v", want, names)
		}
	}
}

func TestLoadPacks_CircularImports(t *testing.T) {
	dir := t.TempDir()
	aPath := writeFile(t, dir, "a.yaml", `apiVersion: promptshield.io/v1
kind: RulePack
metadata: {name: A}
imports: ["b.yaml"]
rules: []
`)
	_ = writeFile(t, dir, "b.yaml", `apiVersion: promptshield.io/v1
kind: RulePack
metadata: {name: B}
imports: ["a.yaml"]
rules: []
`)
	packs, err := LoadPacks(aPath)
	if err != nil {
		t.Fatalf("LoadPacks: %v", err)
	}
	// Should contain A and B once each
	counts := map[string]int{}
	for _, p := range packs {
		counts[p.Metadata.Name]++
	}
	if counts["A"] != 1 || counts["B"] != 1 {
		t.Fatalf("expected one of each A/B, got %+v", counts)
	}
}

func TestLoadPacks_NetworkImportsGated(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Avoid CRLF surprises in quick test; logic is platform-agnostic
	}
	// Serve a simple RulePack over HTTP
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("apiVersion: promptshield.io/v1\nkind: RulePack\nmetadata: {name: remote}\nrules: []\n"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	base := writeFile(t, dir, "base.yaml", "apiVersion: promptshield.io/v1\nkind: RulePack\nmetadata: {name: base}\nimports: [\n  \""+srv.URL+"/pack.yaml\"\n]\nrules: []\n")

	// By default network imports disabled
	if _, err := LoadPacks(base); err == nil {
		t.Fatalf("expected error when PS_ALLOW_NET_IMPORTS not set")
	}
	// Enable and expect success
	os.Setenv("PS_ALLOW_NET_IMPORTS", "1")
	defer os.Unsetenv("PS_ALLOW_NET_IMPORTS")
	packs, err := LoadPacks(base)
	if err != nil {
		t.Fatalf("LoadPacks network: %v", err)
	}
	var hasRemote bool
	for _, p := range packs {
		if strings.EqualFold(p.Metadata.Name, "remote") {
			hasRemote = true
		}
	}
	if !hasRemote {
		t.Fatalf("expected remote pack, got %+v", packs)
	}
}
