package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindConfigFile(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string // Returns temp dir
		cleanup   func(string)
		files     map[string]string // relative path -> content
		envVars   map[string]string
		want      string // expected relative path from temp dir
		wantFound bool
	}{
		{
			name: "finds promptshield.yaml in current directory",
			files: map[string]string{
				"promptshield.yaml": "output_format: json\n",
			},
			want:      "promptshield.yaml",
			wantFound: true,
		},
		{
			name: "finds promptshield.yml variant",
			files: map[string]string{
				"promptshield.yml": "output_format: json\n",
			},
			want:      "promptshield.yml",
			wantFound: true,
		},
		{
			name: "finds .promptshieldrc.yaml",
			files: map[string]string{
				".promptshieldrc.yaml": "output_format: json\n",
			},
			want:      ".promptshieldrc.yaml",
			wantFound: true,
		},
		{
			name: "finds .promptshieldrc.json",
			files: map[string]string{
				".promptshieldrc.json": `{"output_format": "json"}`,
			},
			want:      ".promptshieldrc.json",
			wantFound: true,
		},
		{
			name: "prefers promptshield.yaml over variants",
			files: map[string]string{
				"promptshield.yaml":    "output_format: json\n",
				".promptshieldrc.yaml": "output_format: stylish\n",
			},
			want:      "promptshield.yaml",
			wantFound: true,
		},
		{
			name: "finds config in home directory",
			files: map[string]string{
				".promptshield/promptshield.yaml": "output_format: json\n",
			},
			envVars: map[string]string{
				"HOME": "{TEMPDIR}", // Will be replaced with actual temp dir
			},
			want:      ".promptshield/promptshield.yaml",
			wantFound: true,
		},
		{
			name: "finds config in XDG_CONFIG_HOME",
			files: map[string]string{
				"xdg/promptshield/config.yaml": "output_format: json\n",
			},
			envVars: map[string]string{
				"XDG_CONFIG_HOME": "{TEMPDIR}/xdg",
			},
			want:      "xdg/promptshield/config.yaml",
			wantFound: true,
		},
		{
			name: "current dir takes precedence over home",
			files: map[string]string{
				"promptshield.yaml":               "output_format: json\n",
				".promptshield/promptshield.yaml": "output_format: stylish\n",
			},
			envVars: map[string]string{
				"HOME": "{TEMPDIR}",
			},
			want:      "promptshield.yaml",
			wantFound: true,
		},
		{
			name:      "returns empty when no config found",
			files:     map[string]string{},
			want:      "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()

			// Save and restore original working directory
			origWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origWd)

			// Change to temp directory
			if err := os.Chdir(tempDir); err != nil {
				t.Fatal(err)
			}

			// Create test files
			for path, content := range tt.files {
				dir := filepath.Dir(path)
				if dir != "." {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Set environment variables
			originalEnv := make(map[string]string)
			for key, value := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
				// Replace {TEMPDIR} placeholder
				value = strings.ReplaceAll(value, "{TEMPDIR}", tempDir)
				os.Setenv(key, value)
			}
			defer func() {
				// Restore original environment
				for key, value := range originalEnv {
					if value == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, value)
					}
				}
			}()

			// Call the enhanced findConfigFile function
			got := findConfigFileEnhanced()

			if tt.wantFound {
				if got == "" {
					t.Errorf("Expected to find config file %s, but got empty", tt.want)
					return
				}

				// Check if the found file matches expected
				rel, err := filepath.Rel(tempDir, got)
				if err != nil {
					// If not relative to tempDir, check if it's the absolute path we expect
					expectedAbs := filepath.Join(tempDir, tt.want)
					if got != expectedAbs {
						t.Errorf("Expected config file %s, got %s", expectedAbs, got)
					}
				} else {
					// Normalize path separators for comparison
					rel = filepath.ToSlash(rel)
					want := filepath.ToSlash(tt.want)
					if rel != want {
						t.Errorf("Expected config file %s, got %s", want, rel)
					}
				}
			} else {
				if got != "" {
					t.Errorf("Expected no config file, but got %s", got)
				}
			}
		})
	}
}

func TestConfigDiscoveryPriority(t *testing.T) {
	// Test that config files are discovered in the correct priority order
	tempDir := t.TempDir()

	// Save and restore original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// Create all possible config files
	configs := []struct {
		path     string
		content  string
		priority int // lower number = higher priority
	}{
		{"promptshield.yaml", "source: promptshield.yaml", 1},
		{"promptshield.yml", "source: promptshield.yml", 2},
		{".promptshieldrc.yaml", "source: .promptshieldrc.yaml", 3},
		{".promptshieldrc.yml", "source: .promptshieldrc.yml", 4},
		{".promptshieldrc.json", `{"source": ".promptshieldrc.json"}`, 5},
		{".promptshield/promptshield.yaml", "source: home", 6},
		{".promptshield/config.yaml", "source: home-config", 7},
	}

	// Test each priority level
	for i, config := range configs {
		t.Run(fmt.Sprintf("priority_%d", config.priority), func(t *testing.T) {
			// Create a fresh temp dir for this test
			testDir := filepath.Join(tempDir, fmt.Sprintf("test%d", i))
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatal(err)
			}

			if err := os.Chdir(testDir); err != nil {
				t.Fatal(err)
			}

			// Create all configs with equal or lower priority
			for _, c := range configs {
				if c.priority >= config.priority {
					dir := filepath.Dir(c.path)
					if dir != "." {
						if err := os.MkdirAll(dir, 0755); err != nil {
							t.Fatal(err)
						}
					}
					if err := os.WriteFile(c.path, []byte(c.content), 0644); err != nil {
						t.Fatal(err)
					}
				}
			}

			// Set HOME for home directory configs
			os.Setenv("HOME", testDir)
			defer os.Unsetenv("HOME")

			// Find config
			found := findConfigFileEnhanced()

			// Verify the highest priority config was found
			expectedPath := filepath.Join(testDir, config.path)
			if found != expectedPath {
				rel, _ := filepath.Rel(testDir, found)
				expectedRel, _ := filepath.Rel(testDir, expectedPath)
				t.Errorf("Expected to find %s (priority %d), but got %s",
					expectedRel, config.priority, rel)
			}
		})
	}
}

// Enhanced version of findConfigFile with all the improvements
// This would normally replace the existing function in config.go
func findConfigFileEnhanced() string {
	// Build candidate list in priority order
	var candidates []string

	// Current directory configs (highest priority)
	candidates = append(candidates,
		"./promptshield.yaml",
		"./promptshield.yml",
		"./.promptshieldrc.yaml",
		"./.promptshieldrc.yml",
		"./.promptshieldrc.json",
	)

	// Home directory configs: prefer HOME env var for testability, then UserHomeDir
	if home := os.Getenv("HOME"); home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".promptshield", "promptshield.yaml"),
			filepath.Join(home, ".promptshield", "config.yaml"),
		)
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(homeDir, ".promptshield", "promptshield.yaml"),
			filepath.Join(homeDir, ".promptshield", "config.yaml"),
		)
	}

	// XDG config directory (Linux standard)
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates,
			filepath.Join(xdg, "promptshield", "config.yaml"),
			filepath.Join(xdg, "promptshield", "promptshield.yaml"),
		)
	} else if home, err := os.UserHomeDir(); err == nil {
		// Default XDG location if XDG_CONFIG_HOME not set
		candidates = append(candidates,
			filepath.Join(home, ".config", "promptshield", "config.yaml"),
			filepath.Join(home, ".config", "promptshield", "promptshield.yaml"),
		)
	}

	// Check each candidate in order
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// Return absolute path for consistency
			if abs, err := filepath.Abs(path); err == nil {
				return abs
			}
			return path
		}
	}

	return ""
}
