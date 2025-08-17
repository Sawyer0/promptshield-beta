package rules

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func loadDir(dir string) ([]RulePack, error) {
	var packs []RulePack
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// non-recursive initial implementation
			if path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if !hasYAMLExt(path) {
			return nil
		}
		p, err := loadFile(path)
		if err != nil {
			return err
		}
		packs = append(packs, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return packs, nil
}

func loadFile(path string) (RulePack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RulePack{}, fmt.Errorf("read %s: %w", path, err)
	}
	var p RulePack
	if err := yaml.Unmarshal(data, &p); err != nil {
		return RulePack{}, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	if p.Kind == "" {
		p.Kind = "RulePack"
	}
	if p.APIVersion == "" {
		p.APIVersion = "promptshield.io/v1"
	}
	if p.Metadata.Name == "" {
		p.Metadata.Name = filepath.Base(path)
	}
	p.SourcePath = path

	// Validate no unsupported features are used
	if err := validateSupportedFeatures(p); err != nil {
		return RulePack{}, fmt.Errorf("unsupported feature in %s: %w", path, err)
	}

	return p, nil
}

func hasYAMLExt(path string) bool {
	switch filepath.Ext(path) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}
