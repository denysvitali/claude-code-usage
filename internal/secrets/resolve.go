// Package secrets resolves references without exposing a provider-specific secret model.
package secrets

import (
	"fmt"
	"os"
	"strings"
)

// Resolver resolves values using explicit reference prefixes.
type Resolver struct{}

// Resolve supports env:NAME, file:/path, and literal values. Literal values
// are retained for compatibility with the version-one config format.
func (Resolver) Resolve(ref string) (string, error) {
	switch {
	case strings.HasPrefix(ref, "env:"):
		name := strings.TrimPrefix(ref, "env:")
		if name == "" {
			return "", fmt.Errorf("empty environment variable reference")
		}
		value, ok := os.LookupEnv(name)
		if !ok {
			return "", fmt.Errorf("environment variable %q is not set", name)
		}
		return value, nil
	case strings.HasPrefix(ref, "file:"):
		path := strings.TrimPrefix(ref, "file:")
		if path == "" {
			return "", fmt.Errorf("empty secret file reference")
		}
		value, err := os.ReadFile(path) //nolint:gosec // explicit user config
		if err != nil {
			return "", fmt.Errorf("read secret file: %w", err)
		}
		return strings.TrimSpace(string(value)), nil
	default:
		return ref, nil
	}
}
