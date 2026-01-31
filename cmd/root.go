// Package cmd provides the Cobra CLI commands for llm-usage.
package cmd

import (
	"fmt"
	"os"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/usage"
	"github.com/denysvitali/llm-usage/internal/version"
	"github.com/spf13/cobra"
)

var (
	providerFlag    string
	accountFlag     string
	allAccountsFlag bool
	jsonOutput      bool
	waybarOutput    bool
)

var rootCmd = &cobra.Command{
	Use:     "llm-usage",
	Short:   "Display LLM API usage statistics",
	Long:    `llm-usage displays API usage statistics across multiple LLM providers including Claude, Kimi, and Z.AI.`,
	Version: version.Version,
	RunE:    runUsage,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "all", "Provider: claude, kimi, zai, or all")
	rootCmd.Flags().StringVarP(&accountFlag, "account", "a", "", "Account to use")
	rootCmd.Flags().BoolVar(&allAccountsFlag, "all-accounts", false, "Aggregate usage across all accounts")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.Flags().BoolVar(&waybarOutput, "waybar", false, "Output in waybar JSON format")
}

func runUsage(_ *cobra.Command, _ []string) error {
	credsMgr := credentials.NewManager()

	// Determine which providers to query
	providers := usage.GetProviders(providerFlag, accountFlag, allAccountsFlag, credsMgr)
	if len(providers) == 0 {
		if waybarOutput {
			usage.OutputWaybarError("No providers configured")
			return nil
		}
		return fmt.Errorf("no providers configured. Run 'llm-usage setup' to configure providers")
	}

	// Fetch usage from all providers concurrently
	stats := usage.FetchAllUsage(providers)

	switch {
	case waybarOutput:
		usage.OutputWaybar(stats)
	case jsonOutput:
		usage.OutputJSON(stats)
	default:
		usage.OutputPretty(stats)
	}

	return nil
}
