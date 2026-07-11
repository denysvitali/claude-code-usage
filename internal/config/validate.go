package config

import (
	"fmt"
	"strings"

	"github.com/denysvitali/llm-usage/providers"
)

var validOutputFormats = map[string]struct{}{
	"pretty": {},
	"json":   {},
	"waybar": {},
}

var validColorValues = map[string]struct{}{
	"auto": {},
	"on":   {},
	"off":  {},
}

// Validate checks that cfg is a valid version-1 configuration.
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if cfg.Version != CurrentVersion {
		return fmt.Errorf("unsupported config version %q (expected %q)", cfg.Version, CurrentVersion)
	}

	if len(cfg.Providers) == 0 {
		return fmt.Errorf("at least one provider is required")
	}

	seenProviders := make(map[string]struct{}, len(cfg.Providers))
	for _, p := range cfg.Providers {
		if p.ID == "" {
			return fmt.Errorf("provider ID is required")
		}

		if _, ok := seenProviders[p.ID]; ok {
			return fmt.Errorf("duplicate provider ID %q", p.ID)
		}
		seenProviders[p.ID] = struct{}{}

		capability, err := providers.Lookup(p.ID)
		if err != nil {
			return err
		}
		if !capability.Implemented {
			return fmt.Errorf("provider %q is registered but not implemented", p.ID)
		}

		if len(p.Accounts) == 0 {
			return fmt.Errorf("provider %q requires at least one account", p.ID)
		}

		seenAccounts := make(map[string]struct{}, len(p.Accounts))
		for _, a := range p.Accounts {
			if a.Name == "" {
				return fmt.Errorf("account name is required for provider %q", p.ID)
			}

			if _, ok := seenAccounts[a.Name]; ok {
				return fmt.Errorf("duplicate account name %q for provider %q", a.Name, p.ID)
			}
			seenAccounts[a.Name] = struct{}{}
		}
	}

	if cfg.Defaults.Output.Format != "" {
		if _, ok := validOutputFormats[strings.ToLower(cfg.Defaults.Output.Format)]; !ok {
			return fmt.Errorf("invalid output format %q", cfg.Defaults.Output.Format)
		}
	}

	if cfg.Defaults.Output.Color != "" {
		if _, ok := validColorValues[strings.ToLower(cfg.Defaults.Output.Color)]; !ok {
			return fmt.Errorf("invalid output color %q", cfg.Defaults.Output.Color)
		}
	}

	return nil
}
