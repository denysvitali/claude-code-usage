package cmd

import (
	"github.com/spf13/cobra"

	"github.com/denysvitali/llm-usage/internal/setup"
)

var setupAddAccountName string

var setupAddCmd = &cobra.Command{
	Use:   "add <provider>",
	Short: "Add an account for a provider",
	Long:  `Add a new account for a provider (claude, kimi, or minimax).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSetupAdd,
}

func init() {
	setupAddCmd.Flags().StringVar(&setupAddAccountName, "account", "", "Account name (default: prompt interactively)")
	setupCmd.AddCommand(setupAddCmd)
}

func runSetupAdd(_ *cobra.Command, args []string) error {
	providerID := args[0]
	mgr := getCredentialsManager()
	return setup.AddAccount(mgr, providerID, setupAddAccountName)
}
