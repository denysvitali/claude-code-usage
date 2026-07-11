package cmd

import (
	"fmt"
	"os"

	"github.com/denysvitali/llm-usage/internal/config"
	"github.com/spf13/cobra"
)

var configPath string

var configCmd = &cobra.Command{Use: "config", Short: "Manage llm-usage configuration"}

var configInitCmd = &cobra.Command{
	Use: "init", Short: "Create a new configuration file", Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		path := effectiveConfigPath()
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists at %s", path)
		}
		cfg := config.Config{Version: "1", Providers: []config.ProviderConfig{{ID: "claude", Accounts: []config.Account{{Name: "default"}}}}}
		if err := config.Save(path, cfg); err != nil {
			return err
		}
		fmt.Printf("Created %s\n", path)
		return nil
	},
}

var configPathCmd = &cobra.Command{Use: "path", Short: "Print the configuration path", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error { fmt.Println(effectiveConfigPath()); return nil }}

var configValidateCmd = &cobra.Command{
	Use: "validate", Short: "Validate the configuration file", Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		path := effectiveConfigPath()
		var cfg config.Config
		if err := config.Load(path, &cfg); err != nil {
			return err
		}
		fmt.Printf("Configuration is valid: %s\n", path)
		return nil
	},
}

func effectiveConfigPath() string {
	if configPath != "" {
		return configPath
	}
	return config.DefaultPath()
}

func init() {
	configCmd.PersistentFlags().StringVar(&configPath, "file", "", "Configuration file path")
	configCmd.AddCommand(configInitCmd, configPathCmd, configValidateCmd)
	rootCmd.AddCommand(configCmd)
}
