package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestV1RoundTrip(t *testing.T) {
	cfg := Config{
		Version: "1",
		Defaults: Defaults{
			Output: OutputDefaults{
				Format: "pretty",
				Color:  "on",
			},
			Server: ServerDefaults{
				Listen: "127.0.0.1:9000",
			},
		},
		Providers: []ProviderConfig{
			{
				ID:   "claude",
				Name: "Claude",
				Accounts: []Account{
					{
						Name: "personal",
						Auth: map[string]SecretRef{
							"access_token": "oauth-token",
						},
					},
				},
			},
			{
				ID:   "minimax",
				Name: "MiniMax",
				Accounts: []Account{
					{
						Name: "work",
						Auth: map[string]SecretRef{
							"cookie":   "session=abc",
							"group_id": "12345",
						},
					},
				},
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Sanity check: secrets should appear as plain strings in this phase.
	if !strings.Contains(string(data), "oauth-token") {
		t.Errorf("marshaled YAML missing secret value")
	}

	var parsed Config
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Version != cfg.Version {
		t.Errorf("Version = %q, want %q", parsed.Version, cfg.Version)
	}
	if parsed.Defaults.Output.Format != cfg.Defaults.Output.Format {
		t.Errorf("Output.Format = %q, want %q", parsed.Defaults.Output.Format, cfg.Defaults.Output.Format)
	}
	if parsed.Providers[1].Accounts[0].Auth["group_id"] != "12345" {
		t.Errorf("group_id = %q, want %q", parsed.Providers[1].Accounts[0].Auth["group_id"], "12345")
	}
}

func TestSecretRefYAML(t *testing.T) {
	type wrapper struct {
		Value SecretRef `yaml:"value"`
	}

	var w wrapper
	if err := yaml.Unmarshal([]byte("value: my-secret"), &w); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if w.Value != "my-secret" {
		t.Errorf("Value = %q, want %q", w.Value, "my-secret")
	}

	out, err := yaml.Marshal(wrapper{Value: "out-secret"})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !strings.Contains(string(out), "out-secret") {
		t.Errorf("marshaled output missing secret: %s", out)
	}
}
