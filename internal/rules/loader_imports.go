package rules

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// resolveImportSpec resolves an import specification relative to baseDir.
func resolveImportSpec(baseDir, spec string) ([]RulePack, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	// URL import
	if strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "https://") {
		if os.Getenv("PS_ALLOW_NET_IMPORTS") != "1" {
			return nil, fmt.Errorf("network imports disabled (set PS_ALLOW_NET_IMPORTS=1 to enable)")
		}
		return fetchRemotePack(spec)
	}
	// Marketplace slug
	if strings.Contains(spec, "@") && strings.Contains(spec, "/") {
		ps, err := resolveMarketplace(spec)
		if err == nil && len(ps) > 0 {
			return ps, nil
		}
		// Fall through to file resolution if marketplace resolution fails
	}
	// Glob patterns (including recursive **)
	if strings.ContainsAny(spec, "*?[") || strings.Contains(spec, "**") {
		pattern := spec
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(baseDir, pattern)
		}
		if strings.Contains(pattern, "**") {
			return walkAndMatch(pattern)
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		var out []RulePack
		for _, m := range matches {
			st, err := os.Stat(m)
			if err != nil {
				continue
			}
			if st.IsDir() {
				ps, err := loadDir(m)
				if err != nil {
					return nil, err
				}
				out = append(out, ps...)
			} else if hasYAMLExt(m) {
				p, err := loadFile(m)
				if err != nil {
					return nil, err
				}
				out = append(out, p)
			}
		}
		return out, nil
	}
	// Filesystem path
	impPath := spec
	if !filepath.IsAbs(impPath) {
		impPath = filepath.Join(baseDir, impPath)
	}
	st, err := os.Stat(impPath)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return loadDir(impPath)
	}
	if hasYAMLExt(impPath) {
		p, err := loadFile(impPath)
		if err != nil {
			return nil, err
		}
		return []RulePack{p}, nil
	}
	return nil, fmt.Errorf("unsupported import: %s", spec)
}

func walkAndMatch(pattern string) ([]RulePack, error) {
	idx := strings.Index(pattern, "**")
	if idx < 0 {
		return nil, fmt.Errorf("invalid recursive glob: %s", pattern)
	}
	root := filepath.Dir(pattern[:idx])
	if root == "" {
		root = "."
	}
	leaf := pattern[idx+2:]
	leaf = strings.TrimPrefix(leaf, string(filepath.Separator))
	var out []RulePack
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == "dist" || base == "vendor" || base == "build" || base == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		test := filepath.Base(path)
		if strings.Contains(leaf, string(filepath.Separator)) {
			test = filepath.ToSlash(rel)
		}
		ok, _ := filepath.Match(leaf, test)
		if ok && hasYAMLExt(path) {
			p, err := loadFile(path)
			if err == nil {
				out = append(out, p)
			}
		}
		return nil
	})
	return out, nil
}

func fetchRemotePack(url string) ([]RulePack, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var p RulePack
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", url, err)
	}
	p.SourcePath = ""
	return []RulePack{p}, nil
}

func resolveMarketplace(slug string) ([]RulePack, error) {
	// slug format: namespace/name@version
	nsAndName, ver, ok := strings.Cut(slug, "@")
	if !ok {
		return nil, fmt.Errorf("invalid marketplace slug: %s", slug)
	}
	ns, name, ok := strings.Cut(nsAndName, "/")
	if !ok {
		return nil, fmt.Errorf("invalid marketplace slug: %s", slug)
	}
	// Try local marketplace directory first
	base := os.Getenv("PS_MARKETPLACE_DIR")
	if base == "" {
		base = "rules"
	}
	candidates := []string{
		filepath.Join(base, ns, name+"@"+ver+".yaml"),
		filepath.Join(base, ns, name, ver+".yaml"),
		filepath.Join(base, ns, name, ver, "rulepack.yaml"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			p, err := loadFile(c)
			if err != nil {
				return nil, err
			}
			return []RulePack{p}, nil
		}
	}
	// Optionally fetch from registry
	if os.Getenv("PS_ALLOW_NET_IMPORTS") == "1" {
		reg := os.Getenv("PS_REGISTRY_URL")
		if reg == "" {
			reg = "https://registry.promptshield.io"
		}
		urls := []string{
			strings.TrimRight(reg, "/") + "/" + ns + "/" + name + "/" + ver + "/rulepack.yaml",
			strings.TrimRight(reg, "/") + "/" + ns + "/" + name + "@" + ver + ".yaml",
		}
		var lastErr error
		for _, u := range urls {
			ps, err := fetchRemotePack(u)
			if err == nil {
				return ps, nil
			}
			lastErr = err
		}
		if lastErr != nil {
			return nil, lastErr
		}
	}
	return nil, errors.New("marketplace pack not found locally and network disabled")
}
