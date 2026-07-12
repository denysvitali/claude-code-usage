// Package grok provides a context-aware client for Grok Build billing usage.
package grok

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/denysvitali/llm-usage/provider"
)

// DefaultBillingEndpoint is the Grok Build billing endpoint used by default.
const DefaultBillingEndpoint = "https://cli-chat-proxy.grok.com/v1/billing?format=credits"

// ClientOptions configures a new Grok billing client.
type ClientOptions struct {
	// AccessToken is the bearer token used to authenticate billing requests.
	AccessToken string
	// HTTPClient is the client used to make requests; nil uses http.DefaultClient.
	HTTPClient *http.Client
	// Endpoint overrides the default billing endpoint when non-empty.
	Endpoint string
	// CaptureRaw requests that GetUsage include the raw API response in Extra.
	CaptureRaw bool
}

// Client communicates with the Grok Build billing endpoint.
type Client struct {
	accessToken string
	httpClient  *http.Client
	endpoint    string
	captureRaw  bool
}

// NewClient creates a Grok billing client from the supplied options.
func NewClient(options ClientOptions) (*Client, error) {
	if strings.TrimSpace(options.AccessToken) == "" {
		return nil, fmt.Errorf("grok access token is required")
	}
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	if options.Endpoint == "" {
		options.Endpoint = DefaultBillingEndpoint
	}
	return &Client{accessToken: options.AccessToken, httpClient: options.HTTPClient, endpoint: options.Endpoint, captureRaw: options.CaptureRaw}, nil
}

// GetUsage fetches the current Grok Build usage from the configured endpoint.
func (c *Client) GetUsage(ctx context.Context) (*provider.Usage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create Grok billing request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("User-Agent", "grok-build")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("grok billing request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grok billing request returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read Grok billing response: %w", err)
	}
	var billing billingResponse
	if err := json.Unmarshal(body, &billing); err != nil {
		return nil, fmt.Errorf("decode Grok billing response: %w", err)
	}
	usage, err := normalize(billing)
	if err != nil {
		return nil, err
	}
	if c.captureRaw {
		if usage.Extra == nil {
			usage.Extra = make(map[string]any)
		}
		usage.Extra["raw"] = json.RawMessage(body)
	}
	return usage, nil
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
