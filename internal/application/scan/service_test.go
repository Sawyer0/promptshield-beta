package scan

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfg "github.com/promptshield/promptshield/internal/config"
	"github.com/promptshield/promptshield/internal/testutil/fixtures"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
	"github.com/promptshield/promptshield/pkg/types"
)

// helper to write a temp file with content and return its path
func writeTempFile(t *testing.T, dir, name, content string) string {
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func newService(fake *mocks.FakeScanner, c *cfg.Config) *Service {
	return &Service{scanner: fake, config: c}
}

func TestWorkerAutoSizing(t *testing.T) {
	tempDir := t.TempDir()
	file := writeTempFile(t, tempDir, "single.txt", fixtures.CleanTextContent)

	fake := mocks.NewFakeScanner()
	fake.ScanResults = []types.ScanResult{{Input: file}}

	svc := newService(fake, &cfg.Config{})
	_, err := svc.Scan(context.Background(), []string{file}, Options{})
	require.NoError(t, err)
	assert.Equal(t, 1, fake.LastWorkerCount)
}

func TestTotalScanTimeout(t *testing.T) {
	tempDir := t.TempDir()
	file := writeTempFile(t, tempDir, "slow.txt", fixtures.CleanTextContent)

	fake := mocks.NewFakeScanner()
	fake.SleepDuration = 50 // ms

	c := &cfg.Config{}
	c.Performance.TotalScanTimeout = "10ms"

	svc := newService(fake, c)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := svc.Scan(ctx, []string{file}, Options{})
	assert.Error(t, err)
}

func TestLevel3RequiresSemanticAnalyzer(t *testing.T) {
	tempDir := t.TempDir()
	ruleFile := writeTempFile(t, tempDir, "l3.json", fixtures.Level3RulepackJSON)

	fake := mocks.NewFakeScanner() // HasSemantic defaults to false

	svc := newService(fake, &cfg.Config{})
	_, err := svc.Scan(context.Background(), []string{tempDir}, Options{RulepackPath: ruleFile})
	assert.Error(t, err)
}

func TestContextKVsMerged(t *testing.T) {
	tempDir := t.TempDir()
	file := writeTempFile(t, tempDir, "ctx.txt", fixtures.CleanTextContent)

	fake := mocks.NewFakeScanner()
	svc := newService(fake, &cfg.Config{})

	kvs := []string{"source=cli", "user=test"}
	_, err := svc.Scan(context.Background(), []string{file}, Options{ContextKVs: kvs})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"source": "cli", "user": "test"}, fake.RuntimeContext)
}
