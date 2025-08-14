package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/security/cred"
	"github.com/promptshield/promptshield/internal/shared/termui"
	"github.com/spf13/cobra"
)

func NewAuthCommand(deps *bootstrap.Deps) *cobra.Command {
	_ = deps
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(os.Stdout, termui.Heading(true, "Usage"))
			fmt.Fprintln(os.Stdout, "  promptshield auth set <provider>")
			fmt.Fprintln(os.Stdout, "  Providers: openai, anthropic")
			return nil
		},
	}
	cmd.AddCommand(newAuthSetCommand())
	return cmd
}

func newAuthSetCommand() *cobra.Command {
	var provider string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Store a provider API key in your OS keychain",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p := strings.ToLower(strings.TrimSpace(provider))
			if p != string(cred.ProviderOpenAI) && p != string(cred.ProviderAnthropic) {
				return fmt.Errorf("unsupported provider: %s (use openai or anthropic)", p)
			}
			fmt.Fprintf(os.Stdout, "%s\n", termui.Heading(true, fmt.Sprintf("Saving %s API key", p)))
			fmt.Fprint(os.Stdout, termui.Bullet(true, "Enter API key (input hidden): ", ""))
			var key string
			if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
				fmt.Fprint(os.Stdout, "\n> ")
				b, _ := term.ReadPassword(fd)
				fmt.Fprintln(os.Stdout)
				key = strings.TrimSpace(string(b))
			} else {
				fmt.Fprint(os.Stdout, "\n> ")
				reader := bufio.NewReader(os.Stdin)
				k, _ := reader.ReadString('\n')
				key = strings.TrimSpace(k)
			}
			if key == "" {
				return fmt.Errorf("api key cannot be empty")
			}
			if err := cred.SaveProviderAPIKey(context.Background(), cred.Provider(p), key); err != nil {
				return fmt.Errorf("saving api key: %w", err)
			}
			fmt.Fprintln(os.Stdout, termui.Bullet(true, "Saved to OS keychain", ""))
			return nil
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "provider: openai|anthropic")
	_ = cmd.MarkFlagRequired("provider")
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"openai", "anthropic"}, cobra.ShellCompDirectiveNoFileComp
	}
	return cmd
}
