// Package claude implements the Claude API provider for llm-usage.
package claude

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL       = "https://api.anthropic.com"
	usageEndpoint = "/api/oauth/usage"
	userAgent     = "llm-usage/1.0.0"
	betaHeader    = "oauth-2025-04-20"
)

// Client is an HTTP client for the Anthropic OAuth API
type Client struct {
	httpClient  *http.Client
	accessToken string
}

// NewClient creates a new API client with the given access token
func NewClient(accessToken string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken: accessToken,
	}
}

// GetUsage fetches the current usage from the OAuth usage endpoint
func (c *Client) GetUsage() (*UsageResponse, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+usageEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("anthropic-beta", betaHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var usage UsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &usage, nil
}
