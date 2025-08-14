package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadPacks_File(t *testing.T) {
	dir := t.TempDir()
	yml := `
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: sample
  version: 0.1.0
rules:
  - id: kw
    level: 1
    severity: WARNING
    keywords: ["hello"]
`
	p := filepath.Join(dir, "pack.yaml")
	require.NoError(t, os.WriteFile(p, []byte(yml), 0o600))

	packs, err := LoadPacks(p)
	require.NoError(t, err)
	require.Len(t, packs, 1)
	require.Equal(t, "sample", packs[0].Metadata.Name)
	require.Len(t, packs[0].Rules, 1)
	require.Equal(t, "kw", packs[0].Rules[0].ID)
}

func TestLoadPacks_EmptyRulesFailsValidate(t *testing.T) {
	dir := t.TempDir()
	yml := `
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: empty
  version: 0.1.0
rules: []
`
	p := filepath.Join(dir, "pack.yaml")
	require.NoError(t, os.WriteFile(p, []byte(yml), 0o600))

	packs, err := LoadPacks(p)
	require.NoError(t, err)
	require.Len(t, packs, 1)
	require.Equal(t, "empty", packs[0].Metadata.Name)
	require.Len(t, packs[0].Rules, 0)
}
