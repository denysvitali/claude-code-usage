// Package setup provides the setup wizard and account management for llm-usage.
package setup

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/denysvitali/llm-usage/internal/credentials"
)

// SetupWizard runs an interactive setup wizard for first-time users
func SetupWizard(mgr *credentials.Manager) error {
	fmt.Println("Welcome to llm-usage setup!")
	fmt.Println("This wizard will help you configure your LLM provider credentials.")
	fmt.Println()

	providers := []struct {
		id   string
		name string
	}{
		{"claude", "Claude (Anthropic)"},
		{"kimi", "Kimi"},
		{"zai", "Z.AI"},
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
	case "claude":
		return addClaudeAccount(mgr, accountName)
	case "kimi":
		return addKimiAccount(mgr, accountName)
	case "zai":
		return addZaiAccount(mgr, accountName)
	default:
		return fmt.Errorf("unknown provider: %s", providerID)
	}
}

// addClaudeAccount adds a Claude account
func addClaudeAccount(mgr *credentials.Manager, _ string) error {
	fmt.Println("\nClaude (Anthropic) Setup")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Claude uses OAuth authentication which requires a browser flow.")
	fmt.Println("Please follow these steps:")
	fmt.Println()
	fmt.Println("1. Ensure you have the Claude CLI installed and authenticated:")
	fmt.Println("   npm install -g @anthropic-ai/claude-cli")
	fmt.Println("   claude login")
	fmt.Println()
	fmt.Println("2. Run the migration command to copy your credentials:")
	fmt.Println("   llm-usage setup migrate-claude")
	fmt.Println()
	fmt.Println("Or manually copy ~/.claude/.credentials.json to ~/.llm-usage/claude.json")
	fmt.Println()

	// Check if they want to migrate now
	fmt.Print("Would you like to migrate Claude CLI credentials now? [y/N]: ")
	if confirm() {
		if err := mgr.MigrateFromClaudeCLI(); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println("Successfully migrated Claude credentials!")
	}

	return nil
}

// addKimiAccount adds a Kimi account
func addKimiAccount(mgr *credentials.Manager, accountName string) error {
	fmt.Println("\nKimi Setup")
	fmt.Println("==========")
	fmt.Println()

	// Get account name if not provided
	if accountName == "" {
		fmt.Print("Enter account name (default): ")
		accountName = readLine()
		if accountName == "" {
			accountName = "default"
		}
	}

	// Get API key
	fmt.Print("Enter your Kimi API key: ")
	apiKey := readLine()
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Load existing credentials or create new
	var creds credentials.KimiCredentials
	if mgr.ProviderExists("kimi") {
		if err := mgr.LoadProvider("kimi", &creds); err != nil {
			// If load fails, start fresh
			creds = credentials.KimiCredentials{}
		}
	}

	// Initialize accounts map if needed
	if creds.Accounts == nil {
		creds.Accounts = make(map[string]*credentials.KimiAccount)
		// Migrate legacy API key if present
		if creds.APIKey != "" {
			creds.Accounts["default"] = &credentials.KimiAccount{APIKey: creds.APIKey}
			creds.APIKey = "" // Clear legacy field
		}
	}

	// Add/update account
	creds.Accounts[accountName] = &credentials.KimiAccount{APIKey: apiKey}

	// Save credentials
	if err := mgr.SaveProvider("kimi", creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("Successfully added Kimi account '%s'!\n", accountName)
	return nil
}

// addZaiAccount adds a Z.AI account
func addZaiAccount(mgr *credentials.Manager, accountName string) error {
	fmt.Println("\nZ.AI Setup")
	fmt.Println("==========")
	fmt.Println()

	// Get account name if not provided
	if accountName == "" {
		fmt.Print("Enter account name (default): ")
		accountName = readLine()
		if accountName == "" {
			accountName = "default"
		}
	}

	// Get API key
	fmt.Print("Enter your Z.AI API key: ")
	apiKey := readLine()
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Load existing credentials or create new
	var creds credentials.ZAiCredentials
	if mgr.ProviderExists("zai") {
		if err := mgr.LoadProvider("zai", &creds); err != nil {
			// If load fails, start fresh
			creds = credentials.ZAiCredentials{}
		}
	}

	// Initialize accounts map if needed
	if creds.Accounts == nil {
		creds.Accounts = make(map[string]*credentials.ZAiAccount)
		// Migrate legacy API key if present
		if creds.APIKey != "" {
			creds.Accounts["default"] = &credentials.ZAiAccount{APIKey: creds.APIKey}
			creds.APIKey = "" // Clear legacy field
		}
	}

	// Add/update account
	creds.Accounts[accountName] = &credentials.ZAiAccount{APIKey: apiKey}

	// Save credentials
	if err := mgr.SaveProvider("zai", creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("Successfully added Z.AI account '%s'!\n", accountName)
	return nil
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
	} else {
		for _, acc := range accounts {
			fmt.Printf("  - %s\n", acc)
		}
	}
	return nil
}

// RemoveAccount removes an account from a provider
func RemoveAccount(mgr *credentials.Manager, providerID, accountName string) error {
	if accountName == "" {
		return fmt.Errorf("account name is required")
	}

	switch providerID {
	case "claude":
		return removeClaudeAccount(mgr, accountName)
	case "kimi":
		return removeKimiAccount(mgr, accountName)
	case "zai":
		return removeZaiAccount(mgr, accountName)
	default:
		return fmt.Errorf("unknown provider: %s", providerID)
	}
}

// removeClaudeAccount removes a Claude account
func removeClaudeAccount(mgr *credentials.Manager, accountName string) error {
	var creds credentials.ClaudeCredentials
	if err := mgr.LoadProvider("claude", &creds); err != nil {
		return err
	}

	if creds.Accounts == nil || creds.Accounts[accountName] == nil {
		return fmt.Errorf("account '%s' not found", accountName)
	}

	delete(creds.Accounts, accountName)

	// If no accounts left, delete the file
	if len(creds.Accounts) == 0 {
		return mgr.DeleteProvider("claude")
	}

	return mgr.SaveProvider("claude", creds)
}

// removeKimiAccount removes a Kimi account
func removeKimiAccount(mgr *credentials.Manager, accountName string) error {
	var creds credentials.KimiCredentials
	if err := mgr.LoadProvider("kimi", &creds); err != nil {
		return err
	}

	if creds.Accounts == nil || creds.Accounts[accountName] == nil {
		return fmt.Errorf("account '%s' not found", accountName)
	}

	delete(creds.Accounts, accountName)

	// If no accounts left, delete the file
	if len(creds.Accounts) == 0 {
		return mgr.DeleteProvider("kimi")
	}

	return mgr.SaveProvider("kimi", creds)
}

// removeZaiAccount removes a Z.AI account
func removeZaiAccount(mgr *credentials.Manager, accountName string) error {
	var creds credentials.ZAiCredentials
	if err := mgr.LoadProvider("zai", &creds); err != nil {
		return err
	}

	if creds.Accounts == nil || creds.Accounts[accountName] == nil {
		return fmt.Errorf("account '%s' not found", accountName)
	}

	delete(creds.Accounts, accountName)

	// If no accounts left, delete the file
	if len(creds.Accounts) == 0 {
		return mgr.DeleteProvider("zai")
	}

	return mgr.SaveProvider("zai", creds)
}

// RenameAccount renames an account for a provider
func RenameAccount(mgr *credentials.Manager, providerID, oldName, newName string) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("both old and new account names are required")
	}

	switch providerID {
	case "claude":
		return renameClaudeAccount(mgr, oldName, newName)
	case "kimi":
		return renameKimiAccount(mgr, oldName, newName)
	case "zai":
		return renameZaiAccount(mgr, oldName, newName)
	default:
		return fmt.Errorf("unknown provider: %s", providerID)
	}
}

// renameClaudeAccount renames a Claude account
func renameClaudeAccount(mgr *credentials.Manager, oldName, newName string) error {
	var creds credentials.ClaudeCredentials
	if err := mgr.LoadProvider("claude", &creds); err != nil {
		return err
	}

	if creds.Accounts == nil || creds.Accounts[oldName] == nil {
		return fmt.Errorf("account '%s' not found", oldName)
	}

	if creds.Accounts[newName] != nil {
		return fmt.Errorf("account '%s' already exists", newName)
	}

	creds.Accounts[newName] = creds.Accounts[oldName]
	delete(creds.Accounts, oldName)

	return mgr.SaveProvider("claude", creds)
}

// renameKimiAccount renames a Kimi account
func renameKimiAccount(mgr *credentials.Manager, oldName, newName string) error {
	var creds credentials.KimiCredentials
	if err := mgr.LoadProvider("kimi", &creds); err != nil {
		return err
	}

	if creds.Accounts == nil || creds.Accounts[oldName] == nil {
		return fmt.Errorf("account '%s' not found", oldName)
	}

	if creds.Accounts[newName] != nil {
		return fmt.Errorf("account '%s' already exists", newName)
	}

	creds.Accounts[newName] = creds.Accounts[oldName]
	delete(creds.Accounts, oldName)

	return mgr.SaveProvider("kimi", creds)
}

// renameZaiAccount renames a Z.AI account
func renameZaiAccount(mgr *credentials.Manager, oldName, newName string) error {
	var creds credentials.ZAiCredentials
	if err := mgr.LoadProvider("zai", &creds); err != nil {
		return err
	}

	if creds.Accounts == nil || creds.Accounts[oldName] == nil {
		return fmt.Errorf("account '%s' not found", oldName)
	}

	if creds.Accounts[newName] != nil {
		return fmt.Errorf("account '%s' already exists", newName)
	}

	creds.Accounts[newName] = creds.Accounts[oldName]
	delete(creds.Accounts, oldName)

	return mgr.SaveProvider("zai", creds)
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
	switch id {
	case "claude":
		return "Claude (Anthropic)"
	case "kimi":
		return "Kimi"
	case "zai":
		return "Z.AI"
	default:
		return strings.ToUpper(id)
	}
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
