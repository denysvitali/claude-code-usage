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

var configExplainCmd = &cobra.Command{
	Use: "explain", Short: "Show the effective non-secret configuration", Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		path := effectiveConfigPath()
		cfg, err := config.LoadOptional(path)
		if err != nil {
			return err
		}
		fmt.Printf("Configuration: %s\n", path)
		if len(cfg.Providers) == 0 {
			fmt.Println("Providers: auto-discovered")
		} else {
			fmt.Println("Providers:")
			for _, p := range cfg.Providers {
				fmt.Printf("  - %s", p.ID)
				if len(p.Accounts) > 0 {
					fmt.Printf(" (%d account(s))", len(p.Accounts))
				}
				fmt.Println()
			}
		}
		fmt.Printf("Output: %s\n", defaultString(cfg.Defaults.Output.Format, "pretty"))
		fmt.Printf("Cache: %s\n", defaultString(cfg.Defaults.Cache.TTL, "disabled"))
		return nil
	},
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func effectiveConfigPath() string {
	if configPath != "" {
		return configPath
	}
	return config.DefaultPath()
}

func init() {
	configCmd.PersistentFlags().StringVar(&configPath, "file", "", "Configuration file path")
	configCmd.AddCommand(configInitCmd, configPathCmd, configValidateCmd, configExplainCmd)
	rootCmd.AddCommand(configCmd)
}
