package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	usageURL          = "https://chatgpt.com/backend-api/wham/usage"
	errPrefixRequest  = "codex usage request"
	errPrefixResponse = "codex usage response"
)

// Client communicates with the Codex usage API.
type Client struct {
	// HTTPClient is used to make requests; if nil, http.DefaultClient is used.
	HTTPClient *http.Client
	// AccessToken is the bearer token used for authentication.
	AccessToken string
	// AccountID is an optional ChatGPT account ID sent as a request header.
	AccountID string
}

// GetUsage fetches the current usage from the Codex API.
func (c Client) GetUsage(ctx context.Context) (*UsageResponse, []byte, error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usageURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	if c.AccountID != "" {
		req.Header.Set("ChatGPT-Account-ID", c.AccountID)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%s failed: %w", errPrefixRequest, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%s returned HTTP %d", errPrefixRequest, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", errPrefixResponse, err)
	}
	var result UsageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("decode %s: %w", errPrefixResponse, err)
	}
	return &result, body, nil
}
