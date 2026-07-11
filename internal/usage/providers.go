// Package usage provides shared usage fetching and output logic.
package usage

import (
	"sync"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/provider"
	registry "github.com/denysvitali/llm-usage/providers"
)

// ProviderInstance is a provider ready for concurrent usage retrieval.
type ProviderInstance = registry.Instance

// LoadClaudeFromKeychain loads the credentials managed by the Claude CLI.
// It remains exported for the HTTP provider-discovery endpoint.
func LoadClaudeFromKeychain() (*credentials.OAuthCredentials, string, error) {
	creds, err := credentials.Load()
	if err != nil {
		return nil, "", err
	}
	return creds.ClaudeAiOauth, credentials.DefaultAccountName, nil
}

func ProviderName(id string) string      { return registry.Name(id) }
func ProviderShortName(id string) string { return registry.ShortName(id) }

// GetProviders resolves registered provider definitions into ready instances.
func GetProviders(providerFlag, accountFlag string, allAccounts, debug bool, credsMgr *credentials.Manager) ([]ProviderInstance, []provider.Usage) {
	return registry.Resolve(registry.Request{
		Provider: providerFlag, Account: accountFlag, AllAccounts: allAccounts,
		Debug: debug, Explicit: providerFlag != "all" && providerFlag != "",
	}, credsMgr)
}

// FetchAllUsage fetches usage from all providers concurrently.
func FetchAllUsage(providers []ProviderInstance) *provider.UsageStats {
	var wg sync.WaitGroup
	var mu sync.Mutex

	stats := &provider.UsageStats{Providers: make([]provider.Usage, len(providers))}
	for i, instance := range providers {
		wg.Add(1)
		go func(index int, current ProviderInstance) {
			defer wg.Done()
			result, err := current.GetUsage()
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				stats.Providers[index] = registry.Failure(current.ID(), current.AccountName, err)
				return
			}
			if current.AccountName != "" {
				if result.Extra == nil {
					result.Extra = make(map[string]any)
				}
				result.Extra["account"] = current.AccountName
			}
			stats.Providers[index] = *result
		}(i, instance)
	}
	wg.Wait()

	filtered := make([]provider.Usage, 0, len(stats.Providers))
	for _, item := range stats.Providers {
		if item.Provider != "" {
			filtered = append(filtered, item)
		}
	}
	stats.Providers = filtered
	return stats
}
