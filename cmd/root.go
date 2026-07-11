// Package cmd provides the Cobra CLI commands for llm-usage.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/denysvitali/llm-usage/internal/app"
	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/usage"
	"github.com/denysvitali/llm-usage/internal/version"
	"github.com/denysvitali/llm-usage/provider"
	registry "github.com/denysvitali/llm-usage/providers"
	"github.com/spf13/cobra"
)

var (
	providerFlag    string
	accountFlag     string
	allAccountsFlag bool
	jsonOutput      bool
	waybarOutput    bool
	credentialsFile string
	debugFlag       bool
	timeoutFlag     time.Duration
)

// validProviders lists the provider IDs accepted by --provider, derived from
// the central provider registry.
var validProviders = func() []string {
	available := registry.All()
	ids := make([]string, 0, len(available))
	for _, p := range available {
		ids = append(ids, p.ID)
	}
	return ids
}()

var rootCmd = &cobra.Command{
	Use:   "llm-usage",
	Short: "Display LLM API usage statistics",
	Long:  `llm-usage displays API usage statistics across Claude, Codex, Kimi, Z.AI, and MiniMax.`,
	Example: `  # Show all configured providers
  llm-usage

  # Show one or more specific providers
  llm-usage --provider=claude
  llm-usage --provider=claude,kimi

  # Show a specific account
  llm-usage --provider=kimi --account=work

  # Machine-readable output
  llm-usage --json
  llm-usage --waybar`,
	Version:       version.Version,
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE:          runUsage,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "all", "Provider(s), comma-separated: claude, codex, grok, kimi, zai, minimax, or all")
	rootCmd.Flags().StringVarP(&accountFlag, "account", "a", "", "Account to use")
	rootCmd.Flags().BoolVar(&allAccountsFlag, "all-accounts", false, "Aggregate usage across all accounts")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.Flags().BoolVar(&waybarOutput, "waybar", false, "Output in waybar JSON format")
	rootCmd.Flags().StringVar(&credentialsFile, "credentials-file", "", "Path to a combined credentials file (values may use $VAR or ${VAR} env references)")
	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Include raw provider API responses in the output")
	rootCmd.Flags().DurationVar(&timeoutFlag, "timeout", 30*time.Second, "Maximum time to wait for provider responses")

	_ = rootCmd.RegisterFlagCompletionFunc("provider", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return append([]string{"all"}, validProviders...), cobra.ShellCompDirectiveNoFileComp
	})
	rootCmd.AddCommand(&cobra.Command{Use: "completion [bash|zsh|fish|powershell]", Short: "Generate shell completion", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell %q", args[0])
		}
	}})
}

func runUsage(_ *cobra.Command, _ []string) error {
	var credsMgr *credentials.Manager
	if credentialsFile != "" {
		credsMgr = credentials.NewManagerFromFile(credentialsFile)
	} else {
		credsMgr = credentials.NewManager()
	}

	stats, err := (app.QueryService{Credentials: credsMgr}).Query(context.Background(), app.QueryOptions{
		Providers: providerFlag, Account: accountFlag, AllAccounts: allAccountsFlag,
		Debug: debugFlag, Timeout: timeoutFlag,
	})
	if err != nil {
		if waybarOutput {
			usage.OutputWaybarError("No providers configured")
			return nil
		}
		return fmt.Errorf("%w. Run 'llm-usage config init' to configure providers", err)
	}

	switch {
	case waybarOutput:
		usage.OutputWaybar(stats)
	case jsonOutput:
		usage.OutputJSON(stats)
	default:
		usage.OutputPretty(stats)
	}

	// Signal failure to scripts when every provider errored (waybar output
	// must always exit 0 so the bar module keeps rendering).
	if !waybarOutput && allFailed(stats.Providers) {
		return fmt.Errorf("all selected providers failed")
	}

	return nil
}

// allFailed reports whether every provider entry carries an error.
func allFailed(providers []provider.Usage) bool {
	for _, p := range providers {
		if p.Error == nil {
			return false
		}
	}
	return true
}
