package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/denysvitali/llm-usage/internal/credentials"
	setuptui "github.com/denysvitali/llm-usage/internal/setup/tui"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure LLM provider credentials",
	Long:  `Configure credentials for LLM providers. Run without subcommands to launch the interactive TUI wizard.`,
	RunE:  runSetupWizard,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetupWizard(_ *cobra.Command, _ []string) error {
	mgr := credentials.NewManager()
	p := tea.NewProgram(setuptui.NewModel(mgr))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

// getCredentialsManager returns a new credentials manager (used by subcommands)
func getCredentialsManager() *credentials.Manager {
	return credentials.NewManager()
}
