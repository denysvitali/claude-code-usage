package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// DefaultFilename is the default configuration file name.
const DefaultFilename = "config.yaml"

// CurrentVersion is the supported configuration file version.
const CurrentVersion = "1"

// DefaultPath returns the default configuration file path inside the XDG
// configuration directory.
func DefaultPath() string { return filepath.Join(xdg.ConfigHome, "llm-usage", DefaultFilename) }

// Load reads and validates the configuration file at path into cfg.
func Load(path string, cfg *Config) error {
	data, err := os.ReadFile(path) //nolint:gosec // path is user-selected
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if err := Validate(cfg); err != nil {
		return err
	}
	return nil
}

// LoadOptional reads the configuration file at path when it exists, otherwise
// returning a zero-valued Config.
func LoadOptional(path string) (*Config, error) {
	cfg := new(Config)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("stat config: %w", err)
	}
	if err := Load(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes cfg to path after validation, creating parent directories as needed.
func Save(path string, cfg Config) error {
	if err := Validate(&cfg); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	return atomicWriteFile(path, data, 0600)
}
