// Package setup provides the setup wizard and account management for llm-usage.
package setup

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/provider"
	"github.com/denysvitali/llm-usage/internal/provider/claude"
	"github.com/denysvitali/llm-usage/internal/provider/kimi"
	"github.com/denysvitali/llm-usage/internal/provider/minimax"
	"golang.org/x/term"
)

const (
	providerClaude  = credentials.ProviderClaude
	providerKimi    = credentials.ProviderKimi
	providerZAi     = credentials.ProviderZAi
	providerMiniMax = credentials.ProviderMiniMax

	defaultAccountName = credentials.DefaultAccountName
)

// Wizard runs an interactive setup wizard for first-time users
func Wizard(mgr *credentials.Manager) error {
	fmt.Println("Welcome to llm-usage setup!")
	fmt.Println("This wizard will help you configure your LLM provider credentials.")
	fmt.Println()

	providers := []struct {
		id   string
		name string
	}{
		{providerClaude, providerName(providerClaude)},
		{providerKimi, providerName(providerKimi)},
		{providerZAi, providerName(providerZAi) + " (usage fetching not yet implemented)"},
		{providerMiniMax, providerName(providerMiniMax)},
	}

	for _, p := range providers {
		fmt.Printf("\nWould you like to set up %s? [y/N]: ", p.name)
		if confirm() {
			if err := AddAccount(mgr, p.id, ""); err != nil {
				fmt.Fprintf(os.Stderr, "Error setting up %s: %v\n", p.name, err)
			}
		}
	}

	fmt.Println("\nSetup complete!")
	return nil
}

// AddAccount adds a new account for a provider
func AddAccount(mgr *credentials.Manager, providerID, accountName string) error {
	// Validate provider
	switch providerID {
	case providerClaude:
		return addClaudeAccount(mgr, accountName)
	case providerKimi:
		return addAPIKeyAccount(mgr, providerKimi, accountName)
	case providerZAi:
		return addAPIKeyAccount(mgr, providerZAi, accountName)
	case providerMiniMax:
		return addMiniMaxAccount(mgr, accountName)
	default:
		return fmt.Errorf("unknown provider: %s", providerID)
	}
}

// addClaudeAccount adds a Claude account, either by migrating from the Claude
// CLI or by manually entering OAuth tokens.
func addClaudeAccount(mgr *credentials.Manager, accountName string) error {
	fmt.Println("\nClaude (Anthropic) Setup")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Claude uses OAuth authentication. You can either:")
	fmt.Println()
	fmt.Println("1. Migrate credentials from the Claude CLI (recommended).")
	fmt.Println("   Install and authenticate first if needed:")
	fmt.Println("   npm install -g @anthropic-ai/claude-code")
	fmt.Println("   claude   (then authenticate when prompted)")
	fmt.Println()
	fmt.Println("2. Manually enter an OAuth access token (and optional refresh token).")
	fmt.Println()

	fmt.Print("Migrate Claude CLI credentials now? [y/N]: ")
	if confirm() {
		if err := mgr.MigrateFromClaudeCLI(); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println("Successfully migrated Claude credentials!")
		return nil
	}

	// Manual token entry
	fmt.Print("Enter tokens manually instead? [y/N]: ")
	if !confirm() {
		return nil
	}

	if accountName == "" {
		fmt.Printf("Enter account name (%s): ", defaultAccountName)
		accountName = readLine()
		if accountName == "" {
			accountName = defaultAccountName
		}
	}

	accessToken := readSecret("Enter your Claude OAuth access token: ")
	if accessToken == "" {
		return fmt.Errorf("access token is required")
	}
	refreshToken := readSecret("Enter your refresh token (optional, enables auto-refresh): ")

	if err := SaveClaudeAccount(mgr, accountName, accessToken, refreshToken); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}
	fmt.Printf("Successfully added Claude account '%s'!\n", accountName)
	verifyAccount(claude.NewProvider(accessToken, false))
	return nil
}

// addAPIKeyAccount adds an account for API key-based providers (Kimi, Z.AI)
func addAPIKeyAccount(mgr *credentials.Manager, providerID, accountName string) error {
	displayName := providerName(providerID)
	fmt.Printf("\n%s Setup\n", displayName)
	fmt.Println(strings.Repeat("=", len(displayName)+6))
	fmt.Println()

	// Get account name if not provided
	if accountName == "" {
		fmt.Printf("Enter account name (%s): ", defaultAccountName)
		accountName = readLine()
		if accountName == "" {
			accountName = defaultAccountName
		}
	}

	// Get API key
	apiKey := readSecret(fmt.Sprintf("Enter your %s API key: ", displayName))
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	if err := SaveAPIKeyAccount(mgr, providerID, accountName, apiKey); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}
	fmt.Printf("Successfully added %s account '%s'!\n", displayName, accountName)

	if providerID == providerKimi {
		verifyAccount(kimi.NewProvider(apiKey))
	}
	return nil
}

// addMiniMaxAccount adds a MiniMax account
func addMiniMaxAccount(mgr *credentials.Manager, accountName string) error {
	fmt.Println("\nMiniMax Setup")
	fmt.Println("=============")
	fmt.Println()
	fmt.Println("MiniMax uses cookie-based authentication.")
	fmt.Println()

	// Get account name if not provided
	if accountName == "" {
		fmt.Printf("Enter account name (%s): ", defaultAccountName)
		accountName = readLine()
		if accountName == "" {
			accountName = defaultAccountName
		}
	}

	// Get Group ID
	fmt.Print("Enter your MiniMax Group ID: ")
	groupID := readLine()
	if groupID == "" {
		return fmt.Errorf("group ID is required")
	}

	// Get Cookie
	cookie := readSecret("Enter your MiniMax cookie: ")
	if cookie == "" {
		return fmt.Errorf("cookie is required")
	}

	if err := SaveMiniMaxAccount(mgr, accountName, cookie, groupID); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}
	fmt.Printf("Successfully added MiniMax account '%s'!\n", accountName)
	verifyAccount(minimax.NewProvider(cookie, groupID))
	return nil
}

// verifyAccount makes a test API call with the freshly saved credentials and
// reports the result, so typos are caught at setup time instead of first use.
func verifyAccount(p provider.Provider) {
	fmt.Print("Verifying credentials... ")
	if _, err := p.GetUsage(); err != nil {
		fmt.Println("failed!")
		fmt.Fprintf(os.Stderr, "Warning: test API call failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "The credentials were saved anyway - double-check them if usage fetching fails.")
		return
	}
	fmt.Println("OK")
}

// SaveClaudeAccount saves a Claude account with manually provided OAuth tokens.
func SaveClaudeAccount(mgr *credentials.Manager, accountName, accessToken, refreshToken string) error {
	var creds credentials.ClaudeCredentials
	if mgr.ProviderExists(providerClaude) {
		if err := mgr.LoadProvider(providerClaude, &creds); err != nil {
			creds = credentials.ClaudeCredentials{}
		}
	}

	if creds.Accounts == nil {
		creds.Accounts = make(map[string]*credentials.ClaudeAccount)
		if creds.ClaudeAiOauth != nil {
			creds.Accounts[defaultAccountName] = &credentials.ClaudeAccount{
				AccessToken:  creds.ClaudeAiOauth.AccessToken,
				RefreshToken: creds.ClaudeAiOauth.RefreshToken,
				ExpiresAt:    creds.ClaudeAiOauth.ExpiresAt,
				Scopes:       creds.ClaudeAiOauth.Scopes,
			}
			creds.ClaudeAiOauth = nil
		}
	}

	creds.Accounts[accountName] = &credentials.ClaudeAccount{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return mgr.SaveProvider(providerClaude, creds)
}

// SaveAPIKeyAccount saves an account for API key-based providers (Kimi, Z.AI),
// migrating any legacy single-key credentials into the accounts map.
func SaveAPIKeyAccount(mgr *credentials.Manager, providerID, accountName, apiKey string) error {
	switch providerID {
	case providerKimi:
		var creds credentials.KimiCredentials
		if mgr.ProviderExists(providerKimi) {
			if err := mgr.LoadProvider(providerKimi, &creds); err != nil {
				creds = credentials.KimiCredentials{}
			}
		}
		if creds.Accounts == nil {
			creds.Accounts = make(map[string]*credentials.KimiAccount)
			if creds.APIKey != "" {
				creds.Accounts[defaultAccountName] = &credentials.KimiAccount{APIKey: creds.APIKey}
				creds.APIKey = ""
			}
		}
		creds.Accounts[accountName] = &credentials.KimiAccount{APIKey: apiKey}
		return mgr.SaveProvider(providerKimi, creds)
	case providerZAi:
		var creds credentials.ZAiCredentials
		if mgr.ProviderExists(providerZAi) {
			if err := mgr.LoadProvider(providerZAi, &creds); err != nil {
				creds = credentials.ZAiCredentials{}
			}
		}
		if creds.Accounts == nil {
			creds.Accounts = make(map[string]*credentials.ZAiAccount)
			if creds.APIKey != "" {
				creds.Accounts[defaultAccountName] = &credentials.ZAiAccount{APIKey: creds.APIKey}
				creds.APIKey = ""
			}
		}
		creds.Accounts[accountName] = &credentials.ZAiAccount{APIKey: apiKey}
		return mgr.SaveProvider(providerZAi, creds)
	default:
		return fmt.Errorf("unsupported provider: %s", providerID)
	}
}

// SaveMiniMaxAccount saves a MiniMax account, migrating any legacy
// single-account credentials into the accounts map.
func SaveMiniMaxAccount(mgr *credentials.Manager, accountName, cookie, groupID string) error {
	var creds credentials.MiniMaxCredentials
	if mgr.ProviderExists(providerMiniMax) {
		if err := mgr.LoadProvider(providerMiniMax, &creds); err != nil {
			creds = credentials.MiniMaxCredentials{}
		}
	}

	if creds.Accounts == nil {
		creds.Accounts = make(map[string]*credentials.MiniMaxAccount)
		if creds.Cookie != "" {
			creds.Accounts[defaultAccountName] = &credentials.MiniMaxAccount{Cookie: creds.Cookie, GroupID: creds.GroupID}
			creds.Cookie = ""
			creds.GroupID = ""
		}
	}

	creds.Accounts[accountName] = &credentials.MiniMaxAccount{Cookie: cookie, GroupID: groupID}

	return mgr.SaveProvider(providerMiniMax, creds)
}

// ListAccounts lists all configured accounts
func ListAccounts(mgr *credentials.Manager, providerID string) error {
	if providerID == "" {
		// List all providers and their accounts
		providers := mgr.ListAvailable()
		if len(providers) == 0 {
			fmt.Println("No providers configured.")
			fmt.Println("Run 'llm-usage setup' to configure providers.")
			return nil
		}

		fmt.Println("Configured Accounts")
		fmt.Println("===================")
		for _, pid := range providers {
			if err := listProviderAccounts(mgr, pid); err != nil {
				fmt.Fprintf(os.Stderr, "Error listing %s accounts: %v\n", pid, err)
			}
		}
	} else {
		// List specific provider
		return listProviderAccounts(mgr, providerID)
	}
	return nil
}

// listProviderAccounts lists accounts for a specific provider
func listProviderAccounts(mgr *credentials.Manager, providerID string) error {
	accounts, err := mgr.ListAccounts(providerID)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s:\n", providerName(providerID))
	if len(accounts) == 0 {
		fmt.Println("  (no accounts configured)")
		return nil
	}

	// For Claude, show token expiry status alongside each account
	var claudeCreds *credentials.ClaudeCredentials
	if providerID == providerClaude {
		claudeCreds, _ = mgr.LoadClaude()
	}

	for _, acc := range accounts {
		status := ""
		if claudeCreds != nil {
			if oauth := claudeCreds.GetAccount(acc); oauth != nil && oauth.ExpiresAt > 0 {
				if oauth.IsExpired() {
					status = " (token expired"
					if oauth.RefreshToken != "" {
						status += ", will auto-refresh"
					}
					status += ")"
				} else {
					status = fmt.Sprintf(" (token expires in %s)", oauth.ExpiresIn().Round(1e9))
				}
			}
		}
		fmt.Printf("  - %s%s\n", acc, status)
	}
	return nil
}

// RemoveAccount removes an account from a provider
func RemoveAccount(mgr *credentials.Manager, providerID, accountName string) error {
	if accountName == "" {
		return fmt.Errorf("account name is required")
	}

	creds, err := mgr.LoadAccountCredentials(providerID)
	if err != nil {
		return err
	}
	if err := creds.RemoveAccount(accountName); err != nil {
		return err
	}

	// If no accounts left, delete the file
	if len(creds.ListAccounts()) == 0 {
		return mgr.DeleteProvider(providerID)
	}
	return mgr.SaveProvider(providerID, creds)
}

// RenameAccount renames an account for a provider
func RenameAccount(mgr *credentials.Manager, providerID, oldName, newName string) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("both old and new account names are required")
	}

	creds, err := mgr.LoadAccountCredentials(providerID)
	if err != nil {
		return err
	}
	if err := creds.RenameAccount(oldName, newName); err != nil {
		return err
	}
	return mgr.SaveProvider(providerID, creds)
}

// MigrateClaudeCLI migrates credentials from the Claude CLI
func MigrateClaudeCLI(mgr *credentials.Manager) error {
	if err := mgr.MigrateFromClaudeCLI(); err != nil {
		return err
	}
	fmt.Println("Successfully migrated Claude CLI credentials!")
	fmt.Printf("Credentials saved to: %s/claude.json\n", mgr.ConfigDir())
	return nil
}

// providerName returns the display name for a provider
func providerName(id string) string {
	return credentials.ProviderDisplayName(id)
}

// confirm asks the user for confirmation (y/n)
func confirm() bool {
	line := readLine()
	return strings.ToLower(line) == "y" || strings.ToLower(line) == "yes"
}

// readLine reads a line of input from stdin
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// readSecret prompts for a secret and reads it without echoing to the
// terminal. Falls back to plain line reading when stdin is not a terminal.
func readSecret(prompt string) string {
	fmt.Print(prompt)
	fd := int(os.Stdin.Fd()) //nolint:gosec // stdin fd is small, no overflow risk
	if !term.IsTerminal(fd) {
		return readLine()
	}
	secret, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(secret))
}
