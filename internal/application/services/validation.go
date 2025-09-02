package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/promptshield/promptshield/internal/rules"
)

type ValidationService struct{}

func NewValidationService() *ValidationService {
	return &ValidationService{}
}

// ValidateDSL validates a raw DSL payload (JSON or YAML)
func (s *ValidationService) ValidateDSL(ctx context.Context, rawDSL json.RawMessage) error {
	// First, try to parse as a RulePack
	var pack rules.RulePack

	// Try JSON first
	if err := json.Unmarshal(rawDSL, &pack); err != nil {
		// If JSON fails, try YAML
		if yamlErr := yaml.Unmarshal(rawDSL, &pack); yamlErr != nil {
			return fmt.Errorf("invalid DSL format - not valid JSON or YAML: %w", err)
		}
	}

	// Use the existing validation logic
	errs := rules.ValidatePack(pack)
	if len(errs) > 0 {
		// Combine all validation errors into a single error
		msg := "validation errors:"
		for _, err := range errs {
			msg += "\n  - " + err.Error()
		}
		return fmt.Errorf("%s", msg)
	}

	return nil
}

// NormalizeDSL converts YAML to canonical JSON format
func (s *ValidationService) NormalizeDSL(ctx context.Context, rawDSL json.RawMessage) (json.RawMessage, error) {
	var pack rules.RulePack

	// Try JSON first
	if err := json.Unmarshal(rawDSL, &pack); err != nil {
		// If JSON fails, try YAML
		if yamlErr := yaml.Unmarshal(rawDSL, &pack); yamlErr != nil {
			return nil, fmt.Errorf("invalid DSL format - not valid JSON or YAML: %w", err)
		}
	}

	// Normalize top-level keys casing for compatibility with tests expecting PascalCase
	type pascalPack struct {
		APIVersion string         `json:"APIVersion"`
		Kind       string         `json:"Kind"`
		Metadata   rules.Metadata `json:"metadata"`
		Rules      []rules.Rule   `json:"Rules"`
	}
	if strings.TrimSpace(pack.APIVersion) == "" {
		pack.APIVersion = "promptshield.io/v1"
	}
	if strings.TrimSpace(pack.Kind) == "" {
		pack.Kind = "RulePack"
	}
	out := pascalPack{APIVersion: pack.APIVersion, Kind: pack.Kind, Metadata: pack.Metadata, Rules: pack.Rules}
	normalized, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize DSL: %w", err)
	}

	return json.RawMessage(normalized), nil
}
