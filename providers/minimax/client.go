package minimax

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL              = "https://platform.minimax.io"
	codingPlanEndpoint   = "/v1/api/openplatform/coding_plan/remains"
	subscriptionEndpoint = "/v1/api/openplatform/charge/combo/cycle_audio_resource_package"
	userAgent            = "llm-usage/1.0.0"
)

// Client is an HTTP client for the MiniMax API
type Client struct {
	httpClient *http.Client
	cookie     string
	groupID    string
}

// NewClient creates a new API client with cookie-based authentication
func NewClient(cookie, groupID string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cookie:  cookie,
		groupID: groupID,
	}
}

// GetUsage fetches the current usage from the coding_plan/remains endpoint
func (c *Client) GetUsage() (*CodingPlanResponse, error) {
	// Build URL with GroupId query parameter
	reqURL, err := url.Parse(baseURL + codingPlanEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	query := reqURL.Query()
	query.Add("GroupId", c.groupID)
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Cookie", c.cookie)
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

	var usage CodingPlanResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &usage, nil
}

// GetSubscription fetches the subscription details from the subscription endpoint
func (c *Client) GetSubscription() (*SubscriptionResponse, error) {
	// Build URL with query parameters
	reqURL, err := url.Parse(baseURL + subscriptionEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	query := reqURL.Query()
	query.Add("GroupId", c.groupID)
	query.Add("biz_line", "2")
	query.Add("cycle_type", "3")
	query.Add("resource_package_type", "7")
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Cookie", c.cookie)
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

// Cookie returns the cookie for cache key generation
func (c *Client) Cookie() string {
	return c.cookie
}

// GroupID returns the group ID for cache key generation
func (c *Client) GroupID() string {
	return c.groupID
}
