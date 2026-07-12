package cmd

import (
	"github.com/spf13/cobra"

	"github.com/denysvitali/llm-usage/internal/setup"
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
