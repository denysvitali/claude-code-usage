// Package kimi implements the Kimi API provider for llm-usage.
package kimi

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/denysvitali/llm-usage/provider"
)

const (
	featureCoding                   = "FEATURE_CODING"
	subscriptionStatusActive        = "SUBSCRIPTION_STATUS_ACTIVE"
	subscriptionStatusActiveDisplay = "Active"
	membershipLevelBasic            = "LEVEL_BASIC"
	membershipLevelBasicDisplay     = "Basic"

	// APIKeyEnvVar is the environment variable used as API key fallback when
	// no stored credentials exist.
	APIKeyEnvVar = "KIMI_CODE_API_KEY"
)

// Provider implements the provider.Provider interface for Kimi
type Provider struct {
	client     *Client
	captureRaw bool
}

// NewProvider creates a new Kimi provider with the given API key
func NewProvider(apiKey string, captureRaw bool) *Provider {
	return &Provider{
		client:     NewClient(apiKey),
		captureRaw: captureRaw,
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
	resp, raw, err := p.client.GetUsage(ctx)
	if err != nil {
		return nil, err
	}

	windows := make([]provider.UsageWindow, 0, 1+len(resp.Limits))

	// Main coding quota window
	if mainWindow := p.parseDetailWindow("Coding", resp.Usage); mainWindow != nil {
		windows = append(windows, *mainWindow)
	}

	// Rate limit windows
	for _, limit := range resp.Limits {
		if limitWindow := p.parseLimitWindow(limit); limitWindow != nil {
			windows = append(windows, *limitWindow)
		}
	}

	usage := &provider.Usage{
		Provider: "kimi",
		Windows:  windows,
	}

	if p.captureRaw {
		if usage.Extra == nil {
			usage.Extra = make(map[string]any)
		}
		usage.Extra["raw"] = json.RawMessage(raw)
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

// parseDetailWindow parses a usage detail into a UsageWindow with the given label
func (p *Provider) parseDetailWindow(label string, detail UsageDetail) *provider.UsageWindow {
	limit, err := strconv.ParseFloat(detail.Limit, 64)
	if err != nil {
		return nil
	}

	used, err := strconv.ParseFloat(detail.Used, 64)
	if err != nil {
		return nil
	}

	var utilization float64
	if limit > 0 {
		utilization = (used / limit) * 100
	}

	var resetsAt *time.Time
	if detail.ResetTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, detail.ResetTime); err == nil {
			resetsAt = &t
		}
	}

	remaining := limit - used
	if detail.Remaining != "" {
		if r, err := strconv.ParseFloat(detail.Remaining, 64); err == nil {
			remaining = r
		}
	}

	return &provider.UsageWindow{
		Label:       label,
		Utilization: utilization,
		ResetsAt:    resetsAt,
		Limit:       &limit,
		Used:        &used,
		Remaining:   &remaining,
	}
}

// parseLimitWindow parses a rate limit item into a UsageWindow
func (p *Provider) parseLimitWindow(limit LimitItem) *provider.UsageWindow {
	label := p.formatDurationLabel(limit.Window.Duration, limit.Window.TimeUnit)
	return p.parseDetailWindow(label, limit.Detail)
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
	case subscriptionStatusActive:
		return subscriptionStatusActiveDisplay
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
	case membershipLevelBasic:
		return membershipLevelBasicDisplay
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
