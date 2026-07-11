// Package minimax implements the MiniMax API provider for llm-usage.
package minimax

import (
	"time"

	"github.com/denysvitali/llm-usage/internal/cache"
	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/provider"
)

const (
	subscriptionCacheTTL = 30 * time.Minute
)

// Provider implements the provider.Provider interface for MiniMax
type Provider struct {
	client *Client
	cache  *cache.Manager
}

// NewProvider creates a new MiniMax provider with the given cookie and group ID
func NewProvider(cookie, groupID string) *Provider {
	return &Provider{
		client: NewClient(cookie, groupID),
		cache:  cache.NewManager(),
	}
}

// Name returns the provider's display name
func (p *Provider) Name() string {
	return credentials.ProviderDisplayName(credentials.ProviderMiniMax)
}

// ShortName returns the provider's compact label
func (p *Provider) ShortName() string {
	return "M"
}

// ID returns the provider's unique identifier
func (p *Provider) ID() string {
	return "minimax"
}

// GetUsage fetches current usage statistics from MiniMax
func (p *Provider) GetUsage() (*provider.Usage, error) {
	resp, err := p.client.GetUsage()
	if err != nil {
		return nil, err
	}

	windows := make([]provider.UsageWindow, 0)

	for _, item := range resp.ModelRemains {
		window := p.parseModelRemain(item)
		if window != nil {
			windows = append(windows, *window)
		}
	}

	usage := &provider.Usage{
		Provider: "minimax",
		Windows:  windows,
	}

	// Fetch subscription info (with caching)
	if sub := p.getSubscription(); sub != nil {
		if usage.Extra == nil {
			usage.Extra = make(map[string]any)
		}
		usage.Extra["subscription"] = map[string]any{
			"status": sub.BaseResp.StatusMsg,
		}
	}

	return usage, nil
}

// parseModelRemain parses a ModelRemain item into a UsageWindow
func (p *Provider) parseModelRemain(item ModelRemain) *provider.UsageWindow {
	// Convert Unix milliseconds to time.Time
	// end_time is when the window resets
	resetsAt := time.UnixMilli(item.EndTime)

	// Calculate utilization
	total := float64(item.CurrentIntervalTotalCount)
	used := float64(item.CurrentIntervalUsageCount)
	remaining := float64(item.RemainsTime)

	var utilization float64
	if total > 0 {
		// Calculate remaining percentage (inverse of used)
		utilization = ((total - used) / total) * 100
	}

	label := item.ModelName
	if label == "" {
		label = "MiniMax"
	}

	return &provider.UsageWindow{
		Label:       label,
		Utilization: utilization,
		ResetsAt:    &resetsAt,
		Limit:       &total,
		Used:        &used,
		Remaining:   &remaining,
	}
}

// getSubscription fetches subscription info with caching
func (p *Provider) getSubscription() *SubscriptionResponse {
	cacheKey := cache.HashKey("minimax_subscription", p.client.Cookie()+p.client.GroupID())

	// Try to get from cache
	var cached SubscriptionResponse
	if found, err := p.cache.Get(cacheKey, &cached); err == nil && found {
		return &cached
	}

	// Fetch from API
	sub, err := p.client.GetSubscription()
	if err != nil {
		return nil
	}

	// Cache the result
	_ = p.cache.Set(cacheKey, sub, subscriptionCacheTTL)

	return sub
}
