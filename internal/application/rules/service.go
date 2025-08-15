package rulesapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/promptshield/promptshield/internal/rules"
)

type Service struct{}

func NewService() *Service { return &Service{} }

// InitPack writes a skeleton RulePack to dest. It refuses to overwrite unless force is true.
func (s *Service) InitPack(dest string, force bool) (string, error) {
	if dest == "" {
		dest = "rules/example.yaml"
	}
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return "", fmt.Errorf("%s already exists (use --force to overwrite)", dest)
		}
	} else {
		_ = os.MkdirAll(filepath.Dir(dest), 0o750)
	}
	content := `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-pack
  version: 0.1.0
  description: "Example PromptShield RulePack"
rules:
  - id: hello-keyword
    name: Detect 'hello'
    level: 1
    severity: WARNING
    keywords: ["hello"]
`
	if err := os.WriteFile(dest, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", dest, err)
	}
	return dest, nil
}

// InitPackWithTemplate writes a templated RulePack by tier/goal.
// Supported templates: l1, l2, l3, pii, prompt-injection, industry
func (s *Service) InitPackWithTemplate(dest string, force bool, template string) (string, error) {
	if dest == "" {
		dest = "rules/example.yaml"
	}
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return "", fmt.Errorf("%s already exists (use --force to overwrite)", dest)
		}
	} else {
		_ = os.MkdirAll(filepath.Dir(dest), 0o750)
	}
	var content string
	switch template {
	case "l2":
		content = `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-l2
  version: 0.1.0
  description: "Starter Level 2 (regex) pack"
rules:
  - id: api-key-like
    name: API key style token
    level: 2
    severity: ERROR
    patterns:
      - regex: "(?i)sk-[a-z0-9]{10,}"
        flags: [ignorecase]
`
	case "l3":
		content = `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-l3
  version: 0.1.0
  description: "Starter Level 3 (semantic) pack"
rules:
  - id: sem-manipulation
    level: 3
    severity: ERROR
    semantic:
      model: gpt-4o-mini
      analysis_prompt: |
        Respond VIOLATION or SAFE for: {input}
      confidence_threshold: 0.85
      fallback_on_error: true
`
	case "pii":
		content = `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-pii
  version: 0.1.0
  description: "Starter PII detection pack"
rules:
  - id: pii-email
    name: Email address
    level: 2
    severity: WARNING
    patterns:
      - regex: "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}"
`
	case "prompt-injection":
		content = `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-prompt-injection
  version: 0.1.0
  description: "Starter prompt-injection pack"
rules:
  - id: ignore-previous
    name: Detect instruction skipping
    level: 1
    severity: ERROR
    keywords: ["ignore previous instructions", "disregard earlier instructions"]
`
	case "industry":
		content = `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-industry
  version: 0.1.0
  description: "Starter industry pack to extend"
  tags: ["industry", "template"]
rules:
  - id: placeholder
    name: Replace with your industry signals
    level: 1
    severity: INFO
    keywords: ["replace-me"]
`
	default:
		content = `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-l1
  version: 0.1.0
  description: "Starter Level 1 (keywords) pack"
rules:
  - id: hello-keyword
    name: Detect 'hello'
    level: 1
    severity: WARNING
    keywords: ["hello"]
`
	}
	if err := os.WriteFile(dest, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", dest, err)
	}
	return dest, nil
}

// List returns merged rule IDs from the pack path or directory.
func (s *Service) List(path string) ([]string, error) {
	packs, err := rules.LoadPacks(path)
	if err != nil {
		return nil, err
	}
	merged := rules.MergePacks(packs)
	ids := make([]string, 0, len(merged))
	for _, r := range merged {
		ids = append(ids, r.ID)
	}
	sort.Strings(ids)
	return ids, nil
}
