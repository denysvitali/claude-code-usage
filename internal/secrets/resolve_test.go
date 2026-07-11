package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvironmentAndFile(t *testing.T) {
	t.Setenv("LLM_USAGE_TEST_SECRET", "value")
	resolver := Resolver{}
	if got, err := resolver.Resolve("env:LLM_USAGE_TEST_SECRET"); err != nil || got != "value" {
		t.Fatalf("env resolve = %q, %v", got, err)
	}
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte(" file-value \n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := resolver.Resolve("file:" + path); err != nil || got != "file-value" {
		t.Fatalf("file resolve = %q, %v", got, err)
	}
}
