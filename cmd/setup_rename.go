package cmd

import (
	"fmt"

	"github.com/denysvitali/llm-usage/internal/setup"
	"github.com/spf13/cobra"
)

var setupRenameCmd = &cobra.Command{
	Use:   "rename <provider> <old-name> <new-name>",
	Short: "Rename an account",
	Long:  `Rename an account for a provider.`,
	Args:  cobra.ExactArgs(3),
	RunE:  runSetupRename,
}

func init() {
	setupCmd.AddCommand(setupRenameCmd)
}

func runSetupRename(_ *cobra.Command, args []string) error {
	providerID := args[0]
	oldName := args[1]
	newName := args[2]
	mgr := getCredentialsManager()
	if err := setup.RenameAccount(mgr, providerID, oldName, newName); err != nil {
		return err
	}
	fmt.Printf("Successfully renamed account '%s' to '%s' for %s\n", oldName, newName, providerID)
	return nil
}
