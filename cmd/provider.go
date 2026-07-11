package cmd

import (
	"fmt"

	"github.com/denysvitali/llm-usage/providers"
	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{Use: "provider", Short: "Inspect supported providers"}
var providerListCmd = &cobra.Command{
	Use: "list", Short: "List supported providers", Args: cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		for _, p := range providers.All() {
			status := "ready"
			if !p.Implemented {
				status = "unsupported"
			}
			fmt.Printf("%-8s %-22s %-18s %s\n", p.ID, p.Name, p.Auth, status)
		}
	},
}

func init() { providerCmd.AddCommand(providerListCmd); rootCmd.AddCommand(providerCmd) }
