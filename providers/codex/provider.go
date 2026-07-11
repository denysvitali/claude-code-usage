// Package codex implements usage reporting for the OpenAI Codex CLI account.
package codex

import (
	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/provider"
)

type Provider struct{ client Client }

func NewProvider(accessToken, accountID string) *Provider {
	return &Provider{client: Client{AccessToken: accessToken, AccountID: accountID}}
}
func (p *Provider) Name() string      { return credentials.ProviderDisplayName(credentials.ProviderCodex) }
func (p *Provider) ShortName() string { return "Cdx" }
func (p *Provider) ID() string        { return credentials.ProviderCodex }

func (p *Provider) GetUsage() (*provider.Usage, error) {
	data, err := p.client.GetUsage()
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
	return result, nil
}

func toWindow(label string, w Window) provider.UsageWindow {
	return provider.UsageWindow{Label: label, Utilization: w.UsedPercent, ResetsAt: w.ResetTime()}
}
