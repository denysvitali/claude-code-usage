// Package main provides the CLI for llm-usage.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/provider"
	"github.com/denysvitali/llm-usage/internal/provider/claude"
	"github.com/denysvitali/llm-usage/internal/provider/kimi"
	"github.com/denysvitali/llm-usage/internal/provider/zai"
	"github.com/denysvitali/llm-usage/internal/setup"
	setuptui "github.com/denysvitali/llm-usage/internal/setup/tui"
)

// loadClaudeFromKeychain tries to load Claude credentials from the CLI keychain location
func loadClaudeFromKeychain() (*credentials.OAuthCredentials, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", err
	}

	credPath := homeDir + "/.claude/.credentials.json"
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, "", err
	}

	var result struct {
		ClaudeAiOauth *credentials.OAuthCredentials `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, "", err
	}

	if result.ClaudeAiOauth == nil || result.ClaudeAiOauth.AccessToken == "" {
		return nil, "", fmt.Errorf("no valid credentials in keychain")
	}

	return result.ClaudeAiOauth, "default", nil
}

const (
	barWidth = 20
	barFull  = "█"
	barEmpty = "░"
)

var (
	// Version is set by goreleaser
	version = "dev"
)

func main() {
	// Check for setup subcommand first
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		handleSetupCommand()
		return
	}

	// Main flags
	providerFlag := flag.String("provider", "all", "Provider to show: claude, kimi, zai, or all")
	accountFlag := flag.String("account", "", "Account to use (default: show all accounts)")
	allAccountsFlag := flag.Bool("all-accounts", false, "Aggregate usage across all accounts (implicit when --account is not specified)")
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	waybarOutput := flag.Bool("waybar", false, "Output in waybar JSON format")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("llm-usage %s\n", version)
		os.Exit(0)
	}

	credsMgr := credentials.NewManager()

	// Determine which providers to query
	providers := getProviders(*providerFlag, *accountFlag, *allAccountsFlag, credsMgr)
	if len(providers) == 0 {
		if *waybarOutput {
			outputWaybarError("No providers configured")
			return
		}
		fmt.Fprintf(os.Stderr, "Error: No providers configured. Run 'llm-usage setup' to configure providers.\n")
		os.Exit(1)
	}

	// Fetch usage from all providers concurrently
	stats := fetchAllUsage(providers)

	switch {
	case *waybarOutput:
		outputWaybarMulti(stats)
	case *jsonOutput:
		outputJSONMulti(stats)
	default:
		outputPrettyMulti(stats)
	}
}

// handleSetupCommand handles the setup subcommand and its sub-subcommands
func handleSetupCommand() {
	if len(os.Args) < 3 {
		// Run interactive TUI setup wizard
		mgr := credentials.NewManager()
		p := tea.NewProgram(setuptui.NewModel(mgr))
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	subcommand := os.Args[2]
	mgr := credentials.NewManager()

	switch subcommand {
	case "add":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Usage: llm-usage setup add <provider> [--account <name>]\n")
			os.Exit(1)
		}
		providerID := os.Args[3]
		accountName := ""
		// Parse optional --account flag
		for i := 4; i < len(os.Args); i++ {
			if os.Args[i] == "--account" && i+1 < len(os.Args) {
				accountName = os.Args[i+1]
				break
			}
		}
		if err := setup.AddAccount(mgr, providerID, accountName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "list":
		providerID := ""
		if len(os.Args) >= 4 {
			providerID = os.Args[3]
		}
		if err := setup.ListAccounts(mgr, providerID); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "remove":
		if len(os.Args) < 5 {
			fmt.Fprintf(os.Stderr, "Usage: llm-usage setup remove <provider> <account>\n")
			os.Exit(1)
		}
		providerID := os.Args[3]
		accountName := os.Args[4]
		if err := setup.RemoveAccount(mgr, providerID, accountName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully removed account '%s' from %s\n", accountName, providerID)

	case "rename":
		if len(os.Args) < 6 {
			fmt.Fprintf(os.Stderr, "Usage: llm-usage setup rename <provider> <old-name> <new-name>\n")
			os.Exit(1)
		}
		providerID := os.Args[3]
		oldName := os.Args[4]
		newName := os.Args[5]
		if err := setup.RenameAccount(mgr, providerID, oldName, newName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully renamed account '%s' to '%s' for %s\n", oldName, newName, providerID)

	case "migrate-claude":
		if err := setup.MigrateClaudeCLI(mgr); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown setup subcommand: %s\n", subcommand)
		fmt.Fprintf(os.Stderr, "\nUsage: llm-usage setup [<command>]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  (no args)          Run interactive setup wizard\n")
		fmt.Fprintf(os.Stderr, "  add <provider>    Add an account for a provider\n")
		fmt.Fprintf(os.Stderr, "  list [<provider>] List configured accounts\n")
		fmt.Fprintf(os.Stderr, "  remove <p> <acc> Remove an account\n")
		fmt.Fprintf(os.Stderr, "  rename <p> <old> <new>\n")
		fmt.Fprintf(os.Stderr, "                     Rename an account\n")
		fmt.Fprintf(os.Stderr, "  migrate-claude     Migrate from Claude CLI\n")
		os.Exit(1)
	}
}

// ProviderInstance holds a provider instance along with its account info
type ProviderInstance struct {
	provider.Provider
	AccountName string
}

// getProviders returns the list of providers to query based on the flag
func getProviders(providerFlag, accountFlag string, allAccounts bool, credsMgr *credentials.Manager) []ProviderInstance {
	var providerIDs []string

	if providerFlag == "all" {
		// Show all configured providers
		providerIDs = credsMgr.ListAvailable()
		// If no providers are configured, default to claude
		if len(providerIDs) == 0 {
			providerIDs = []string{"claude"}
		}
	} else {
		providerIDs = strings.Split(providerFlag, ",")
	}

	var providers []ProviderInstance
	for _, pid := range providerIDs {
		pid = strings.TrimSpace(pid)
		switch pid {
		case "claude":
			// Try loading from keychain first (Claude CLI location)
			keychainCreds, keychainAccount, keychainErr := loadClaudeFromKeychain()

			// Also try loading from the new multi-account location
			multiCreds, multiErr := credsMgr.LoadClaude()

			// Determine which source to use
			if keychainErr != nil && multiErr != nil {
				// Neither source available, skip
				continue
			}

			// If a specific account is requested, only use the multi-account location
			if accountFlag != "" {
				if multiErr != nil {
					continue
				}
				oauth := multiCreds.GetAccount(accountFlag)
				if oauth == nil || claude.IsExpired(oauth.ExpiresAt) {
					continue
				}
				providers = append(providers, ProviderInstance{
					Provider:    claude.NewProvider(oauth.AccessToken),
					AccountName: accountFlag,
				})
				continue
			}

			// No specific account requested - show all available
			// Add from keychain if available
			if keychainErr == nil && !claude.IsExpired(keychainCreds.ExpiresAt) {
				providers = append(providers, ProviderInstance{
					Provider:    claude.NewProvider(keychainCreds.AccessToken),
					AccountName: keychainAccount,
				})
			}
			// Add from multi-account location if available
			if multiErr == nil {
				for _, accName := range multiCreds.ListAccounts() {
					// Skip if this was already added from keychain
					if keychainErr == nil && accName == "default" {
						continue
					}
					oauth := multiCreds.GetAccount(accName)
					if oauth == nil || claude.IsExpired(oauth.ExpiresAt) {
						continue
					}
					providers = append(providers, ProviderInstance{
						Provider:    claude.NewProvider(oauth.AccessToken),
						AccountName: accName,
					})
				}
			}
		case "kimi":
			creds, err := credsMgr.LoadKimi()
			if err != nil {
				continue
			}
			if allAccounts || accountFlag == "" {
				// Add all accounts when --all-accounts is set or no specific account requested
				for _, accName := range creds.ListAccounts() {
					acc := creds.GetAccount(accName)
					if acc == nil {
						continue
					}
					providers = append(providers, ProviderInstance{
						Provider:    kimi.NewProvider(acc.APIKey),
						AccountName: accName,
					})
				}
			} else {
				// Use specified account
				acc := creds.GetAccount(accountFlag)
				if acc == nil {
					continue
				}
				providers = append(providers, ProviderInstance{
					Provider:    kimi.NewProvider(acc.APIKey),
					AccountName: accountFlag,
				})
			}
		case "zai":
			creds, err := credsMgr.LoadZAi()
			if err != nil {
				continue
			}
			if allAccounts || accountFlag == "" {
				// Add all accounts when --all-accounts is set or no specific account requested
				for _, accName := range creds.ListAccounts() {
					acc := creds.GetAccount(accName)
					if acc == nil {
						continue
					}
					providers = append(providers, ProviderInstance{
						Provider:    zai.NewProvider(acc.APIKey),
						AccountName: accName,
					})
				}
			} else {
				// Use specified account
				acc := creds.GetAccount(accountFlag)
				if acc == nil {
					continue
				}
				providers = append(providers, ProviderInstance{
					Provider:    zai.NewProvider(acc.APIKey),
					AccountName: accountFlag,
				})
			}
		}
	}

	return providers
}

// fetchAllUsage fetches usage from all providers concurrently
func fetchAllUsage(providers []ProviderInstance) *provider.UsageStats {
	var wg sync.WaitGroup
	var mu sync.Mutex

	stats := &provider.UsageStats{
		Providers: make([]provider.Usage, len(providers)),
	}

	for i, p := range providers {
		wg.Add(1)
		go func(idx int, prov ProviderInstance) {
			defer wg.Done()

			usage, err := prov.GetUsage()
			if err != nil {
				mu.Lock()
				stats.Providers[idx] = *provider.NewUsageError(prov.ID(), prov.Name(), err)
				mu.Unlock()
				return
			}

			// Add account name to usage if available
			if prov.AccountName != "" {
				if usage.Extra == nil {
					usage.Extra = make(map[string]any)
				}
				usage.Extra["account"] = prov.AccountName
			}

			mu.Lock()
			stats.Providers[idx] = *usage
			mu.Unlock()
		}(i, p)
	}

	wg.Wait()

	// Filter out empty providers (from concurrent initialization)
	var filtered []provider.Usage
	for _, p := range stats.Providers {
		if p.Provider != "" {
			filtered = append(filtered, p)
		}
	}
	stats.Providers = filtered

	return stats
}

// WaybarOutput represents the JSON format expected by waybar custom modules
type WaybarOutput struct {
	Text       string `json:"text"`
	Tooltip    string `json:"tooltip"`
	Class      string `json:"class"`
	Percentage int    `json:"percentage"`
}

func outputWaybarMulti(stats *provider.UsageStats) {
	// Build compact text for the bar
	var textParts []string
	for _, p := range stats.Providers {
		if p.Error != nil {
			continue
		}
		providerLabel := providerShortName(p.Provider)
		if len(p.Windows) > 0 {
			// Use the first window's utilization for the compact display
			textParts = append(textParts, fmt.Sprintf("%s:%.0f%%", providerLabel, p.Windows[0].Utilization))
		}
	}
	text := strings.Join(textParts, " ")

	// Build detailed tooltip
	var tooltipLines []string
	tooltipLines = append(tooltipLines, "LLM Usage")
	tooltipLines = append(tooltipLines, "")

	for _, p := range stats.Providers {
		if p.Error != nil {
			tooltipLines = append(tooltipLines, fmt.Sprintf("%s: Error", providerName(p.Provider)))
			continue
		}

		// Get account name if available
		accountSuffix := ""
		if acc, ok := p.Extra["account"]; ok && acc != "" {
			accountSuffix = fmt.Sprintf(" (%s)", acc)
		}

		for _, w := range p.Windows {
			line := fmt.Sprintf("%s%s %s: %.1f%%", providerName(p.Provider), accountSuffix, w.Label, w.Utilization)
			if d := w.TimeUntilReset(); d != nil {
				line += fmt.Sprintf(" (resets in %s)", formatDuration(*d))
			}
			tooltipLines = append(tooltipLines, line)
		}
	}

	output := WaybarOutput{
		Text:       text,
		Tooltip:    strings.Join(tooltipLines, "\n"),
		Class:      stats.GetClass(),
		Percentage: int(stats.MaxUtilization()),
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

func outputWaybarError(msg string) {
	output := WaybarOutput{
		Text:       "LLM: Error",
		Tooltip:    msg,
		Class:      "error",
		Percentage: 0,
	}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

func outputJSONMulti(stats *provider.UsageStats) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(stats); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputPrettyMulti(stats *provider.UsageStats) {
	fmt.Println("LLM Usage Statistics")
	fmt.Println("====================")
	fmt.Println()

	for _, p := range stats.Providers {
		if p.Error != nil {
			fmt.Printf("%s:\n", providerName(p.Provider))
			fmt.Printf("  Error: %s\n", p.Error)
			fmt.Println()
			continue
		}

		// Get account name if available
		accountSuffix := ""
		if acc, ok := p.Extra["account"]; ok && acc != "" {
			accountSuffix = fmt.Sprintf(" (%s)", acc)
		}

		fmt.Printf("%s%s:\n", providerName(p.Provider), accountSuffix)
		fmt.Println(strings.Repeat("-", len(providerName(p.Provider))+len(accountSuffix)+1))

		for _, w := range p.Windows {
			printUsageWindow(w.Label, &w)
		}

		// Print extra usage if available (for Claude)
		if extra, ok := p.Extra["extra_usage"]; ok {
			printExtraUsageFromMap(extra)
		}

		fmt.Println()
	}
}

func printExtraUsageFromMap(extra any) {
	extraMap, ok := extra.(map[string]any)
	if !ok {
		return
	}

	fmt.Println("Extra Usage Credits:")
	if utilization, ok := extraMap["utilization"]; ok {
		if util, ok := utilization.(float64); ok {
			bar := renderProgressBar(util)
			fmt.Printf("  Usage:    %s  %.1f%%\n", bar, util)
		}
	}
	if used, ok := extraMap["used_credits"]; ok {
		if limit, ok := extraMap["monthly_limit"]; ok {
			if usedFloat, ok := used.(float64); ok {
				if limitFloat, ok := limit.(float64); ok {
					fmt.Printf("  Credits:  $%.2f / $%.2f\n", usedFloat, limitFloat)
				}
			}
		}
	}
}

func printUsageWindow(label string, window *provider.UsageWindow) {
	fmt.Printf("  %s:\n", label)

	bar := renderProgressBar(window.Utilization)
	fmt.Printf("    Usage:    %s  %.1f%%\n", bar, window.Utilization)

	if resetDur := window.TimeUntilReset(); resetDur != nil {
		fmt.Printf("    Resets:   in %s\n", formatDuration(*resetDur))
	} else {
		fmt.Printf("    Resets:   N/A\n")
	}
}

func renderProgressBar(percentage float64) string {
	filled := int(percentage / 100 * float64(barWidth))
	filled = max(0, min(filled, barWidth))

	return strings.Repeat(barFull, filled) + strings.Repeat(barEmpty, barWidth-filled)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	return strings.Join(parts, " ")
}

func providerName(id string) string {
	switch id {
	case "claude":
		return "Claude (Pro/Max Subscription)"
	case "kimi":
		return "Kimi"
	case "zai":
		return "Z.AI"
	default:
		return strings.ToUpper(id)
	}
}

func providerShortName(id string) string {
	switch id {
	case "claude":
		return "C"
	case "kimi":
		return "K"
	case "zai":
		return "Z"
	default:
		return string(strings.ToUpper(id)[0])
	}
}
