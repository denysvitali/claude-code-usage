package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const DefaultFilename = "config.yaml"

const CurrentVersion = "1"

func DefaultPath() string { return filepath.Join(xdg.ConfigHome, "llm-usage", DefaultFilename) }

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
