package claude

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
	tokenEndpoint = "https://console.anthropic.com/v1/oauth/token" //nolint:gosec // OAuth endpoint URL, not a credential
	// OAuth client ID used by the Claude CLI / Claude Code. This is a public
	// identifier, not a secret.
	oauthClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e" //gitleaks:allow
)

// RefreshedToken holds the result of an OAuth token refresh.
type RefreshedToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64 // Unix milliseconds
}

// RefreshAccessToken exchanges a refresh token for a new access token.
func RefreshAccessToken(refreshToken string) (*RefreshedToken, error) {
	payload, err := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     oauthClientID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute refresh request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}
	if result.AccessToken == "" {
		return nil, fmt.Errorf("token refresh returned no access token")
	}

	return &RefreshedToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).UnixMilli(),
	}, nil
}
