// Package zai implements the Z.AI API provider for llm-usage.
package zai

import (
	"fmt"

	"github.com/dvitali/llm-usage/internal/provider"
)

// Provider implements the provider.Provider interface for Z.AI
type Provider struct {
	apiKey string
}

// API endpoint reference:
// https://z.ai/manage-apikey/rate-limits
// TODO: Research the rate-limits endpoint and implement authentication

// NewProvider creates a new Z.AI provider with the given API key
func NewProvider(apiKey string) *Provider {
	return &Provider{
		apiKey: apiKey,
	}
}

// Name returns the provider's display name
func (p *Provider) Name() string {
	return "Z.AI"
}

// ID returns the provider's unique identifier
func (p *Provider) ID() string {
	return "zai"
}

// GetUsage fetches current usage statistics from Z.AI
func (p *Provider) GetUsage() (*provider.Usage, error) {
	// TODO: Implement Z.AI API call
	// The reference URL is: https://z.ai/manage-apikey/rate-limits
	// Need to research the actual API endpoint and authentication method
	return nil, fmt.Errorf("Z.AI provider not yet implemented - API endpoint needs research, implementation pending")
}
