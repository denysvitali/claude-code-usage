package claude

import (
	"encoding/json"
	"time"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/provider"
)

// Provider implements the provider.Provider interface for Claude
type Provider struct {
	client *Client
	debug  bool
}

// NewProvider creates a new Claude provider with the given access token.
// If debug is true, the raw API response is included in Usage.Extra["raw"].
func NewProvider(accessToken string, debug bool) *Provider {
	return &Provider{
		client: NewClient(accessToken),
		debug:  debug,
	}
}

// Name returns the provider's display name
func (p *Provider) Name() string {
	return credentials.ProviderDisplayName(credentials.ProviderClaude)
}

// ShortName returns the provider's compact label
func (p *Provider) ShortName() string {
	return "C"
}

// ID returns the provider's unique identifier
func (p *Provider) ID() string {
	return "claude"
}

// GetUsage fetches current usage statistics from Claude
func (p *Provider) GetUsage() (*provider.Usage, error) {
	usage, raw, err := p.client.GetUsage()
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

	for _, limit := range usage.Limits {
		if limit.Scope == nil || limit.Scope.Model == nil || limit.Scope.Model.DisplayName == "" {
			continue
		}
		windows = append(windows, provider.UsageWindow{
			Label:       limitGroupLabel(limit.Group) + " " + limit.Scope.Model.DisplayName,
			Utilization: limit.Percent,
			ResetsAt:    limit.ResetsAt,
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

	if p.debug {
		extra["raw"] = json.RawMessage(raw)
	}

	return &provider.Usage{
		Provider: "claude",
		Windows:  windows,
		Extra:    extra,
	}, nil
}

// limitGroupLabel maps a UsageLimit's group to the same label prefix used
// for the equivalent top-level window (e.g. "session" -> "5-Hour").
func limitGroupLabel(group string) string {
	switch group {
	case "session":
		return "5-Hour"
	case "weekly":
		return "7-Day"
	default:
		return group
	}
}

// IsExpired checks if the token has expired
func IsExpired(expiresAt int64) bool {
	return time.Now().After(time.UnixMilli(expiresAt))
}

// ExpiresIn returns the duration until the token expires
func ExpiresIn(expiresAt int64) time.Duration {
	return time.Until(time.UnixMilli(expiresAt))
}
