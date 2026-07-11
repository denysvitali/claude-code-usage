package config

import "gopkg.in/yaml.v3"

// SecretRef is a local placeholder for a secret reference.
// It will eventually be replaced/aliased to the internal/secrets package type.
type SecretRef string

// Config is the top-level versioned configuration for llm-usage.
type Config struct {
	Version   string           `yaml:"version"`
	Defaults  Defaults         `yaml:"defaults,omitempty"`
	Providers []ProviderConfig `yaml:"providers"`
}

// Defaults holds global default settings.
type Defaults struct {
	Output OutputDefaults `yaml:"output,omitempty"`
	Server ServerDefaults `yaml:"server,omitempty"`
	Cache  CacheDefaults  `yaml:"cache,omitempty"`
}

type CacheDefaults struct {
	TTL          string `yaml:"ttl,omitempty"`
	StaleIfError bool   `yaml:"stale_if_error,omitempty"`
}

// OutputDefaults holds default output settings.
type OutputDefaults struct {
	Format string `yaml:"format,omitempty"`
	Color  string `yaml:"color,omitempty"`
}

// ServerDefaults holds default server settings.
type ServerDefaults struct {
	Listen string `yaml:"listen,omitempty"`
}

// ProviderConfig describes a single provider and its accounts.
type ProviderConfig struct {
	ID       string    `yaml:"id"`
	Name     string    `yaml:"name,omitempty"`
	Accounts []Account `yaml:"accounts"`
}

// Account describes a single provider account.
type Account struct {
	Name string               `yaml:"name"`
	Auth map[string]SecretRef `yaml:"auth,omitempty"`
}

// MarshalYAML serializes a SecretRef as a plain scalar string.
func (s SecretRef) MarshalYAML() (interface{}, error) {
	return string(s), nil
}

// UnmarshalYAML parses a SecretRef from a YAML scalar.
func (s *SecretRef) UnmarshalYAML(n *yaml.Node) error {
	if n == nil {
		*s = ""
		return nil
	}
	*s = SecretRef(n.Value)
	return nil
}
