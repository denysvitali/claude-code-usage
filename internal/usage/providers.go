// Package usage provides shared usage fetching and output logic.
package usage

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/provider"
	"github.com/denysvitali/llm-usage/internal/provider/claude"
	"github.com/denysvitali/llm-usage/internal/provider/kimi"
	"github.com/denysvitali/llm-usage/internal/provider/zai"
)

const (
	providerClaude = "claude"
	providerKimi   = "kimi"
	providerZAi    = "zai"
)

// ProviderInstance holds a provider instance along with its account info
type ProviderInstance struct {
	provider.Provider
	AccountName string
}

// LoadClaudeFromKeychain tries to load Claude credentials from the CLI keychain location
func LoadClaudeFromKeychain() (*credentials.OAuthCredentials, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", err
	}

	credPath := homeDir + "/.claude/.credentials.json"
	data, err := os.ReadFile(credPath) //nolint:gosec // Path is constructed from home directory
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
		return nil, "", ErrNoValidCredentials
	}

	return result.ClaudeAiOauth, "default", nil
}

// GetProviders returns the list of providers to query based on the flags
func GetProviders(providerFlag, accountFlag string, allAccounts bool, credsMgr *credentials.Manager) []ProviderInstance {
	var providerIDs []string

	if providerFlag == "all" || providerFlag == "" {
		// Show all configured providers
		providerIDs = credsMgr.ListAvailable()
		// If no providers are configured, default to claude
		if len(providerIDs) == 0 {
			providerIDs = []string{providerClaude}
		}
	} else {
		providerIDs = strings.Split(providerFlag, ",")
	}

	var providers []ProviderInstance
	for _, pid := range providerIDs {
		pid = strings.TrimSpace(pid)
		switch pid {
		case providerClaude:
			providers = append(providers, getClaudeProviders(accountFlag, credsMgr)...)
		case providerKimi:
			providers = append(providers, getKimiProviders(accountFlag, allAccounts, credsMgr)...)
		case providerZAi:
			providers = append(providers, getZaiProviders(accountFlag, allAccounts, credsMgr)...)
		}
	}

	return providers
}

// getClaudeProviders returns Claude provider instances
func getClaudeProviders(accountFlag string, credsMgr *credentials.Manager) []ProviderInstance {
	var providers []ProviderInstance

	// Try loading from keychain first (Claude CLI location)
	keychainCreds, keychainAccount, keychainErr := LoadClaudeFromKeychain()

	// Also try loading from the new multi-account location
	multiCreds, multiErr := credsMgr.LoadClaude()

	// Determine which source to use
	if keychainErr != nil && multiErr != nil {
		// Neither source available, skip
		return providers
	}

	// If a specific account is requested, only use the multi-account location
	if accountFlag != "" {
		if multiErr != nil {
			return providers
		}
		oauth := multiCreds.GetAccount(accountFlag)
		if oauth == nil || claude.IsExpired(oauth.ExpiresAt) {
			return providers
		}
		providers = append(providers, ProviderInstance{
			Provider:    claude.NewProvider(oauth.AccessToken),
			AccountName: accountFlag,
		})
		return providers
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

	return providers
}

// getKimiProviders returns Kimi provider instances
func getKimiProviders(accountFlag string, allAccounts bool, credsMgr *credentials.Manager) []ProviderInstance {
	var providers []ProviderInstance

	creds, err := credsMgr.LoadKimi()
	if err != nil {
		return providers
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
			return providers
		}
		providers = append(providers, ProviderInstance{
			Provider:    kimi.NewProvider(acc.APIKey),
			AccountName: accountFlag,
		})
	}

	return providers
}

// getZaiProviders returns Z.AI provider instances
func getZaiProviders(accountFlag string, allAccounts bool, credsMgr *credentials.Manager) []ProviderInstance {
	var providers []ProviderInstance

	creds, err := credsMgr.LoadZAi()
	if err != nil {
		return providers
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
			return providers
		}
		providers = append(providers, ProviderInstance{
			Provider:    zai.NewProvider(acc.APIKey),
			AccountName: accountFlag,
		})
	}

	return providers
}

// FetchAllUsage fetches usage from all providers concurrently
func FetchAllUsage(providers []ProviderInstance) *provider.UsageStats {
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
