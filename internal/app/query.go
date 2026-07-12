// Package app contains application services shared by CLI and HTTP clients.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/denysvitali/llm-usage/internal/cache"
	"github.com/denysvitali/llm-usage/internal/config"
	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/usage"
	"github.com/denysvitali/llm-usage/provider"
)

// QueryOptions controls one usage query.
type QueryOptions struct {
	Providers    string
	Account      string
	AllAccounts  bool
	Debug        bool
	Raw          bool
	Timeout      time.Duration
	CacheTTL     time.Duration
	StaleIfError bool
	Config       *config.Config
}

// QueryService is the shared usage orchestration service.
type QueryService struct {
	Credentials *credentials.Manager
}

// Query fetches all selected providers concurrently and preserves partial results.
func (s QueryService) Query(ctx context.Context, opts QueryOptions) (*provider.UsageStats, error) {
	if s.Credentials == nil {
		return nil, fmt.Errorf("credentials manager is required")
	}
	providerSelection := opts.Providers
	if (providerSelection == "" || providerSelection == "all") && opts.Config != nil && len(opts.Config.Providers) > 0 {
		ids := make([]string, 0, len(opts.Config.Providers))
		for _, configured := range opts.Config.Providers {
			ids = append(ids, configured.ID)
		}
		providerSelection = strings.Join(ids, ",")
	}
	configuredAccounts := make(map[string][]string)
	if opts.Config != nil {
		for _, configured := range opts.Config.Providers {
			for _, account := range configured.Accounts {
				configuredAccounts[configured.ID] = append(configuredAccounts[configured.ID], account.Name)
			}
		}
	}
	providers, failures := usage.GetProviders(providerSelection, opts.Account, opts.AllAccounts, opts.Debug, opts.Raw, configuredAccounts, s.Credentials)
	if len(providers) == 0 && len(failures) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	queryCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	result := make(chan *provider.UsageStats, 1)
	go func() {
		var cacheManager *cache.Manager
		if opts.CacheTTL > 0 {
			cacheManager = cache.NewManager()
		}
		stats := usage.FetchAllUsage(queryCtx, providers, usage.CacheOptions{Manager: cacheManager, TTL: opts.CacheTTL, StaleIfError: opts.StaleIfError})
		stats.Providers = append(stats.Providers, failures...)
		result <- stats
	}()

	select {
	case stats := <-result:
		return stats, nil
	case <-queryCtx.Done():
		return nil, fmt.Errorf("usage query timed out: %w", queryCtx.Err())
	}
}
