// Package kimi implements the Kimi API provider for llm-usage.
package kimi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/denysvitali/llm-usage/provider"
)

// Provider implements the provider.Provider interface for Kimi
type Provider struct {
	client *Client
}

// NewProvider creates a new Kimi provider with the given API key
func NewProvider(apiKey string) *Provider {
	return &Provider{
		client: NewClient(apiKey),
	}
}

// Name returns the provider's display name
func (p *Provider) Name() string {
	return "Kimi"
}

// ShortName returns the provider's compact label
func (p *Provider) ShortName() string {
	return "K"
}

// ID returns the provider's unique identifier
func (p *Provider) ID() string {
	return "kimi"
}

// GetUsage fetches current usage statistics from Kimi
func (p *Provider) GetUsage(ctx context.Context) (*provider.Usage, error) {
	resp, err := p.client.GetUsage(ctx)
	if err != nil {
		return nil, err
	}

	windows := make([]provider.UsageWindow, 0)

	for _, item := range resp.Usages {
		// Add main scope window
		if scopeWindow := p.parseScopeWindow(item); scopeWindow != nil {
			windows = append(windows, *scopeWindow)
		}

		// Add rate limit windows
		for _, limit := range item.Limits {
			if limitWindow := p.parseLimitWindow(item.Scope, limit); limitWindow != nil {
				windows = append(windows, *limitWindow)
			}
		}
	}

	usage := &provider.Usage{
		Provider: "kimi",
		Windows:  windows,
	}

	// Fetch subscription info (with caching)
	if sub, err := p.client.GetSubscription(ctx); err == nil {
		if usage.Extra == nil {
			usage.Extra = make(map[string]any)
		}
		usage.Extra["subscription"] = p.formatSubscriptionExtra(sub)
	}

	return usage, nil
}

// parseScopeWindow parses the main scope usage detail into a UsageWindow
func (p *Provider) parseScopeWindow(item UsageItem) *provider.UsageWindow {
	limit, err := strconv.ParseFloat(item.Detail.Limit, 64)
	if err != nil {
		return nil
	}

	used, err := strconv.ParseFloat(item.Detail.Used, 64)
	if err != nil {
		return nil
	}

	utilization := (used / limit) * 100

	var resetsAt *time.Time
	if item.Detail.ResetTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, item.Detail.ResetTime); err == nil {
			resetsAt = &t
		}
	}

	remaining := limit - used

	return &provider.UsageWindow{
		Label:       p.formatScopeLabel(item.Scope),
		Utilization: utilization,
		ResetsAt:    resetsAt,
		Limit:       &limit,
		Used:        &used,
		Remaining:   &remaining,
	}
}

// parseLimitWindow parses a rate limit item into a UsageWindow
func (p *Provider) parseLimitWindow(_ string, limit LimitItem) *provider.UsageWindow {
	limitVal, err := strconv.ParseFloat(limit.Detail.Limit, 64)
	if err != nil {
		return nil
	}

	usedVal, err := strconv.ParseFloat(limit.Detail.Used, 64)
	if err != nil {
		return nil
	}

	utilization := (usedVal / limitVal) * 100

	var resetsAt *time.Time
	if limit.Detail.ResetTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, limit.Detail.ResetTime); err == nil {
			resetsAt = &t
		}
	}

	remaining := limitVal - usedVal

	label := p.formatDurationLabel(limit.Window.Duration, limit.Window.TimeUnit)

	return &provider.UsageWindow{
		Label:       label,
		Utilization: utilization,
		ResetsAt:    resetsAt,
		Limit:       &limitVal,
		Used:        &usedVal,
		Remaining:   &remaining,
	}
}

// formatScopeLabel formats the scope name for display
func (p *Provider) formatScopeLabel(scope string) string {
	// Convert FEATURE_CODING to "Feature Coding"
	parts := strings.Split(scope, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

// formatDurationLabel formats the window duration for display
func (p *Provider) formatDurationLabel(duration int, timeUnit string) string {
	// Convert TIME_UNIT_MINUTE to "5-Min Rate Limit"
	unit := strings.ToLower(strings.TrimPrefix(timeUnit, "TIME_UNIT_"))
	unit = strings.TrimSuffix(unit, "s") // Remove plural

	// Capitalize first letter
	if len(unit) > 0 {
		unit = strings.ToUpper(unit[:1]) + unit[1:]
	}

	return fmt.Sprintf("%d-%s Rate Limit", duration, unit)
}

// formatSubscriptionExtra formats subscription data for the Extra map
func (p *Provider) formatSubscriptionExtra(sub *SubscriptionResponse) map[string]any {
	result := map[string]any{
		"subscribed": sub.Subscribed,
	}

	if sub.Subscription != nil {
		// Parse expiry date
		var expiresAt *time.Time
		if sub.Subscription.CurrentEndTime != "" {
			if t, err := time.Parse(time.RFC3339Nano, sub.Subscription.CurrentEndTime); err == nil {
				expiresAt = &t
			}
		}

		// Format status for display
		status := formatSubscriptionStatus(sub.Subscription.Status)

		plan := map[string]any{
			"title":  sub.Subscription.Goods.Title,
			"level":  formatMembershipLevel(sub.Subscription.Goods.MembershipLevel),
			"status": status,
		}
		result["plan"] = plan

		if expiresAt != nil {
			result["expires_at"] = expiresAt.Format(time.RFC3339)
		}
	}

	if len(sub.Memberships) > 0 {
		features := make([]map[string]any, 0, len(sub.Memberships))
		for _, m := range sub.Memberships {
			features = append(features, map[string]any{
				"feature": formatFeatureName(m.Feature),
				"left":    m.LeftCount,
				"total":   m.TotalCount,
			})
		}
		result["features"] = features
	}

	return result
}

// formatSubscriptionStatus converts status constants to display strings
func formatSubscriptionStatus(status string) string {
	switch status {
	case "SUBSCRIPTION_STATUS_ACTIVE":
		return "Active"
	case "SUBSCRIPTION_STATUS_CANCELLED":
		return "Cancelled"
	case "SUBSCRIPTION_STATUS_EXPIRED":
		return "Expired"
	default:
		return strings.TrimPrefix(status, "SUBSCRIPTION_STATUS_")
	}
}

// formatMembershipLevel converts level constants to display strings
func formatMembershipLevel(level string) string {
	switch level {
	case "LEVEL_BASIC":
		return "Basic"
	case "LEVEL_STANDARD":
		return "Standard"
	case "LEVEL_PREMIUM":
		return "Premium"
	default:
		return strings.TrimPrefix(level, "LEVEL_")
	}
}

// formatFeatureName converts feature constants to display strings
func formatFeatureName(feature string) string {
	// Convert FEATURE_CODING to "Coding"
	name := strings.TrimPrefix(feature, "FEATURE_")
	if len(name) > 0 {
		return strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
	}
	return name
}
