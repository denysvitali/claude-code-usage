package kimi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL              = "https://www.kimi.com"
	usageEndpoint        = "/apiv2/kimi.gateway.billing.v1.BillingService/GetUsages"
	subscriptionEndpoint = "/apiv2/kimi.gateway.order.v1.SubscriptionService/GetSubscription"
	userAgent            = "llm-usage/1.0.0"
)

// Client is an HTTP client for the Kimi API
type Client struct {
	httpClient *http.Client
	apiKey     string
}

// NewClient creates a new API client with the given API key
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}
}

// usageRequest represents the request body for the usage endpoint
type usageRequest struct {
	Scope []string `json:"scope"`
}

// GetUsage fetches the current usage from the usage endpoint
func (c *Client) GetUsage(ctx context.Context) (*UsageResponse, []byte, error) {
	reqBody := usageRequest{
		Scope: []string{featureCoding},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+usageEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var usage UsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &usage, body, nil
}

// GetSubscription fetches the subscription details from the subscription endpoint
func (c *Client) GetSubscription(ctx context.Context) (*SubscriptionResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+subscriptionEndpoint, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var subscription SubscriptionResponse
	if err := json.Unmarshal(body, &subscription); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &subscription, nil
}

// APIKey returns the API key for cache key generation
func (c *Client) APIKey() string {
	return c.apiKey
}
