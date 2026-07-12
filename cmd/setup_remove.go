package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/denysvitali/llm-usage/internal/setup"
)

var setupRemoveYes bool

var setupRemoveCmd = &cobra.Command{
	Use:   "remove <provider> <account>",
	Short: "Remove an account",
	Long:  `Remove an account from a provider.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runSetupRemove,
}

func init() {
	setupRemoveCmd.Flags().BoolVarP(&setupRemoveYes, "yes", "y", false, "Skip confirmation prompt")
	setupCmd.AddCommand(setupRemoveCmd)
}

func runSetupRemove(_ *cobra.Command, args []string) error {
	providerID := args[0]
	accountName := args[1]

	if !setupRemoveYes {
		fmt.Printf("Remove account '%s' from %s? [y/N]: ", accountName, providerID)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	mgr := getCredentialsManager()
	if err := setup.RemoveAccount(mgr, providerID, accountName); err != nil {
		return err
	}
	fmt.Printf("Successfully removed account '%s' from %s\n", accountName, providerID)
	return nil
}
