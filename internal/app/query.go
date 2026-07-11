// Package app contains application services shared by CLI and HTTP clients.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/usage"
	"github.com/denysvitali/llm-usage/provider"
)

// QueryOptions controls one usage query.
type QueryOptions struct {
	Providers   string
	Account     string
	AllAccounts bool
	Debug       bool
	Timeout     time.Duration
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
	providers, failures := usage.GetProviders(opts.Providers, opts.Account, opts.AllAccounts, opts.Debug, s.Credentials)
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
		stats := usage.FetchAllUsage(providers)
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
