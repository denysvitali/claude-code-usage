package cmd

import (
	"github.com/denysvitali/llm-usage/internal/setup"
	"github.com/spf13/cobra"
)

var setupListCmd = &cobra.Command{
	Use:   "list [provider]",
	Short: "List configured accounts",
	Long:  `List all configured accounts, optionally filtered by provider.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSetupList,
}

func init() {
	setupCmd.AddCommand(setupListCmd)
}

func runSetupList(_ *cobra.Command, args []string) error {
	providerID := ""
	if len(args) > 0 {
		providerID = args[0]
	}
	mgr := getCredentialsManager()
	return setup.ListAccounts(mgr, providerID)
}
