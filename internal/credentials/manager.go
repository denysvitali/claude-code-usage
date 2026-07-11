package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// ProviderConfig is the interface for provider-specific credential configs
type ProviderConfig interface {
	// Validate checks if the credentials are valid
	Validate() error
}

// Manager handles loading credentials for multiple providers
type Manager struct {
	configDir       string // $XDG_CONFIG_HOME/llm-usage (defaults to ~/.config/llm-usage)
	credentialsFile string // optional path to a combined credentials file
}

// NewManager creates a new credential manager
func NewManager() *Manager {
	return &Manager{
		configDir: filepath.Join(xdg.ConfigHome, "llm-usage"),
	}
}

// NewManagerFromFile creates a credential manager that loads all credentials
// from a single combined JSON file. Values in the file may reference environment
// variables using $VAR or ${VAR} syntax, which are expanded at load time.
//
// The file format is a JSON object mapping provider IDs to their credential
// structures, for example:
//
//	{
//	  "claude": { "accounts": { "default": { "accessToken": "$CLAUDE_TOKEN", ... } } },
//	  "kimi":   { "accounts": { "default": { "apiKey": "$KIMI_API_KEY" } } }
//	}
func NewManagerFromFile(path string) *Manager {
	return &Manager{
		configDir:       filepath.Join(xdg.ConfigHome, "llm-usage"),
		credentialsFile: path,
	}
}

// UsesCombinedFile reports whether credentials come from a combined file
// (via --credentials-file) rather than the per-provider config directory.
func (m *Manager) UsesCombinedFile() bool {
	return m.credentialsFile != ""
}

// ConfigDir returns the configuration directory path
func (m *Manager) ConfigDir() string {
	return m.configDir
}

// EnsureConfigDir creates the config directory if it doesn't exist
func (m *Manager) EnsureConfigDir() error {
	return os.MkdirAll(m.configDir, 0700)
}

// LoadProvider loads credentials for a specific provider
func (m *Manager) LoadProvider(providerID string, config ProviderConfig) error {
	if m.credentialsFile != "" {
		return m.loadProviderFromFile(providerID, config)
	}

	configPath := m.providerPath(providerID)

	data, err := os.ReadFile(configPath) //nolint:gosec
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("credentials file not found at %s", configPath)
		}
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse credentials file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	return nil
}

// loadProviderFromFile loads a single provider's credentials from the combined
// credentials file, expanding environment variable references before parsing.
func (m *Manager) loadProviderFromFile(providerID string, config ProviderConfig) error {
	all, err := m.readCombinedFile()
	if err != nil {
		return err
	}

	raw, ok := all[providerID]
	if !ok {
		return fmt.Errorf("provider %q not found in credentials file %s", providerID, m.credentialsFile)
	}

	if err := json.Unmarshal(raw, config); err != nil {
		return fmt.Errorf("failed to parse credentials for provider %q: %w", providerID, err)
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid credentials for provider %q: %w", providerID, err)
	}

	return nil
}

// readCombinedFile reads, env-expands, and parses the combined credentials file.
func (m *Manager) readCombinedFile() (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(m.credentialsFile) //nolint:gosec
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials file not found at %s", m.credentialsFile)
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	var all map[string]json.RawMessage
	if err := json.Unmarshal([]byte(expanded), &all); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file %s: %w", m.credentialsFile, err)
	}

	return all, nil
}

// providerPath returns the path to a provider's credential file
func (m *Manager) providerPath(providerID string) string {
	return filepath.Join(m.configDir, providerID+".json")
}

// ProviderExists checks if a provider's credential file exists
func (m *Manager) ProviderExists(providerID string) bool {
	_, err := os.Stat(m.providerPath(providerID))
	return err == nil
}

// ListAvailable returns a list of providers that have credential files
func (m *Manager) ListAvailable() []string {
	if m.credentialsFile != "" {
		all, err := m.readCombinedFile()
		if err != nil {
			return nil
		}
		providers := make([]string, 0, len(all))
		for id := range all {
			providers = append(providers, id)
		}
		return providers
	}

	entries, err := os.ReadDir(m.configDir)
	if err != nil {
		return nil
	}

	var providers []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check if it's a JSON file
		if filepath.Ext(name) == ".json" {
			// Remove .json extension to get provider ID
			providerID := name[:len(name)-5]
			providers = append(providers, providerID)
		}
	}
	return providers
}

// LoadClaude loads Claude credentials from the config file
func (m *Manager) LoadClaude() (*ClaudeCredentials, error) {
	var creds ClaudeCredentials
	if err := m.LoadProvider("claude", &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// LoadKimi loads Kimi credentials from the config file
func (m *Manager) LoadKimi() (*KimiCredentials, error) {
	var creds KimiCredentials
	if err := m.LoadProvider("kimi", &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// LoadZAi loads Z.AI credentials from the config file
func (m *Manager) LoadZAi() (*ZAiCredentials, error) {
	var creds ZAiCredentials
	if err := m.LoadProvider("zai", &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// LoadMiniMax loads MiniMax credentials from the config file
func (m *Manager) LoadMiniMax() (*MiniMaxCredentials, error) {
	var creds MiniMaxCredentials
	if err := m.LoadProvider("minimax", &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (m *Manager) LoadCodex() (*CodexCredentials, error) {
	var creds CodexCredentials
	if err := m.LoadProvider(ProviderCodex, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (m *Manager) LoadGrok() (*GrokCredentials, error) {
	var creds GrokCredentials
	if err := m.LoadProvider(ProviderGrok, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// ClaudeCredentials represents Claude OAuth credentials with multi-account support
type ClaudeCredentials struct {
	ClaudeAiOauth *OAuthCredentials         `json:"claudeAiOauth,omitempty"` // Legacy single-account format
	Accounts      map[string]*ClaudeAccount `json:"accounts,omitempty"`      // Multi-account format
}

// ClaudeAccount represents a single Claude account's credentials
type ClaudeAccount struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	ExpiresAt    int64    `json:"expiresAt"`
	Scopes       []string `json:"scopes"`
}

// ToOAuthCredentials converts a ClaudeAccount to OAuthCredentials
func (a *ClaudeAccount) ToOAuthCredentials() *OAuthCredentials {
	if a == nil {
		return nil
	}
	return &OAuthCredentials{
		AccessToken:  a.AccessToken,
		RefreshToken: a.RefreshToken,
		ExpiresAt:    a.ExpiresAt,
		Scopes:       a.Scopes,
	}
}

// GetAccount returns the specified account's credentials, or the default/first available account
func (c *ClaudeCredentials) GetAccount(accountName string) *OAuthCredentials {
	// Try multi-account format first
	if c.Accounts != nil {
		if accountName == "" {
			// Try "default" first, then fall back to first account
			if acc, ok := c.Accounts["default"]; ok {
				return acc.ToOAuthCredentials()
			}
			for _, acc := range c.Accounts {
				return acc.ToOAuthCredentials()
			}
		} else {
			if acc, ok := c.Accounts[accountName]; ok {
				return acc.ToOAuthCredentials()
			}
		}
	}

	// Fall back to legacy format
	return c.ClaudeAiOauth
}

// ListAccounts returns all account names for this provider
func (c *ClaudeCredentials) ListAccounts() []string {
	if c.Accounts != nil {
		names := make([]string, 0, len(c.Accounts))
		for name := range c.Accounts {
			names = append(names, name)
		}
		return names
	}
	if c.ClaudeAiOauth != nil {
		return []string{"default"}
	}
	return nil
}

// Validate checks if the Claude credentials are valid
func (c *ClaudeCredentials) Validate() error {
	if len(c.Accounts) > 0 {
		for name, acc := range c.Accounts {
			if acc.AccessToken == "" {
				return fmt.Errorf("no access token found for account %q", name)
			}
		}
		return nil
	}
	if c.ClaudeAiOauth == nil {
		return fmt.Errorf("no OAuth credentials found")
	}
	if c.ClaudeAiOauth.AccessToken == "" {
		return fmt.Errorf("no access token found")
	}
	return nil
}

// KimiCredentials represents Kimi API credentials with multi-account support
type KimiCredentials struct {
	APIKey   string                  `json:"apiKey,omitempty"`   // Legacy single-account format
	Accounts map[string]*KimiAccount `json:"accounts,omitempty"` // Multi-account format
}

// KimiAccount represents a single Kimi account's credentials
type KimiAccount struct {
	APIKey string `json:"apiKey"`
}

// GetAccount returns the specified account's credentials, or the default/first available account
func (k *KimiCredentials) GetAccount(accountName string) *KimiAccount {
	// Try multi-account format first
	if k.Accounts != nil {
		if accountName == "" {
			// Try "default" first, then fall back to first account
			if acc, ok := k.Accounts["default"]; ok {
				return acc
			}
			for _, acc := range k.Accounts {
				return acc
			}
		} else {
			if acc, ok := k.Accounts[accountName]; ok {
				return acc
			}
		}
	}

	// Fall back to legacy format
	if k.APIKey != "" {
		return &KimiAccount{APIKey: k.APIKey}
	}
	return nil
}

// ListAccounts returns all account names for this provider
func (k *KimiCredentials) ListAccounts() []string {
	if k.Accounts != nil {
		names := make([]string, 0, len(k.Accounts))
		for name := range k.Accounts {
			names = append(names, name)
		}
		return names
	}
	if k.APIKey != "" {
		return []string{"default"}
	}
	return nil
}

// Validate checks if the Kimi credentials are valid
func (k *KimiCredentials) Validate() error {
	if len(k.Accounts) > 0 {
		for name, acc := range k.Accounts {
			if acc.APIKey == "" {
				return fmt.Errorf("no API key found for account %q", name)
			}
		}
		return nil
	}
	if k.APIKey == "" {
		return fmt.Errorf("no API key found")
	}
	return nil
}

// ZAiCredentials represents Z.AI API credentials with multi-account support
type ZAiCredentials struct {
	APIKey   string                 `json:"apiKey,omitempty"`   // Legacy single-account format
	Accounts map[string]*ZAiAccount `json:"accounts,omitempty"` // Multi-account format
}

// ZAiAccount represents a single Z.AI account's credentials
type ZAiAccount struct {
	APIKey string `json:"apiKey"`
}

// GetAccount returns the specified account's credentials, or the default/first available account
func (z *ZAiCredentials) GetAccount(accountName string) *ZAiAccount {
	// Try multi-account format first
	if z.Accounts != nil {
		if accountName == "" {
			// Try "default" first, then fall back to first account
			if acc, ok := z.Accounts["default"]; ok {
				return acc
			}
			for _, acc := range z.Accounts {
				return acc
			}
		} else {
			if acc, ok := z.Accounts[accountName]; ok {
				return acc
			}
		}
	}

	// Fall back to legacy format
	if z.APIKey != "" {
		return &ZAiAccount{APIKey: z.APIKey}
	}
	return nil
}

// ListAccounts returns all account names for this provider
func (z *ZAiCredentials) ListAccounts() []string {
	if z.Accounts != nil {
		names := make([]string, 0, len(z.Accounts))
		for name := range z.Accounts {
			names = append(names, name)
		}
		return names
	}
	if z.APIKey != "" {
		return []string{"default"}
	}
	return nil
}

// Validate checks if the Z.AI credentials are valid
func (z *ZAiCredentials) Validate() error {
	if len(z.Accounts) > 0 {
		for name, acc := range z.Accounts {
			if acc.APIKey == "" {
				return fmt.Errorf("no API key found for account %q", name)
			}
		}
		return nil
	}
	if z.APIKey == "" {
		return fmt.Errorf("no API key found")
	}
	return nil
}

// MiniMaxCredentials represents MiniMax credentials with multi-account support
type MiniMaxCredentials struct {
	Cookie   string                     `json:"cookie,omitempty"`
	GroupID  string                     `json:"groupId,omitempty"`
	Accounts map[string]*MiniMaxAccount `json:"accounts,omitempty"`
}

// MiniMaxAccount represents a single MiniMax account's credentials
type MiniMaxAccount struct {
	Cookie  string `json:"cookie"`
	GroupID string `json:"groupId"`
}

// GetAccount returns the specified account's credentials, or the default/first available account
func (m *MiniMaxCredentials) GetAccount(accountName string) *MiniMaxAccount {
	// Try multi-account format first
	if m.Accounts != nil {
		if accountName == "" {
			// Try "default" first, then fall back to first account
			if acc, ok := m.Accounts["default"]; ok {
				return acc
			}
			for _, acc := range m.Accounts {
				return acc
			}
		} else {
			if acc, ok := m.Accounts[accountName]; ok {
				return acc
			}
		}
	}

	// Fall back to legacy format
	if m.Cookie != "" {
		return &MiniMaxAccount{Cookie: m.Cookie, GroupID: m.GroupID}
	}
	return nil
}

// ListAccounts returns all account names for this provider
func (m *MiniMaxCredentials) ListAccounts() []string {
	if m.Accounts != nil {
		names := make([]string, 0, len(m.Accounts))
		for name := range m.Accounts {
			names = append(names, name)
		}
		return names
	}
	if m.Cookie != "" {
		return []string{"default"}
	}
	return nil
}

// Validate checks if the MiniMax credentials are valid
func (m *MiniMaxCredentials) Validate() error {
	if len(m.Accounts) > 0 {
		for name, acc := range m.Accounts {
			if acc.Cookie == "" {
				return fmt.Errorf("no cookie found for account %q", name)
			}
			if acc.GroupID == "" {
				return fmt.Errorf("no group ID found for account %q", name)
			}
		}
		return nil
	}
	if m.Cookie == "" {
		return fmt.Errorf("no cookie found")
	}
	if m.GroupID == "" {
		return fmt.Errorf("no group ID found")
	}
	return nil
}

// SaveProvider saves provider credentials to the config file
func (m *Manager) SaveProvider(providerID string, data any) error {
	if err := m.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := m.providerPath(providerID)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(configPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// DeleteProvider deletes a provider's credential file
func (m *Manager) DeleteProvider(providerID string) error {
	configPath := m.providerPath(providerID)
	if err := os.Remove(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no credentials found for provider %q", providerID)
		}
		return fmt.Errorf("failed to delete credentials file: %w", err)
	}
	return nil
}

// ListAccounts returns all account names for a provider
func (m *Manager) ListAccounts(providerID string) ([]string, error) {
	creds, err := m.LoadAccountCredentials(providerID)
	if err != nil {
		return nil, err
	}
	return creds.ListAccounts(), nil
}

// MigrateFromClaudeCLI copies credentials from the Claude CLI to the new format.
// It reads from the CLI's storage (macOS Keychain first, then
// ~/.claude/.credentials.json) via Load.
func (m *Manager) MigrateFromClaudeCLI() error {
	newPath := m.providerPath(ProviderClaude)

	// Check if new file already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("credentials already exist at %s - remove that file first to re-migrate", newPath)
	}

	// Load from the Claude CLI's storage (keychain on macOS, file elsewhere)
	creds, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load Claude CLI credentials: %w", err)
	}

	// Ensure config directory exists
	if err := m.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(newPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write new credentials: %w", err)
	}

	return nil
}
