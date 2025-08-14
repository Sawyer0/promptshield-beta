package rules

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoadPacks loads all YAML rule packs from a specific file or directory.
// - If path is a file, only that file is loaded.
// - If path is a directory, all *.y{a,}ml files are loaded (non-recursive for now).
func LoadPacks(path string) ([]RulePack, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	var packs []RulePack
	if stat.IsDir() {
		ps, err := loadDir(path)
		if err != nil {
			return nil, err
		}
		packs = append(packs, ps...)
	} else {
		p, err := loadFile(path)
		if err != nil {
			return nil, err
		}
		packs = append(packs, p)
	}
	// Resolve imports recursively (filesystem paths, globs, optional URLs/marketplace slugs)
	// seen: cycle prevention; emitted: de-duplication of output
	seen := make(map[string]struct{})
	emitted := make(map[string]struct{})
	var out []RulePack
	var importErr error

	var resolve func(baseDir string, pack RulePack)
	resolve = func(baseDir string, pack RulePack) {
		// Emit current pack once
		selfKey := pack.SourcePath
		if selfKey == "" {
			selfKey = pack.Metadata.Name
		}
		if _, ok := emitted[selfKey]; !ok {
			emitted[selfKey] = struct{}{}
			out = append(out, pack)
		}
		for _, spec := range pack.Imports {
			subpacks, err := resolveImportSpec(baseDir, spec)
			if err != nil {
				importErr = fmt.Errorf("resolving import %q from %s: %w", spec, pack.Metadata.Name, err)
				return
			}
			for _, sp := range subpacks {
				key := sp.SourcePath
				if key == "" {
					// Remote or virtual source; use spec + name as a key
					key = spec + "::" + sp.Metadata.Name
				}
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				b := filepath.Dir(sp.SourcePath)
				if b == "." && sp.SourcePath == "" {
					b = baseDir
				}
				resolve(b, sp)
			}
		}
	}
	for _, p := range packs {
		base := filepath.Dir(p.SourcePath)
		if base == "." && p.SourcePath == "" {
			base = "."
		}
		// Seed seen with the base pack's key to avoid immediate re-entry
		k := p.SourcePath
		if k == "" {
			k = p.Metadata.Name
		}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
		}
		resolve(base, p)
		if importErr != nil {
			return nil, importErr
		}
	}
	return out, nil
}
