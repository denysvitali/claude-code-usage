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

	stats := &provider.UsageStats{Providers: make([]provider.Usage, len(providers))}
	for i, instance := range providers {
		wg.Add(1)
		go func(index int, current ProviderInstance) {
			defer wg.Done()
			// Each goroutine owns exactly one slice element, so no lock is needed.
			stats.Providers[index] = fetchOne(ctx, current, cacheOptions)
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

// fetchOne resolves the usage for a single account, preferring cached data over
// a request the provider is likely to reject.
func fetchOne(ctx context.Context, instance ProviderInstance, options CacheOptions) provider.Usage {
	manager := options.Manager
	key := cache.HashKey(instance.ID(), instance.AccountName)
	caching := manager != nil && options.TTL > 0

	if caching {
		if cached, ok := lookup(manager, key, freshOnly); ok {
			return cached
		}
	}

	// A provider that rate-limited us stays off-limits until the cooldown it
	// asked for expires; retrying sooner only extends the lockout. Cooldowns
	// are honored even with caching disabled, since they are the only thing
	// keeping repeated invocations from hammering a limit we already hit.
	if manager != nil {
		if retryAt := manager.Cooldown(key); time.Now().Before(retryAt) {
			if cached, ok := lookup(manager, key, anyAge); ok {
				markRateLimited(&cached, retryAt)
				return cached
			}
			return registry.Failure(instance.ID(), instance.AccountName, &provider.RateLimitError{
				Provider: instance.ID(),
				RetryAt:  retryAt,
				Detail:   "waiting out an earlier rate limit",
			})
		}
	}

	result, err := instance.GetUsage(ctx)
	if err != nil {
		retryAt, rateLimited := provider.RetryAt(err)
		if manager != nil {
			if rateLimited {
				_ = manager.MarkCooldown(key, retryAt)
			}
			// Rate limiting always falls back to the last good read: another
			// request cannot fix it, so stale data beats no data.
			if rateLimited || options.StaleIfError {
				if cached, ok := lookup(manager, key, staleOnly); ok {
					if rateLimited {
						markRateLimited(&cached, retryAt)
					}
					return cached
				}
			}
		}
		return registry.Failure(instance.ID(), instance.AccountName, err)
	}

	if instance.AccountName != "" {
		if result.Extra == nil {
			result.Extra = make(map[string]any)
		}
		result.Extra["account"] = instance.AccountName
	}
	if caching {
		_ = manager.Set(key, *result, options.TTL)
	}
	return *result
}

// freshness selects which cached entries a lookup accepts.
type freshness int

const (
	freshOnly freshness = iota
	staleOnly
	anyAge
)

// lookup reads a cached report and tags it with its age, or reports a miss when
// no entry matches the requested freshness.
func lookup(manager *cache.Manager, key string, accept freshness) (provider.Usage, bool) {
	var cached provider.Usage
	found, fresh, age, err := manager.Lookup(key, &cached)
	if err != nil || !found {
		return provider.Usage{}, false
	}
	if (accept == freshOnly && !fresh) || (accept == staleOnly && fresh) {
		return provider.Usage{}, false
	}
	markCached(&cached, age, !fresh)
	return cached, true
}

func markCached(report *provider.Usage, age time.Duration, stale bool) {
	if report.Extra == nil {
		report.Extra = make(map[string]any)
	}
	report.Extra["cache"] = map[string]any{"age_seconds": int(age.Seconds()), "stale": stale}
}

// markRateLimited records why a report could not be refreshed, so the CLI, the
// dashboard, and --json all show that the data is deliberately frozen.
func markRateLimited(report *provider.Usage, retryAt time.Time) {
	if report.Extra == nil {
		report.Extra = make(map[string]any)
	}
	report.Extra["rate_limit"] = map[string]any{"retry_at": retryAt.UTC()}
}
