package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/denysvitali/llm-usage/internal/app"
	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/usage"
)

var watchInterval time.Duration

var watchCmd = &cobra.Command{
	Use: "watch", Short: "Continuously refresh usage", Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		mgr := credentials.NewManager()
		service := app.QueryService{Credentials: mgr}
		cfg, err := loadRuntimeConfig()
		if err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		opts, err := newQueryOptions(cfg)
		if err != nil {
			return err
		}
		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()
		for {
			stats, err := service.Query(context.Background(), opts)
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
