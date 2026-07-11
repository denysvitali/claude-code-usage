// Package grok provides a context-aware client for Grok Build billing usage.
package grok

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/denysvitali/llm-usage/provider"
)

const DefaultBillingEndpoint = "https://cli-chat-proxy.grok.com/v1/billing?format=credits"

type ClientOptions struct {
	AccessToken string
	HTTPClient  *http.Client
	Endpoint    string
}

type Client struct {
	accessToken string
	httpClient  *http.Client
	endpoint    string
}

func NewClient(options ClientOptions) (*Client, error) {
	if strings.TrimSpace(options.AccessToken) == "" {
		return nil, fmt.Errorf("Grok access token is required")
	}
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	if options.Endpoint == "" {
		options.Endpoint = DefaultBillingEndpoint
	}
	return &Client{accessToken: options.AccessToken, httpClient: options.HTTPClient, endpoint: options.Endpoint}, nil
}

func (c *Client) GetUsage(ctx context.Context) (*provider.Usage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create Grok billing request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("User-Agent", "grok-build")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Grok billing request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Grok billing request returned HTTP %d", response.StatusCode)
	}
	var billing billingResponse
	if err := json.NewDecoder(response.Body).Decode(&billing); err != nil {
		return nil, fmt.Errorf("decode Grok billing response: %w", err)
	}
	return normalize(billing)
}

type billingResponse struct {
	Config billingConfig `json:"config"`
}
type billingConfig struct {
	CurrentPeriod      billingPeriod  `json:"currentPeriod"`
	CreditUsagePercent float64        `json:"creditUsagePercent"`
	ProductUsage       []productUsage `json:"productUsage"`
}
type billingPeriod struct {
	End string `json:"end"`
}
type productUsage struct {
	Product      string  `json:"product"`
	UsagePercent float64 `json:"usagePercent"`
}

func normalize(billing billingResponse) (*provider.Usage, error) {
	reset, err := time.Parse(time.RFC3339Nano, billing.Config.CurrentPeriod.End)
	if err != nil {
		return nil, fmt.Errorf("parse Grok billing period end: %w", err)
	}
	percent, label := billing.Config.CreditUsagePercent, "Weekly Limit"
	for _, product := range billing.Config.ProductUsage {
		if product.Product == "GrokBuild" {
			percent, label = product.UsagePercent, "Grok Build Weekly"
			break
		}
	}
	return &provider.Usage{Provider: "grok", Windows: []provider.UsageWindow{{Label: label, Utilization: percent, ResetsAt: &reset}}}, nil
}
