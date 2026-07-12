// Package codex implements usage reporting for the OpenAI Codex CLI account.
package codex

import (
	"context"
	"encoding/json"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/provider"
)

// Provider fetches usage for a single Codex CLI account.
type Provider struct {
	client     Client
	captureRaw bool
}

// NewProvider creates a Codex provider from an access token and optional account ID.
func NewProvider(accessToken, accountID string, captureRaw bool) *Provider {
	return &Provider{client: Client{AccessToken: accessToken, AccountID: accountID}, captureRaw: captureRaw}
}

// Name returns the provider display name.
func (p *Provider) Name() string { return "Codex (OpenAI)" }

// ShortName returns the provider short label.
func (p *Provider) ShortName() string { return "Cdx" }

// ID returns the provider unique identifier.
func (p *Provider) ID() string { return credentials.ProviderCodex }

// GetUsage fetches the current usage from the Codex API.
func (p *Provider) GetUsage(ctx context.Context) (*provider.Usage, error) {
	data, raw, err := p.client.GetUsage(ctx)
	if err != nil {
		return nil, err
	}
	result := &provider.Usage{Provider: credentials.ProviderCodex}
	if data.RateLimit.PrimaryWindow != nil {
		result.Windows = append(result.Windows, toWindow("5-Hour", *data.RateLimit.PrimaryWindow))
	}
	if data.RateLimit.SecondaryWindow != nil {
		result.Windows = append(result.Windows, toWindow("7-Day", *data.RateLimit.SecondaryWindow))
	}
	if p.captureRaw {
		if result.Extra == nil {
			result.Extra = make(map[string]any)
		}
		result.Extra["raw"] = json.RawMessage(raw)
	}
	return result, nil
}

func toWindow(label string, w Window) provider.UsageWindow {
	return provider.UsageWindow{Label: label, Utilization: w.UsedPercent, ResetsAt: w.ResetTime()}
}
