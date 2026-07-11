package providers

import (
	"context"
	"fmt"

	"github.com/denysvitali/llm-usage/provider"
	publicgrok "github.com/denysvitali/llm-usage/providers/grok"
)

type grokAdapter struct {
	client *publicgrok.Client
	err    error
}

func newGrokAdapter(token string) *grokAdapter {
	client, err := publicgrok.NewClient(publicgrok.ClientOptions{AccessToken: token})
	return &grokAdapter{client: client, err: err}
}

func (p *grokAdapter) Name() string      { return "Grok (xAI)" }
func (p *grokAdapter) ShortName() string { return "G" }
func (p *grokAdapter) ID() string        { return "grok" }
func (p *grokAdapter) GetUsage(ctx context.Context) (*provider.Usage, error) {
	if p.err != nil {
		return nil, fmt.Errorf("configure Grok client: %w", p.err)
	}
	return p.client.GetUsage(ctx)
}
