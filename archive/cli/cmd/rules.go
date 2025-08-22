package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rulesapp "github.com/promptshield/promptshield/internal/application/rules"
	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/spf13/cobra"
)

var (
	rulesInitDest     string
	rulesInitForce    bool
	rulesInitTemplate string
	rulesListPath     string
	rulesValidatePath string
	rulesValidateJSON bool
)

var rulesInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a skeleton RulePack",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = bootstrap.From(cmd) // reserved for future DI usage
		svc := rulesapp.NewService()
		if rulesInitTemplate == "" {
			rulesInitTemplate = "l1"
		}
		dest, err := svc.InitPackWithTemplate(rulesInitDest, rulesInitForce, rulesInitTemplate)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, dest)
		return nil
	},
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List rules in a RulePack file or directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = bootstrap.From(cmd)
		if rulesListPath == "" {
			rulesListPath = "rules"
		}
		svc := rulesapp.NewService()
		ids, err := svc.List(rulesListPath)
		if err != nil {
			return err
		}
		for _, id := range ids {
			fmt.Fprintln(os.Stdout, id)
		}
		return nil
	},
}

// NewRulesListCommand exposes rules:list with alias rl.
func NewRulesListCommand(deps *bootstrap.Deps) *cobra.Command {
	_ = deps
	c := &cobra.Command{Use: "rules:list", Aliases: []string{"rl"}, Short: "List rules in a RulePack file or directory",
		Example: "  promptshield rules:list --path rules\n  promptshield rules:list --path ./packs/security.yaml",
	}
	c.RunE = rulesListCmd.RunE
	c.Flags().StringVarP(&rulesListPath, "path", "p", "rules", "RulePack file or directory")
	return c
}

// NewRulesCreateCommand exposes rules:create with alias rc.
func NewRulesCreateCommand(deps *bootstrap.Deps) *cobra.Command {
	_ = deps
	c := &cobra.Command{Use: "rules:create", Aliases: []string{"rc"}, Short: "Create a skeleton RulePack",
		Example: "  promptshield rules:create --dest rules/example.yaml --template l1\n  promptshield rules:create --dest rules/sem.yaml --template l3\n  promptshield rules:create -f --dest rules/pii.yaml --template pii",
	}
	c.RunE = rulesInitCmd.RunE
	c.Flags().StringVarP(&rulesInitDest, "dest", "o", "rules/example.yaml", "destination file path")
	c.Flags().BoolVarP(&rulesInitForce, "force", "f", false, "overwrite if exists")
	c.Flags().StringVar(&rulesInitTemplate, "template", "l1", "template: l1|l2|l3|pii|prompt-injection|industry")
	return c
}

// NewInitCommand scaffolds a starter promptshield.yaml and optional demo files for fast onboarding.
func NewInitCommand(deps *bootstrap.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup to create a promptshield.yaml and select RulePacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Non-interactive minimal scaffold: create promptshield.yaml if missing
			if _, err := os.Stat("promptshield.yaml"); err == nil {
				fmt.Fprintln(os.Stdout, "promptshield.yaml already exists")
				return nil
			}
			// Wizard: if TTY and not disabled, guide user to select
			useWizard, _ := cmd.Flags().GetBool("wizard")
			isTTY := false
			if fi, err := os.Stdin.Stat(); err == nil {
				// If input is a character device, assume TTY
				isTTY = (fi.Mode() & os.ModeCharDevice) != 0
			}
			var rp string
			if useWizard && isTTY {
				// Discover rulepack candidates
				cands := discoverRulepacks()
				// Present choices
				fmt.Fprintln(os.Stdout, "Select a RulePack (press Enter for demo):")
				for i, c := range cands {
					fmt.Fprintf(os.Stdout, "  [%d] %s\n", i+1, c)
				}
				fmt.Fprint(os.Stdout, "> ")
				var choice string
				fmt.Fscanln(os.Stdin, &choice)
				if choice != "" {
					idx := -1
					// parse 1-based index
					for i := range cands {
						if choice == fmt.Sprintf("%d", i+1) {
							idx = i
							break
						}
					}
					if idx >= 0 {
						rp = cands[idx]
					}
				}
				// Ask output format
				fmt.Fprintln(os.Stdout, "Choose output format [stylish|json] (default stylish):")
				fmt.Fprint(os.Stdout, "> ")
				var outfmt string
				fmt.Fscanln(os.Stdin, &outfmt)
				if strings.TrimSpace(outfmt) == "" {
					outfmt = "stylish"
				} else if strings.ToLower(outfmt) != "stylish" && strings.ToLower(outfmt) != "json" {
					outfmt = "stylish"
				}
				// Optional: workers
				fmt.Fprintln(os.Stdout, "Workers (0=auto, Enter to accept 0):")
				fmt.Fprint(os.Stdout, "> ")
				var workersStr string
				fmt.Fscanln(os.Stdin, &workersStr)
				content := "output_format: " + outfmt + "\n"
				if rp != "" {
					content += "rulepack: " + rp + "\n"
				}
				if strings.TrimSpace(workersStr) != "" {
					content += "workers: " + workersStr + "\n"
				}
				if err := os.WriteFile("promptshield.yaml", []byte(content), 0o644); err != nil {
					return fmt.Errorf("writing promptshield.yaml: %w", err)
				}
				fmt.Fprintln(os.Stdout, "Created promptshield.yaml")
				return nil
			}
			// Prefer built-in rules directory if a single pack exists
			rp = ""
			if entries, err := os.ReadDir("rules"); err == nil {
				var cands []string
				for _, e := range entries {
					if !e.IsDir() {
						name := strings.ToLower(e.Name())
						if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
							cands = append(cands, filepath.ToSlash(filepath.Join("rules", e.Name())))
						}
					}
				}
				if len(cands) == 1 {
					rp = cands[0]
				}
			}
			if rp == "" {
				// Fallback to demo rules
				if err := os.MkdirAll("demo", 0o755); err == nil {
					_ = os.WriteFile("demo/rules.yaml", []byte(demoRulesYAML), 0o600)
					rp = filepath.ToSlash(filepath.Join("demo", "rules.yaml"))
				}
			}
			content := "output_format: stylish\n"
			if rp != "" {
				content += "rulepack: " + rp + "\n"
			}
			if err := os.WriteFile("promptshield.yaml", []byte(content), 0o644); err != nil {
				return fmt.Errorf("writing promptshield.yaml: %w", err)
			}
			fmt.Fprintln(os.Stdout, "Created promptshield.yaml")
			if rp == "" {
				fmt.Fprintln(os.Stdout, "Tip: add a RulePack path under 'rulepack:' or run 'promptshield demo' to generate sample rules and data.")
			}
			return nil
		},
	}
	cmd.Flags().Bool("wizard", true, "run interactive setup when in a terminal")
	return cmd
}

func discoverRulepacks() []string {
	var cands []string
	entries, err := os.ReadDir("rules")
	if err != nil {
		return cands
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			cands = append(cands, filepath.ToSlash(filepath.Join("rules", e.Name())))
		}
	}
	return cands
}

// NewRulesValidateCommand exposes rules:validate (keeps name consistent) for convenience.
func NewRulesValidateCommand(deps *bootstrap.Deps) *cobra.Command {
	_ = deps
	c := &cobra.Command{Use: "rules:validate", Short: "Validate RulePack files with helpful messages",
		Example: "  promptshield rules:validate --path rules\n  promptshield rules:validate --json --path rules/security.yaml",
	}
	c.RunE = rulesValidateCmd.RunE
	c.Flags().StringVarP(&rulesValidatePath, "path", "p", "rules", "RulePack file or directory")
	c.Flags().BoolVar(&rulesValidateJSON, "json", false, "emit validation errors as JSON array")
	return c
}

// rules validate
var rulesValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate RulePack files with helpful messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = bootstrap.From(cmd)
		if rulesValidatePath == "" {
			rulesValidatePath = "rules"
		}
		packs, err := rules.LoadPacks(rulesValidatePath)
		if err != nil {
			return err
		}
		type vErr struct {
			File string `json:"file"`
			Err  string `json:"error"`
		}
		var all []vErr
		for _, p := range packs {
			errs := rules.ValidatePack(p)
			for _, e := range errs {
				all = append(all, vErr{File: p.SourcePath, Err: e.Error()})
			}
		}
		if len(all) == 0 {
			fmt.Fprintln(os.Stdout, "OK: all rule packs valid")
			return nil
		}
		if rulesValidateJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(all)
		}
		// Human-friendly output with grouped errors
		byFile := map[string][]string{}
		for _, ve := range all {
			byFile[ve.File] = append(byFile[ve.File], ve.Err)
		}
		for file, msgs := range byFile {
			fmt.Fprintf(os.Stderr, "Invalid: %s\n", file)
			for _, m := range msgs {
				// Improve common cases with suggestions
				hint := suggestionFor(m)
				if hint != "" {
					fmt.Fprintf(os.Stderr, "  - %s\n    → %s\n", m, hint)
				} else {
					fmt.Fprintf(os.Stderr, "  - %s\n", m)
				}
			}
		}
		return errors.New("rulepack validation failed")
	},
}

func suggestionFor(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "missing apiversion"):
		return "Add 'apiVersion: promptshield.io/v1' to the pack header."
	case strings.Contains(lower, "invalid kind"):
		return "Set 'kind: RulePack'"
	case strings.Contains(lower, "no rules defined"):
		return "Add at least one rule under 'rules:'"
	case strings.Contains(lower, "duplicate rule id"):
		return "Ensure each rule has a unique 'id' across merged packs."
	case strings.Contains(lower, "requires non-empty keywords"):
		return "Add one or more 'keywords' for level 1 rules."
	case strings.Contains(lower, "requires at least one regex") || strings.Contains(lower, "requires at least one regex pattern"):
		return "Provide 'patterns:' with one or more regex entries for level 2 rules."
	case strings.Contains(lower, "regex error"):
		return "Verify 'regex' syntax and allowed flags: ignorecase(i), multiline(m)."
	case strings.Contains(lower, "invalid timeout"):
		return "Use Go duration format, e.g., '50ms', '1s', '2m'."
	case strings.Contains(lower, "requires semantic configuration"):
		return "Provide 'semantic:' configuration for level 3 rules."
	}
	return ""
}
