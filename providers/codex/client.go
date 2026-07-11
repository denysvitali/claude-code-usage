package codex

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const usageURL = "https://chatgpt.com/backend-api/wham/usage"

type Client struct {
	HTTPClient             *http.Client
	AccessToken, AccountID string
}

func (c Client) GetUsage() (*UsageResponse, error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodGet, usageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	if c.AccountID != "" {
		req.Header.Set("ChatGPT-Account-ID", c.AccountID)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Codex usage request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Codex usage request returned HTTP %d", resp.StatusCode)
	}
	var result UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Codex usage response: %w", err)
	}
	return &result, nil
}
