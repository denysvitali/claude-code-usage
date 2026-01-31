package cmd

import (
	"github.com/denysvitali/llm-usage/internal/setup"
	"github.com/spf13/cobra"
)

var setupMigrateCmd = &cobra.Command{
	Use:   "migrate-claude",
	Short: "Migrate credentials from Claude CLI",
	Long:  `Migrate OAuth credentials from the Claude CLI (~/.claude/.credentials.json) to llm-usage.`,
	Args:  cobra.NoArgs,
	RunE:  runSetupMigrate,
}

func init() {
	setupCmd.AddCommand(setupMigrateCmd)
}

func runSetupMigrate(_ *cobra.Command, _ []string) error {
	mgr := getCredentialsManager()
	return setup.MigrateClaudeCLI(mgr)
}
