package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/denysvitali/llm-usage/internal/app"
	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/usage"
	"github.com/spf13/cobra"
)

var watchInterval time.Duration

var watchCmd = &cobra.Command{
	Use: "watch", Short: "Continuously refresh usage", Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		mgr := credentials.NewManager()
		service := app.QueryService{Credentials: mgr}
		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()
		for {
			stats, err := service.Query(context.Background(), app.QueryOptions{Providers: providerFlag, Account: accountFlag, AllAccounts: allAccountsFlag, Debug: debugFlag, Timeout: timeoutFlag})
			if err != nil {
				return err
			}
			fmt.Print("\033[H\033[2J")
			usage.OutputPretty(stats)
			<-ticker.C
		}
	},
}

func init() {
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 5*time.Minute, "Refresh interval")
	rootCmd.AddCommand(watchCmd)
}
