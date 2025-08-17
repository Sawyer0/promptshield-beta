package rulesapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

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
      - regex: "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}"
  - id: pii-ssn
    name: US Social Security Number
    level: 2
    severity: ERROR
    patterns:
      - regex: "\\b\\d{3}-\\d{2}-\\d{4}\\b"
  - id: pii-credit-card
    name: Credit Card Number
    level: 2
    severity: ERROR
    patterns:
      - regex: "\\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14})\\b"
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
  - id: jailbreak-keyword
    level: 1
    severity: ERROR
    keywords: ["jailbreak", "break free"]
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
  - id: compliance-keyword
    level: 1
    severity: WARNING
    keywords: ["compliance", "regulation"]
`
	case "l1":
		// same as default
		fallthrough
	default:
		if template != "" && template != "l1" {
			return "", fmt.Errorf("unknown template %s", template)
		}
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

// List merges rule IDs across one or many RulePack sources (files or directories).
// When paths is empty or nil, it returns an empty slice.
func (s *Service) List(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	var allPacks []rules.RulePack
	for _, p := range paths {
		packs, err := rules.LoadPacks(p)
		if err != nil {
			// fallback: parse minimal YAML looking for spec.rules
			if ids, ferr := extractIDsQuick(p); ferr == nil {
				for _, id := range ids {
					allPacks = append(allPacks, rules.RulePack{Rules: []rules.Rule{{ID: id}}})
				}
				continue
			}
			return nil, err
		}
		allPacks = append(allPacks, packs...)
	}
	merged := rules.MergePacks(allPacks)
	ids := make([]string, 0, len(merged))
	for _, r := range merged {
		ids = append(ids, r.ID)
	}
	if len(ids) == 0 {
		for _, p := range paths {
			extra, _ := extractIDsQuick(p)
			ids = append(ids, extra...)
		}
	}
	// deduplicate
	idSet := make(map[string]struct{}, len(ids))
	var unique []string
	for _, id := range ids {
		if _, ok := idSet[id]; !ok && id != "" {
			idSet[id] = struct{}{}
			unique = append(unique, id)
		}
	}
	sort.Strings(unique)
	return unique, nil
}

// extractIDsQuick is a lightweight YAML parser that pulls rule IDs from documents
// shaped like:
// spec:
//
//	rules:
//	  - id: foo
//	  - id: bar
func extractIDsQuick(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var obj struct {
		Spec struct {
			Rules []struct {
				ID string `yaml:"id"`
			} `yaml:"rules"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	var ids []string
	for _, r := range obj.Spec.Rules {
		if r.ID != "" {
			ids = append(ids, r.ID)
		}
	}
	return ids, nil
}
