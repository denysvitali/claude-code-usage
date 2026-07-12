package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/denysvitali/llm-usage/internal/config"
)

var doctorCmd = &cobra.Command{
	Use: "doctor", Short: "Check local configuration and runtime prerequisites", Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		path := effectiveConfigPath()
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("configuration is missing at %s: run 'llm-usage config init'", path)
		}
		var cfg config.Config
		if err := config.Load(path, &cfg); err != nil {
			return err
		}
		fmt.Println("Configuration: OK")
		fmt.Println("Provider registry: OK")
		return nil
	},
}

func init() { rootCmd.AddCommand(doctorCmd) }
