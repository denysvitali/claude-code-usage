package credentials

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOAuthCredentials_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		expected  bool
	}{
		{
			name:      "future expiry is not expired",
			expiresAt: time.Now().Add(1 * time.Hour).UnixMilli(),
			expected:  false,
		},
		{
			name:      "past expiry is expired",
			expiresAt: time.Now().Add(-1 * time.Hour).UnixMilli(),
			expected:  true,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Second).UnixMilli(),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OAuthCredentials{ExpiresAt: tt.expiresAt}
			if got := o.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOAuthCredentials_ExpiresIn(t *testing.T) {
	t.Run("future expiry returns positive duration", func(t *testing.T) {
		expiresAt := time.Now().Add(2 * time.Hour).UnixMilli()
		o := &OAuthCredentials{ExpiresAt: expiresAt}

		got := o.ExpiresIn()
		// Allow 1 second tolerance
		if got < 1*time.Hour+59*time.Minute || got > 2*time.Hour+1*time.Second {
			t.Errorf("ExpiresIn() = %v, want approximately 2h", got)
		}
	})

	t.Run("past expiry returns negative duration", func(t *testing.T) {
		expiresAt := time.Now().Add(-1 * time.Hour).UnixMilli()
		o := &OAuthCredentials{ExpiresAt: expiresAt}

		got := o.ExpiresIn()
		if got >= 0 {
			t.Errorf("ExpiresIn() = %v, want negative duration", got)
		}
	})
}

func TestLoadFromPath(t *testing.T) {
	t.Run("valid credentials file", func(t *testing.T) {
		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "credentials.json")

		content := `{
			"claudeAiOauth": {
				"accessToken": "test-token",
				"refreshToken": "test-refresh",
				"expiresAt": 1735689600000,
				"scopes": ["read", "write"]
			}
		}`
		if err := os.WriteFile(credPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		creds, err := LoadFromPath(credPath)
		if err != nil {
			t.Fatalf("LoadFromPath() error = %v", err)
		}

		if creds.ClaudeAiOauth.AccessToken != "test-token" {
			t.Errorf("AccessToken = %v, want test-token", creds.ClaudeAiOauth.AccessToken)
		}
		if creds.ClaudeAiOauth.RefreshToken != "test-refresh" {
			t.Errorf("RefreshToken = %v, want test-refresh", creds.ClaudeAiOauth.RefreshToken)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadFromPath("/nonexistent/path/credentials.json")
		if err == nil {
			t.Error("LoadFromPath() expected error for nonexistent file")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "credentials.json")

		if err := os.WriteFile(credPath, []byte("not valid json"), 0600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, err := LoadFromPath(credPath)
		if err == nil {
			t.Error("LoadFromPath() expected error for invalid JSON")
		}
	})

	t.Run("missing OAuth credentials", func(t *testing.T) {
		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "credentials.json")

		if err := os.WriteFile(credPath, []byte("{}"), 0600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, err := LoadFromPath(credPath)
		if err == nil {
			t.Error("LoadFromPath() expected error for missing OAuth credentials")
		}
	})

	t.Run("empty access token", func(t *testing.T) {
		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "credentials.json")

		content := `{
			"claudeAiOauth": {
				"accessToken": "",
				"refreshToken": "test-refresh",
				"expiresAt": 1735689600000
			}
		}`
		if err := os.WriteFile(credPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, err := LoadFromPath(credPath)
		if err == nil {
			t.Error("LoadFromPath() expected error for empty access token")
		}
	})
}
