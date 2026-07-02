// Package usage provides shared usage fetching and output logic.
package usage

import (
	"fmt"
	"strings"
	"sync"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/provider"
	"github.com/denysvitali/llm-usage/internal/provider/claude"
	"github.com/denysvitali/llm-usage/internal/provider/kimi"
	"github.com/denysvitali/llm-usage/internal/provider/minimax"
	"github.com/denysvitali/llm-usage/internal/provider/zai"
)

const (
	providerClaude  = credentials.ProviderClaude
	providerKimi    = credentials.ProviderKimi
	providerZAi     = credentials.ProviderZAi
	providerMiniMax = credentials.ProviderMiniMax

	defaultAccountName = credentials.DefaultAccountName
)

// ProviderInstance holds a provider instance along with its account info
type ProviderInstance struct {
	provider.Provider
	AccountName string
}

// LoadClaudeFromKeychain tries to load Claude credentials from the CLI's storage location:
// the macOS Keychain first, falling back to ~/.claude/.credentials.json.
func LoadClaudeFromKeychain() (*credentials.OAuthCredentials, string, error) {
	creds, err := credentials.Load()
	if err != nil {
		return nil, "", err
	}

	return creds.ClaudeAiOauth, defaultAccountName, nil
}

// metadataProviders holds zero-credential provider instances, used only to
// access display metadata (Name, ShortName) through the Provider interface.
var metadataProviders = map[string]provider.Provider{
	providerClaude:  claude.NewProvider("", false),
	providerKimi:    kimi.NewProvider(""),
	providerZAi:     zai.NewProvider(""),
	providerMiniMax: minimax.NewProvider("", ""),
}

// ProviderName returns the display name for a provider ID.
func ProviderName(id string) string {
	if p, ok := metadataProviders[id]; ok {
		return p.Name()
	}
	return strings.ToUpper(id)
}

// ProviderShortName returns the compact label for a provider ID.
func ProviderShortName(id string) string {
	if p, ok := metadataProviders[id]; ok {
		return p.ShortName()
	}
	return strings.ToUpper(id)[:1]
}

// failure creates a Usage error entry, tagging the account name if provided.
func failure(providerID, accountName string, err error) provider.Usage {
	u := provider.NewUsageError(providerID, ProviderName(providerID), err)
	if accountName != "" {
		u.Extra = map[string]any{"account": accountName}
	}
	return *u
}

// GetProviders returns the list of providers to query based on the flags,
// along with pre-failed usage entries for providers that could not be loaded
// (expired tokens, missing accounts, malformed credentials, unknown providers).
func GetProviders(providerFlag, accountFlag string, allAccounts, debug bool, credsMgr *credentials.Manager) ([]ProviderInstance, []provider.Usage) {
	var providerIDs []string
	explicit := providerFlag != "all" && providerFlag != ""

	if explicit {
		providerIDs = strings.Split(providerFlag, ",")
	} else {
		// Show all configured providers
		providerIDs = credsMgr.ListAvailable()
		// If no providers are configured, default to claude
		if len(providerIDs) == 0 {
			providerIDs = []string{providerClaude}
		}
	}

	var providers []ProviderInstance
	var failures []provider.Usage
	for _, pid := range providerIDs {
		pid = strings.TrimSpace(pid)
		switch pid {
		case providerClaude:
			p, f := getClaudeProviders(accountFlag, debug, explicit, credsMgr)
			providers = append(providers, p...)
			failures = append(failures, f...)
		case providerKimi:
			p, f := getKimiProviders(accountFlag, allAccounts, credsMgr)
			providers = append(providers, p...)
			failures = append(failures, f...)
		case providerZAi:
			p, f := getZaiProviders(accountFlag, allAccounts, credsMgr)
			providers = append(providers, p...)
			failures = append(failures, f...)
		case providerMiniMax:
			p, f := getMiniMaxProviders(accountFlag, allAccounts, credsMgr)
			providers = append(providers, p...)
			failures = append(failures, f...)
		default:
			failures = append(failures, failure(pid, "",
				fmt.Errorf("unknown provider %q (valid: claude, kimi, zai, minimax)", pid)))
		}
	}

	return providers, failures
}

// freshClaudeToken returns valid OAuth credentials for an account, refreshing
// (and persisting) them if the access token has expired.
func freshClaudeToken(oauth *credentials.OAuthCredentials, accountName string, creds *credentials.ClaudeCredentials, credsMgr *credentials.Manager) (*credentials.OAuthCredentials, error) {
	if !claude.IsExpired(oauth.ExpiresAt) {
		return oauth, nil
	}
	if oauth.RefreshToken == "" {
		return nil, fmt.Errorf("access token expired and no refresh token available - run 'llm-usage setup migrate-claude' or 'llm-usage setup add claude'")
	}

	refreshed, err := claude.RefreshAccessToken(oauth.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("access token expired and refresh failed: %w", err)
	}

	oauth.AccessToken = refreshed.AccessToken
	if refreshed.RefreshToken != "" {
		oauth.RefreshToken = refreshed.RefreshToken
	}
	oauth.ExpiresAt = refreshed.ExpiresAt

	// Persist the refreshed token back to the per-provider credentials file
	// (skip when credentials come from a read-only combined file).
	if !credsMgr.UsesCombinedFile() {
		if creds.Accounts != nil {
			if acc, ok := creds.Accounts[accountName]; ok && acc != nil {
				acc.AccessToken = oauth.AccessToken
				acc.RefreshToken = oauth.RefreshToken
				acc.ExpiresAt = oauth.ExpiresAt
			}
		} else if creds.ClaudeAiOauth != nil {
			creds.ClaudeAiOauth = oauth
		}
		_ = credsMgr.SaveProvider(providerClaude, creds)
	}

	return oauth, nil
}

// getClaudeProviders returns Claude provider instances
func getClaudeProviders(accountFlag string, debug, explicit bool, credsMgr *credentials.Manager) ([]ProviderInstance, []provider.Usage) {
	var providers []ProviderInstance
	var failures []provider.Usage

	// Try loading from keychain first (Claude CLI location)
	keychainCreds, keychainAccount, keychainErr := LoadClaudeFromKeychain()

	// Also try loading from the new multi-account location
	multiCreds, multiErr := credsMgr.LoadClaude()

	// Neither source available
	if keychainErr != nil && multiErr != nil {
		if explicit {
			failures = append(failures, failure(providerClaude, "",
				fmt.Errorf("no credentials found - run 'llm-usage setup': %w", multiErr)))
		}
		return providers, failures
	}

	// If a specific account is requested, only use the multi-account location
	if accountFlag != "" {
		if multiErr != nil {
			failures = append(failures, failure(providerClaude, accountFlag, multiErr))
			return providers, failures
		}
		oauth := multiCreds.GetAccount(accountFlag)
		if oauth == nil {
			failures = append(failures, failure(providerClaude, accountFlag,
				fmt.Errorf("account %q not found", accountFlag)))
			return providers, failures
		}
		oauth, err := freshClaudeToken(oauth, accountFlag, multiCreds, credsMgr)
		if err != nil {
			failures = append(failures, failure(providerClaude, accountFlag, err))
			return providers, failures
		}
		providers = append(providers, ProviderInstance{
			Provider:    claude.NewProvider(oauth.AccessToken, debug),
			AccountName: accountFlag,
		})
		return providers, failures
	}

	// No specific account requested - show all available
	// Add from keychain if available
	keychainAdded := false
	if keychainErr == nil {
		if claude.IsExpired(keychainCreds.ExpiresAt) {
			// The Claude CLI owns these credentials; don't rotate its refresh token.
			failures = append(failures, failure(providerClaude, keychainAccount,
				fmt.Errorf("token from the Claude CLI expired - run 'claude' to re-authenticate")))
		} else {
			providers = append(providers, ProviderInstance{
				Provider:    claude.NewProvider(keychainCreds.AccessToken, debug),
				AccountName: keychainAccount,
			})
			keychainAdded = true
		}
	}
	// Add from multi-account location if available
	if multiErr == nil {
		for _, accName := range multiCreds.ListAccounts() {
			// Skip if this was already added from keychain
			if keychainAdded && accName == defaultAccountName {
				continue
			}
			oauth := multiCreds.GetAccount(accName)
			if oauth == nil {
				continue
			}
			oauth, err := freshClaudeToken(oauth, accName, multiCreds, credsMgr)
			if err != nil {
				failures = append(failures, failure(providerClaude, accName, err))
				continue
			}
			providers = append(providers, ProviderInstance{
				Provider:    claude.NewProvider(oauth.AccessToken, debug),
				AccountName: accName,
			})
		}
	}

	return providers, failures
}

// getAccountProviders builds provider instances for account-based providers.
// build must return nil when the account does not exist.
func getAccountProviders[C any](
	providerID, accountFlag string, allAccounts bool,
	load func() (C, error),
	list func(C) []string,
	build func(C, string) provider.Provider,
) ([]ProviderInstance, []provider.Usage) {
	creds, err := load()
	if err != nil {
		return nil, []provider.Usage{failure(providerID, "", err)}
	}

	if allAccounts || accountFlag == "" {
		// Add all accounts when --all-accounts is set or no specific account requested
		var providers []ProviderInstance
		for _, accName := range list(creds) {
			if p := build(creds, accName); p != nil {
				providers = append(providers, ProviderInstance{Provider: p, AccountName: accName})
			}
		}
		return providers, nil
	}

	// Use specified account
	p := build(creds, accountFlag)
	if p == nil {
		return nil, []provider.Usage{failure(providerID, accountFlag,
			fmt.Errorf("account %q not found", accountFlag))}
	}
	return []ProviderInstance{{Provider: p, AccountName: accountFlag}}, nil
}

// getKimiProviders returns Kimi provider instances
func getKimiProviders(accountFlag string, allAccounts bool, credsMgr *credentials.Manager) ([]ProviderInstance, []provider.Usage) {
	return getAccountProviders(providerKimi, accountFlag, allAccounts,
		credsMgr.LoadKimi,
		func(c *credentials.KimiCredentials) []string { return c.ListAccounts() },
		func(c *credentials.KimiCredentials, name string) provider.Provider {
			acc := c.GetAccount(name)
			if acc == nil {
				return nil
			}
			return kimi.NewProvider(acc.APIKey)
		})
}

// getZaiProviders returns Z.AI provider instances
func getZaiProviders(accountFlag string, allAccounts bool, credsMgr *credentials.Manager) ([]ProviderInstance, []provider.Usage) {
	return getAccountProviders(providerZAi, accountFlag, allAccounts,
		credsMgr.LoadZAi,
		func(c *credentials.ZAiCredentials) []string { return c.ListAccounts() },
		func(c *credentials.ZAiCredentials, name string) provider.Provider {
			acc := c.GetAccount(name)
			if acc == nil {
				return nil
			}
			return zai.NewProvider(acc.APIKey)
		})
}

// getMiniMaxProviders returns MiniMax provider instances
func getMiniMaxProviders(accountFlag string, allAccounts bool, credsMgr *credentials.Manager) ([]ProviderInstance, []provider.Usage) {
	return getAccountProviders(providerMiniMax, accountFlag, allAccounts,
		credsMgr.LoadMiniMax,
		func(c *credentials.MiniMaxCredentials) []string { return c.ListAccounts() },
		func(c *credentials.MiniMaxCredentials, name string) provider.Provider {
			acc := c.GetAccount(name)
			if acc == nil {
				return nil
			}
			return minimax.NewProvider(acc.Cookie, acc.GroupID)
		})
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
				stats.Providers[idx] = failure(prov.ID(), prov.AccountName, err)
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
