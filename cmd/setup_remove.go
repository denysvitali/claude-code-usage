package cmd

import (
	"fmt"

	"github.com/denysvitali/llm-usage/internal/setup"
	"github.com/spf13/cobra"
)

var setupRemoveCmd = &cobra.Command{
	Use:   "remove <provider> <account>",
	Short: "Remove an account",
	Long:  `Remove an account from a provider.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runSetupRemove,
}

func init() {
	setupCmd.AddCommand(setupRemoveCmd)
}

func runSetupRemove(_ *cobra.Command, args []string) error {
	providerID := args[0]
	accountName := args[1]
	mgr := getCredentialsManager()
	if err := setup.RemoveAccount(mgr, providerID, accountName); err != nil {
		return err
	}
	fmt.Printf("Successfully removed account '%s' from %s\n", accountName, providerID)
	return nil
}
