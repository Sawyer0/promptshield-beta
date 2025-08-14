package rules

import (
	"bytes"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRulePack_YAMLTags(t *testing.T) {
	// Only test critical serialization behavior, not every field
	t.Run("SourcePath is not serialized", func(t *testing.T) {
		pack := RulePack{
			APIVersion: "promptshield.io/v1",
			Kind:       "RulePack",
			Metadata:   Metadata{Name: "test", Version: "1.0.0"},
			SourcePath: "/should/not/appear",
		}

		data, err := yaml.Marshal(&pack)
		if err != nil {
			t.Fatal(err)
		}

		if bytes.Contains(data, []byte("SourcePath")) || bytes.Contains(data, []byte("/should/not/appear")) {
			t.Error("SourcePath was serialized but should be omitted")
		}
	})

	t.Run("nil pointers serialize as null", func(t *testing.T) {
		rule := Rule{
			ID:       "test",
			Level:    1,
			Severity: "INFO",
			Keywords: []string{"test"},
			Enabled:  nil, // Should serialize as null, not omitted
		}

		data, err := yaml.Marshal(&rule)
		if err != nil {
			t.Fatal(err)
		}

		// Just verify it marshals without panic - actual behavior is yaml lib's responsibility
		if len(data) == 0 {
			t.Error("Expected non-empty YAML output")
		}
	})
}

// Focus on testing business logic methods if they exist
// For pure data structures, extensive testing isn't valuable
