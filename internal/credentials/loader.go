// Package credentials handles loading OAuth credentials from the Claude CLI.
package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/denysvitali/claude-code-usage/internal/keychain"
)

// Credentials represents the structure of ~/.claude/.credentials.json
type Credentials struct {
	ClaudeAiOauth *OAuthCredentials `json:"claudeAiOauth"`
}

// OAuthCredentials contains the OAuth token information
type OAuthCredentials struct {
	AccessToken   string   `json:"accessToken"`
	RefreshToken  string   `json:"refreshToken"`
	ExpiresAt     int64    `json:"expiresAt"`
	Scopes        []string `json:"scopes"`
	RateLimitTier string   `json:"rateLimitTier"`
}

// IsExpired checks if the access token has expired
func (o *OAuthCredentials) IsExpired() bool {
	expiresAt := time.UnixMilli(o.ExpiresAt)
	return time.Now().After(expiresAt)
}

// ExpiresIn returns the duration until the token expires
func (o *OAuthCredentials) ExpiresIn() time.Duration {
	expiresAt := time.UnixMilli(o.ExpiresAt)
	return time.Until(expiresAt)
}

// Load reads credentials from the platform-specific storage.
// On macOS, it attempts to use the keychain first, then falls back to the file.
// On other platforms, it reads from the credentials file.
func Load() (*Credentials, error) {
	// Try keychain first on all platforms
	data, err := keychain.Load()
	if err == nil {
		return parseCredentials(data)
	}
	// If keychain load fails (not supported or not found), fall through to file-based loading

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	credPath := filepath.Join(homeDir, ".claude", ".credentials.json")
	return LoadFromPath(credPath)
}

// parseCredentials parses credentials from JSON data
func parseCredentials(data []byte) (*Credentials, error) {
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	if creds.ClaudeAiOauth == nil {
		return nil, fmt.Errorf("no OAuth credentials found - please run 'claude' to authenticate")
	}

	if creds.ClaudeAiOauth.AccessToken == "" {
		return nil, fmt.Errorf("no access token found in credentials")
	}

	return &creds, nil
}

// LoadFromPath reads credentials from a specific file path
func LoadFromPath(path string) (*Credentials, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) //#nosec G304 -- path is derived from user's home directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials file not found at %s - please run 'claude' first to authenticate", path)
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	return parseCredentials(data)
}
