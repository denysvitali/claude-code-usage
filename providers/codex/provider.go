// Package codex implements usage reporting for the OpenAI Codex CLI account.
package codex

import (
	"context"

	"github.com/denysvitali/llm-usage/provider"
)

type Provider struct{ client Client }

func NewProvider(accessToken, accountID string) *Provider {
	return &Provider{client: Client{AccessToken: accessToken, AccountID: accountID}}
}
func (p *Provider) Name() string      { return "Codex (OpenAI)" }
func (p *Provider) ShortName() string { return "Cdx" }
func (p *Provider) ID() string        { return "codex" }

func (p *Provider) GetUsage(ctx context.Context) (*provider.Usage, error) {
	data, err := p.client.GetUsage(ctx)
	if err != nil {
		return nil, err
	}
	result := &provider.Usage{Provider: "codex"}
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
