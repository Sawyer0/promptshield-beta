package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/blang/semver/v4"
	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/shared/termui"
	"github.com/spf13/cobra"
)

type updateInfo struct {
	CurrentVersion  string `json:"current_version"`
	Commit          string `json:"commit"`
	BuildDate       string `json:"build_date"`
	LatestVersion   string `json:"latest_version,omitempty"`
	UpdateURL       string `json:"update_url"`
	Message         string `json:"message"`
	UpdateAvailable bool   `json:"update_available"`
}

var (
	doUpdate   bool
	prerelease bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and optionally install a newer version",
	Long: `Check for updates from GitHub releases with robust parsing and verification.
	
Checks GitHub releases (no auto-update in this build):
- Proper semver comparison  
- Safe instructions to download the latest binary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		deps := bootstrap.From(cmd)
		ctx := cmd.Context()

		// Set up logger for update process
		var logger *slog.Logger
		if deps != nil {
			logger = deps.Logger
		} else {
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		}

		latest, updateAvailable, err := checkForUpdate(ctx, logger)
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		info := updateInfo{
			CurrentVersion:  version,
			Commit:          commit,
			BuildDate:       buildDate,
			LatestVersion:   latest,
			UpdateURL:       "https://github.com/promptshield/promptshield/releases",
			UpdateAvailable: updateAvailable,
		}

		if updateAvailable {
			info.Message = "New version available"
			if doUpdate {
				info.Message = "Updating to latest version..."
			}
		} else {
			info.Message = "You are up to date"
		}

		// Telemetry: report update outcome (coarse)
		if deps != nil && deps.Telemetry != nil {
			outcome := "unknown"
			if latest != "" {
				if updateAvailable {
					outcome = "newer"
				} else {
					outcome = "same"
				}
			}
			deps.Telemetry.Collect("update_check", map[string]any{
				"outcome":     outcome,
				"auto_update": doUpdate,
			})
		}

		// Respect --output-format
		if jsonOutput || outputFormat == "json" || outputFormat == "ndjson" {
			enc := json.NewEncoder(os.Stdout)
			if outputFormat == "json" {
				enc.SetIndent("", "  ")
			}
			return enc.Encode(info)
		}

		// Human-readable output
		heading := fmt.Sprintf("promptshield %s (commit %s, built %s)", version, commit, buildDate)
		fmt.Fprintf(os.Stdout, "%s\n", termui.Heading(true, heading))

		if updateAvailable {
			fmt.Fprintf(os.Stdout, "%s\n", termui.Bullet(true, "New version available:", latest))
			if doUpdate {
				fmt.Fprintf(os.Stdout, "%s\n", termui.Bullet(true, "Updating...", ""))
				if err := performUpdate(ctx, logger); err != nil {
					return fmt.Errorf("update failed: %w", err)
				}
				fmt.Fprintf(os.Stdout, "%s\n", termui.Bullet(true, "Update completed successfully!", ""))
				fmt.Fprintf(os.Stdout, "%s\n", termui.Bullet(true, "Please restart the application to use the new version.", ""))
			} else {
				fmt.Fprintf(os.Stdout, "%s\n", termui.Bullet(true, "Run with --update to install automatically", ""))
			}
		} else {
			fmt.Fprintf(os.Stdout, "%s\n", termui.Bullet(true, info.Message, ""))
		}

		fmt.Fprintf(os.Stdout, "%s\n", termui.Bullet(true, "Releases:", info.UpdateURL))
		return nil
	},
}

func NewUpdateCommand(deps *bootstrap.Deps) *cobra.Command {
	_ = deps
	return updateCmd
}

func init() {
	updateCmd.Flags().BoolVar(&doUpdate, "update", false, "Automatically download and install the latest version")
	updateCmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include pre-release versions")

	// Hide update from help for v0.2.0 (not publicly released yet)
	updateCmd.Hidden = true
}

// checkForUpdate uses go-github-selfupdate for robust release checking
func checkForUpdate(ctx context.Context, logger *slog.Logger) (latestVersion string, updateAvailable bool, err error) {
	// Parse current version
	currentVer, err := semver.Parse(normalizeVersion(version))
	if err != nil {
		if logger != nil {
			logger.Debug("failed to parse current version", "version", version, "error", err)
		}
		// Fallback for development builds
		currentVer = semver.MustParse("0.0.0")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/promptshield/promptshield/releases", nil)
	req.Header.Set("User-Agent", "promptshield-update-check")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("contacting update server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("update server returned %s", resp.Status)
	}
	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", false, fmt.Errorf("parsing releases: %w", err)
	}
	if len(releases) == 0 {
		return currentVer.String(), false, nil
	}
	latestTag := releases[0].TagName
	latestSem, err := semver.Parse(normalizeVersion(latestTag))
	if err != nil {
		return currentVer.String(), false, nil
	}
	updateAvailable = latestSem.GT(currentVer)
	if logger != nil {
		logger.Debug("version check completed",
			"current", currentVer.String(),
			"latest", latestSem.String(),
			"update_available", updateAvailable)
	}
	return latestSem.String(), updateAvailable, nil
}

// performUpdate downloads and installs the latest version
func performUpdate(_ context.Context, _ *slog.Logger) error {
	return fmt.Errorf("automatic update is not enabled in this build; download the latest binary from https://github.com/promptshield/promptshield/releases")
}

// normalizeVersion converts version strings to semver-compatible format
func normalizeVersion(v string) string {
	// Remove 'v' prefix if present
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}
