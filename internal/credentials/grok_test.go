package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGrokCLI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.Mkdir(filepath.Join(home, ".grok"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".grok", "auth.json"), []byte(`{"session":{"key":"redacted"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	creds, err := LoadGrokCLI()
	if err != nil {
		t.Fatal(err)
	}
	if creds.AccessToken != "redacted" {
		t.Fatal("unexpected token load")
	}
}
