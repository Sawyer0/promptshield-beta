package rulesapp

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRulesService_InitPack(t *testing.T) {
	// TP-4.1 InitPack writes file & returns path
	tempDir := t.TempDir()
	packPath := filepath.Join(tempDir, "test-pack.yaml")

	svc := &Service{}

	resultPath, err := svc.InitPack(packPath, false)
	require.NoError(t, err)
	assert.Equal(t, packPath, resultPath)

	// Verify file exists and contains RulePack
	content, err := ioutil.ReadFile(packPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "RulePack")
	assert.Contains(t, string(content), "apiVersion")
	assert.Contains(t, string(content), "kind")
}

func TestRulesService_InitPack_FileExists(t *testing.T) {
	// TP-4.2 InitPack file exists & force=false → error
	tempDir := t.TempDir()
	packPath := filepath.Join(tempDir, "existing-pack.yaml")

	// Create existing file
	err := ioutil.WriteFile(packPath, []byte("existing content"), 0644)
	require.NoError(t, err)

	svc := &Service{}

	// Should error without force
	_, err = svc.InitPack(packPath, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exists")

	// Original content should be unchanged
	content, err := ioutil.ReadFile(packPath)
	require.NoError(t, err)
	assert.Equal(t, "existing content", string(content))
}

func TestRulesService_InitPackWithTemplate_Force(t *testing.T) {
	// TP-4.3 InitPackWithTemplate force=true overwrites existing
	tempDir := t.TempDir()
	packPath := filepath.Join(tempDir, "force-pack.yaml")

	// Create existing file
	err := ioutil.WriteFile(packPath, []byte("old content"), 0644)
	require.NoError(t, err)

	svc := &Service{}

	// Should overwrite with force=true
	resultPath, err := svc.InitPackWithTemplate(packPath, true, "l1")
	require.NoError(t, err)
	assert.Equal(t, packPath, resultPath)

	// Content should be new template
	content, err := ioutil.ReadFile(packPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "RulePack")
	assert.NotContains(t, string(content), "old content")
}

func TestRulesService_InitPackWithTemplate_AllTemplates(t *testing.T) {
	// TP-4.4 InitPackWithTemplate each template creates file with expected metadata
	tempDir := t.TempDir()

	templates := []struct {
		name     string
		expected []string
	}{
		{
			name:     "l1",
			expected: []string{"level: 1", "RulePack", "keywords"},
		},
		{
			name:     "l2",
			expected: []string{"level: 2", "RulePack", "regex"},
		},
		{
			name:     "l3",
			expected: []string{"level: 3", "RulePack", "semantic"},
		},
		{
			name:     "pii",
			expected: []string{"RulePack", "pii-email", "Email address"},
		},
		{
			name:     "prompt-injection",
			expected: []string{"RulePack", "prompt-injection", "ignore previous"},
		},
		{
			name:     "industry",
			expected: []string{"RulePack", "example-industry", "replace-me"},
		},
	}

	svc := &Service{}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			packPath := filepath.Join(tempDir, tt.name+"-pack.yaml")

			resultPath, err := svc.InitPackWithTemplate(packPath, false, tt.name)
			require.NoError(t, err)
			assert.Equal(t, packPath, resultPath)

			// Verify content matches template expectations
			content, err := ioutil.ReadFile(packPath)
			require.NoError(t, err)
			contentStr := string(content)

			for _, expected := range tt.expected {
				assert.Contains(t, contentStr, expected,
					"Template %s should contain '%s'", tt.name, expected)
			}
		})
	}
}

func TestRulesService_List(t *testing.T) {
	// TP-4.5 List merges IDs across multiple packs and returns sorted slice
	tempDir := t.TempDir()

	// Create multiple pack files with different rules
	pack1 := `
apiVersion: promptshield.io/v1
kind: RulePack
spec:
  rules:
    - id: rule-alpha
      name: Alpha Rule
      level: 1
    - id: rule-charlie
      name: Charlie Rule
      level: 1
`

	pack2 := `
apiVersion: promptshield.io/v1
kind: RulePack
spec:
  rules:
    - id: rule-bravo
      name: Bravo Rule
      level: 1
    - id: rule-delta
      name: Delta Rule
      level: 1
`

	pack1Path := filepath.Join(tempDir, "pack1.yaml")
	pack2Path := filepath.Join(tempDir, "pack2.yaml")

	err := ioutil.WriteFile(pack1Path, []byte(pack1), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(pack2Path, []byte(pack2), 0644)
	require.NoError(t, err)

	svc := &Service{}

	// List should merge and sort IDs
	ids, err := svc.List([]string{pack1Path, pack2Path})
	require.NoError(t, err)

	// Should have all 4 rules sorted alphabetically
	expected := []string{"rule-alpha", "rule-bravo", "rule-charlie", "rule-delta"}
	assert.Equal(t, expected, ids)
}

func TestRulesService_List_Deduplication(t *testing.T) {
	tempDir := t.TempDir()

	// Create packs with duplicate IDs
	pack1 := `
spec:
  rules:
    - id: duplicate-rule
      level: 1
    - id: unique-rule-1
      level: 1
`

	pack2 := `
spec:
  rules:
    - id: duplicate-rule
      level: 1
    - id: unique-rule-2
      level: 1
`

	pack1Path := filepath.Join(tempDir, "pack1.yaml")
	pack2Path := filepath.Join(tempDir, "pack2.yaml")

	err := ioutil.WriteFile(pack1Path, []byte(pack1), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(pack2Path, []byte(pack2), 0644)
	require.NoError(t, err)

	svc := &Service{}

	ids, err := svc.List([]string{pack1Path, pack2Path})
	require.NoError(t, err)

	// Should deduplicate the duplicate-rule
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "duplicate-rule")
	assert.Contains(t, ids, "unique-rule-1")
	assert.Contains(t, ids, "unique-rule-2")

	// Count occurrences of duplicate-rule
	count := 0
	for _, id := range ids {
		if id == "duplicate-rule" {
			count++
		}
	}
	assert.Equal(t, 1, count, "duplicate-rule should appear only once")
}

func TestRulesService_InitPack_InvalidPath(t *testing.T) {
	svc := &Service{}

	// Test with invalid paths
	invalidPaths := []string{
		"/nonexistent/dir/pack.yaml",
		"",
		"/root/no-permission/pack.yaml",
	}

	for _, path := range invalidPaths {
		t.Run(path, func(t *testing.T) {
			_, err := svc.InitPack(path, false)
			assert.Error(t, err)
		})
	}
}

func TestRulesService_InitPackWithTemplate_InvalidTemplate(t *testing.T) {
	tempDir := t.TempDir()
	packPath := filepath.Join(tempDir, "pack.yaml")

	svc := &Service{}

	// Test with invalid template name
	_, err := svc.InitPackWithTemplate(packPath, false, "invalid-template")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template")
}

func TestRulesService_List_EmptyPacks(t *testing.T) {
	svc := &Service{}

	// Empty pack list
	ids, err := svc.List([]string{})
	require.NoError(t, err)
	assert.Empty(t, ids)

	// Nil pack list
	ids, err = svc.List(nil)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestRulesService_List_InvalidFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid YAML file
	invalidPath := filepath.Join(tempDir, "invalid.yaml")
	err := ioutil.WriteFile(invalidPath, []byte("not: valid: yaml: structure:::"), 0644)
	require.NoError(t, err)

	svc := &Service{}

	// Should handle invalid files gracefully
	_, err = svc.List([]string{invalidPath})
	assert.Error(t, err)
}
