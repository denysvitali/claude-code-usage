// Package usage provides shared usage fetching and output logic.
package usage

import (
	"context"
	"sync"
	"time"

	"github.com/denysvitali/llm-usage/internal/cache"
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

// ProviderName returns the display name for a provider ID.
func ProviderName(id string) string { return registry.Name(id) }

// ProviderShortName returns the short label for a provider ID.
func ProviderShortName(id string) string { return registry.ShortName(id) }

// GetProviders resolves registered provider definitions into ready instances.
func GetProviders(providerFlag, accountFlag string, allAccounts, debug, raw bool, configuredAccounts map[string][]string, credsMgr *credentials.Manager) ([]ProviderInstance, []provider.Usage) {
	return registry.Resolve(registry.Request{
		Provider: providerFlag, Account: accountFlag, AllAccounts: allAccounts,
		Debug: debug, Raw: raw, Explicit: providerFlag != "all" && providerFlag != "",
		ConfiguredAccounts: configuredAccounts,
	}, credsMgr)
}

// CacheOptions controls optional caching behavior for FetchAllUsage.
type CacheOptions struct {
	Manager      *cache.Manager
	TTL          time.Duration
	StaleIfError bool
}

// FetchAllUsage fetches usage from all providers concurrently.
func FetchAllUsage(ctx context.Context, providers []ProviderInstance, cacheOptions CacheOptions) *provider.UsageStats {
	var wg sync.WaitGroup
	var mu sync.Mutex

	stats := &provider.UsageStats{Providers: make([]provider.Usage, len(providers))}
	for i, instance := range providers {
		wg.Add(1)
		go func(index int, current ProviderInstance) {
			defer wg.Done()
			key := cache.HashKey(current.ID(), current.AccountName)
			var cached provider.Usage
			if cacheOptions.Manager != nil && cacheOptions.TTL > 0 {
				if found, fresh, age, cacheErr := cacheOptions.Manager.Lookup(key, &cached); cacheErr == nil && found && fresh {
					markCached(&cached, age, false)
					stats.Providers[index] = cached
					return
				}
			}
			result, err := current.GetUsage(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if cacheOptions.Manager != nil && cacheOptions.TTL > 0 && cacheOptions.StaleIfError {
					if found, fresh, age, cacheErr := cacheOptions.Manager.Lookup(key, &cached); cacheErr == nil && found && !fresh {
						markCached(&cached, age, true)
						stats.Providers[index] = cached
						return
					}
				}
				stats.Providers[index] = registry.Failure(current.ID(), current.AccountName, err)
				return
			}
			if current.AccountName != "" {
				if result.Extra == nil {
					result.Extra = make(map[string]any)
				}
				result.Extra["account"] = current.AccountName
			}
			if cacheOptions.Manager != nil && cacheOptions.TTL > 0 {
				_ = cacheOptions.Manager.Set(key, *result, cacheOptions.TTL)
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

func markCached(report *provider.Usage, age time.Duration, stale bool) {
	if report.Extra == nil {
		report.Extra = make(map[string]any)
	}
	report.Extra["cache"] = map[string]any{"age_seconds": int(age.Seconds()), "stale": stale}
}
