package usage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/denysvitali/llm-usage/internal/credentials"
)

func TestGetProvidersDiscoversLocalCodexSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	data := []byte(`{"tokens":{"access_token":"test-token","account_id":"acct"}}`)
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	providers, _ := GetProviders("all", "", false, false, credentials.NewManager())
	found := false
	for _, p := range providers {
		if p.ID() == credentials.ProviderCodex {
			found = true
		}
	}
	if !found {
		t.Fatalf("unexpected providers: %#v", providers)
	}
}
