package claude

import (
	"time"

	"github.com/denysvitali/llm-usage/internal/provider"
)

// Provider implements the provider.Provider interface for Claude
type Provider struct {
	client *Client
}

// NewProvider creates a new Claude provider with the given access token
func NewProvider(accessToken string) *Provider {
	return &Provider{
		client: NewClient(accessToken),
	}
}

// Name returns the provider's display name
func (p *Provider) Name() string {
	return "Claude"
}

// ID returns the provider's unique identifier
func (p *Provider) ID() string {
	return "claude"
}

// GetUsage fetches current usage statistics from Claude
func (p *Provider) GetUsage() (*provider.Usage, error) {
	usage, err := p.client.GetUsage()
	if err != nil {
		return nil, err
	}

	windows := make([]provider.UsageWindow, 0)

	if usage.FiveHour != nil {
		windows = append(windows, provider.UsageWindow{
			Label:       "5-Hour",
			Utilization: usage.FiveHour.Utilization,
			ResetsAt:    usage.FiveHour.ResetsAt,
		})
	}

	if usage.SevenDay != nil {
		windows = append(windows, provider.UsageWindow{
			Label:       "7-Day",
			Utilization: usage.SevenDay.Utilization,
			ResetsAt:    usage.SevenDay.ResetsAt,
		})
	}

	if usage.SevenDaySonnet != nil {
		windows = append(windows, provider.UsageWindow{
			Label:       "7-Day Sonnet",
			Utilization: usage.SevenDaySonnet.Utilization,
			ResetsAt:    usage.SevenDaySonnet.ResetsAt,
		})
	}

	if usage.SevenDayOpus != nil {
		windows = append(windows, provider.UsageWindow{
			Label:       "7-Day Opus",
			Utilization: usage.SevenDayOpus.Utilization,
			ResetsAt:    usage.SevenDayOpus.ResetsAt,
		})
	}

	if usage.SevenDayOAuthApp != nil {
		windows = append(windows, provider.UsageWindow{
			Label:       "7-Day OAuth Apps",
			Utilization: usage.SevenDayOAuthApp.Utilization,
			ResetsAt:    usage.SevenDayOAuthApp.ResetsAt,
		})
	}

	if usage.IguanaNecktie != nil {
		windows = append(windows, provider.UsageWindow{
			Label:       "Iguana Necktie",
			Utilization: usage.IguanaNecktie.Utilization,
			ResetsAt:    usage.IguanaNecktie.ResetsAt,
		})
	}

	extra := make(map[string]interface{})
	if usage.ExtraUsage != nil && usage.ExtraUsage.IsEnabled {
		extra["extra_usage"] = map[string]interface{}{
			"is_enabled":    usage.ExtraUsage.IsEnabled,
			"monthly_limit": usage.ExtraUsage.MonthlyLimit,
			"used_credits":  usage.ExtraUsage.UsedCredits,
			"utilization":   usage.ExtraUsage.Utilization,
		}
	}

	return &provider.Usage{
		Provider: "claude",
		Windows:  windows,
		Extra:    extra,
	}, nil
}

// IsExpired checks if the token has expired
func IsExpired(expiresAt int64) bool {
	return time.Now().After(time.UnixMilli(expiresAt))
}

// ExpiresIn returns the duration until the token expires
func ExpiresIn(expiresAt int64) time.Duration {
	return time.Until(time.UnixMilli(expiresAt))
}
