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
	credentialsFile string
	debugFlag       bool
)

var rootCmd = &cobra.Command{
	Use:     "llm-usage",
	Short:   "Display LLM API usage statistics",
	Long:    `llm-usage displays API usage statistics across multiple LLM providers including Claude, Kimi, Z.AI, and MiniMax.`,
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
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "all", "Provider: claude, kimi, zai, minimax, or all")
	rootCmd.Flags().StringVarP(&accountFlag, "account", "a", "", "Account to use")
	rootCmd.Flags().BoolVar(&allAccountsFlag, "all-accounts", false, "Aggregate usage across all accounts")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.Flags().BoolVar(&waybarOutput, "waybar", false, "Output in waybar JSON format")
	rootCmd.Flags().StringVar(&credentialsFile, "credentials-file", "", "Path to a combined credentials file (values may use $VAR or ${VAR} env references)")
	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Include raw provider API responses in the output")
}

func runUsage(_ *cobra.Command, _ []string) error {
	var credsMgr *credentials.Manager
	if credentialsFile != "" {
		credsMgr = credentials.NewManagerFromFile(credentialsFile)
	} else {
		credsMgr = credentials.NewManager()
	}

	// Determine which providers to query
	providers := usage.GetProviders(providerFlag, accountFlag, allAccountsFlag, debugFlag, credsMgr)
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
