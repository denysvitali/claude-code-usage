// Package cmd provides the Cobra CLI commands for llm-usage.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/denysvitali/llm-usage/internal/app"
	"github.com/denysvitali/llm-usage/internal/config"
	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/usage"
	"github.com/denysvitali/llm-usage/internal/version"
	"github.com/denysvitali/llm-usage/provider"
	registry "github.com/denysvitali/llm-usage/providers"
)

var (
	providerFlag    string
	accountFlag     string
	allAccountsFlag bool
	jsonOutput      bool
	waybarOutput    bool
	rawOutput       bool
	credentialsFile string
	debugFlag       bool
	timeoutFlag     time.Duration
	cacheTTLFlag    time.Duration
	staleIfError    bool
	configFileFlag  string
)

const (
	outputWaybar = "waybar"
	outputJSON   = "json"
	outputRaw    = "raw"
)

// validProviders lists the provider IDs accepted by --provider, derived from
// the central provider registry.
var validProviders = func() []string {
	available := registry.All()
	ids := make([]string, 0, len(available))
	for _, p := range available {
		ids = append(ids, p.ID)
	}
	return ids
}()

var rootCmd = &cobra.Command{
	Use:   "llm-usage",
	Short: "Display LLM API usage statistics",
	Long:  `llm-usage displays API usage statistics across Claude, Codex, Grok, Kimi, and MiniMax.`,
	Example: `  # Show all configured providers
  llm-usage

  # Show one or more specific providers
  llm-usage --provider=claude
  llm-usage --provider=claude,kimi

  # Show a specific account
  llm-usage --provider=kimi --account=work

  # Machine-readable output
  llm-usage --json
  llm-usage --waybar`,
	Version:       version.Version,
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE:          runUsage,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&providerFlag, "provider", "p", "all", "Provider(s), comma-separated: claude, codex, grok, kimi, minimax, or all")
	rootCmd.Flags().StringVarP(&accountFlag, "account", "a", "", "Account to use")
	rootCmd.Flags().BoolVar(&allAccountsFlag, "all-accounts", false, "Aggregate usage across all accounts")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.Flags().BoolVar(&waybarOutput, "waybar", false, "Output in waybar JSON format")
	rootCmd.Flags().BoolVar(&rawOutput, "raw", false, "Output the raw provider API responses in JSON format")
	rootCmd.Flags().StringVar(&credentialsFile, "credentials-file", "", "Path to a combined credentials file (values may use $VAR or ${VAR} env references)")
	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Include raw provider API responses in the output")

	rootCmd.MarkFlagsMutuallyExclusive("json", "waybar", "raw")
	rootCmd.Flags().DurationVar(&timeoutFlag, "timeout", 30*time.Second, "Maximum time to wait for provider responses")
	rootCmd.Flags().DurationVar(&cacheTTLFlag, "cache-ttl", 0, "Cache successful usage responses for this duration (disabled by default)")
	rootCmd.Flags().BoolVar(&staleIfError, "stale-if-error", false, "Use expired cache data when a provider request fails")
	rootCmd.Flags().StringVar(&configFileFlag, "config", "", "Configuration file path (default: XDG config path)")

	_ = rootCmd.RegisterFlagCompletionFunc("provider", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return append([]string{"all"}, validProviders...), cobra.ShellCompDirectiveNoFileComp
	})
	rootCmd.AddCommand(&cobra.Command{Use: "completion [bash|zsh|fish|powershell]", Short: "Generate shell completion", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell %q", args[0])
		}
	}})
}

func runUsage(_ *cobra.Command, _ []string) error {
	var credsMgr *credentials.Manager
	if credentialsFile != "" {
		credsMgr = credentials.NewManagerFromFile(credentialsFile)
	} else {
		credsMgr = credentials.NewManager()
	}

	cfg, err := loadRuntimeConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	opts, err := newQueryOptions(cfg)
	if err != nil {
		return err
	}
	stats, err := (app.QueryService{Credentials: credsMgr}).Query(context.Background(), opts)
	if err != nil {
		if waybarOutput {
			usage.OutputWaybarError("No providers configured")
			return nil
		}
		return fmt.Errorf("%w. Run 'llm-usage config init' to configure providers", err)
	}

	format := cfg.Defaults.Output.Format
	switch {
	case waybarOutput:
		format = outputWaybar
	case jsonOutput:
		format = outputJSON
	case rawOutput:
		format = outputRaw
	}
	isWaybar := format == outputWaybar
	switch format {
	case outputWaybar:
		usage.OutputWaybar(stats)
	case outputJSON:
		usage.OutputJSON(stats)
	case outputRaw:
		usage.OutputRaw(stats)
	default:
		usage.OutputPretty(stats)
	}

	// Signal failure to scripts when every provider errored (waybar output
	// must always exit 0 so the bar module keeps rendering).
	if !isWaybar && allFailed(stats.Providers) {
		return fmt.Errorf("all selected providers failed")
	}

	return nil
}

func loadRuntimeConfig() (*config.Config, error) {
	path := configFileFlag
	if path == "" {
		path = config.DefaultPath()
	}
	return config.LoadOptional(path)
}

func newQueryOptions(cfg *config.Config) (app.QueryOptions, error) {
	ttl := cacheTTLFlag
	if ttl == 0 && cfg != nil && cfg.Defaults.Cache.TTL != "" {
		parsed, err := time.ParseDuration(cfg.Defaults.Cache.TTL)
		if err != nil {
			return app.QueryOptions{}, fmt.Errorf("parse cache TTL: %w", err)
		}
		ttl = parsed
	}
	stale := staleIfError || (cfg != nil && cfg.Defaults.Cache.StaleIfError)
	return app.QueryOptions{
		Providers: providerFlag, Account: accountFlag, AllAccounts: allAccountsFlag,
		Debug: debugFlag, Raw: debugFlag || rawOutput, Timeout: timeoutFlag, CacheTTL: ttl, StaleIfError: stale, Config: cfg,
	}, nil
}

// allFailed reports whether every provider entry carries an error.
func allFailed(providers []provider.Usage) bool {
	for _, p := range providers {
		if p.Error == nil {
			return false
		}
	}
	return true
}
