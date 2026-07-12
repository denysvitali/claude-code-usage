package credentials

import "fmt"

// Provider IDs for all supported providers.
const (
	ProviderClaude  = "claude"
	ProviderKimi    = "kimi"
	ProviderZAi     = "zai"
	ProviderMiniMax = "minimax"
	ProviderCodex   = "codex"
	ProviderGrok    = "grok"

	// DefaultAccountName is the account name used when none is specified.
	DefaultAccountName = "default"
)

// ProviderInfo describes a supported provider.
type ProviderInfo struct {
	ID   string
	Name string
}

// Providers lists all supported providers with their display names.
var Providers = []ProviderInfo{
	{ID: ProviderClaude, Name: "Claude (Anthropic)"},
	{ID: ProviderKimi, Name: "Kimi"},
	{ID: ProviderMiniMax, Name: "MiniMax"},
	{ID: ProviderCodex, Name: "Codex (OpenAI)"},
	{ID: ProviderGrok, Name: "Grok (xAI)"},
}

// ProviderDisplayName returns the display name for a provider ID.
func ProviderDisplayName(id string) string {
	for _, p := range Providers {
		if p.ID == id {
			return p.Name
		}
	}
	return id
}

// AccountCredentials is the generic interface implemented by every
// provider's credential structure. It provides uniform account management
// so callers never need per-provider switch statements.
type AccountCredentials interface {
	ProviderConfig

	// ID returns the provider's unique identifier.
	ID() string
	// Name returns the provider's display name.
	Name() string
	// ListAccounts returns all configured account names.
	ListAccounts() []string
	// RemoveAccount deletes an account by name.
	RemoveAccount(name string) error
	// RenameAccount renames an account.
	RenameAccount(oldName, newName string) error
}

// NewCredentials returns an empty credentials structure for the provider.
func NewCredentials(providerID string) (AccountCredentials, error) {
	switch providerID {
	case ProviderClaude:
		return &ClaudeCredentials{}, nil
	case ProviderKimi:
		return &KimiCredentials{}, nil
	case ProviderZAi:
		return &ZAiCredentials{}, nil
	case ProviderMiniMax:
		return &MiniMaxCredentials{}, nil
	case ProviderCodex:
		return &CodexCredentials{}, nil
	case ProviderGrok:
		return &GrokCredentials{}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
}

// CodexCredentials represents credentials exported by the Codex CLI.
type CodexCredentials struct {
	Tokens *CodexTokens `json:"tokens,omitempty"`
}

// CodexTokens contains the access material exported by the Codex CLI.
type CodexTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
}

// Validate checks that Codex credentials include an access token.
func (c *CodexCredentials) Validate() error {
	if c.Tokens == nil || c.Tokens.AccessToken == "" {
		return fmt.Errorf("no Codex access token found")
	}
	return nil
}

// ID returns the provider's unique identifier.
func (c *CodexCredentials) ID() string { return ProviderCodex }

// Name returns the provider's display name.
func (c *CodexCredentials) Name() string { return ProviderDisplayName(ProviderCodex) }

// ListAccounts returns all account names for this provider.
func (c *CodexCredentials) ListAccounts() []string {
	if c.Tokens != nil {
		return []string{DefaultAccountName}
	}
	return nil
}

// RemoveAccount reports that Codex CLI credentials cannot be removed this way.
func (c *CodexCredentials) RemoveAccount(string) error {
	return fmt.Errorf("codex CLI credentials have one account")
}

// RenameAccount reports that Codex CLI credentials cannot be renamed.
func (c *CodexCredentials) RenameAccount(string, string) error {
	return fmt.Errorf("codex CLI credentials have one account")
}

// GrokCredentials stores consumer-plan quota snapshots. xAI does not expose
// a supported API for the weekly consumer quota, so these values are managed
// explicitly by the user or an external updater.
type GrokCredentials struct {
	Accounts map[string]*GrokAccount `json:"accounts"`
}

// GrokAccount stores consumer-plan quota data for one Grok account.
type GrokAccount struct {
	WeeklyUtilization float64 `json:"weeklyUtilization"`
	ResetsAt          string  `json:"resetsAt"`
	Plan              string  `json:"plan,omitempty"`
}

// Validate checks that Grok credentials contain at least one valid account.
func (g *GrokCredentials) Validate() error {
	if len(g.Accounts) == 0 {
		return fmt.Errorf("no Grok accounts found")
	}
	for name, account := range g.Accounts {
		if account == nil || account.ResetsAt == "" || account.WeeklyUtilization < 0 || account.WeeklyUtilization > 100 {
			return fmt.Errorf("invalid Grok account %q", name)
		}
	}
	return nil
}

// ID returns the provider's unique identifier.
func (g *GrokCredentials) ID() string { return ProviderGrok }

// Name returns the provider's display name.
func (g *GrokCredentials) Name() string { return ProviderDisplayName(ProviderGrok) }

// ListAccounts returns all account names for this provider.
func (g *GrokCredentials) ListAccounts() []string {
	names := make([]string, 0, len(g.Accounts))
	for name := range g.Accounts {
		names = append(names, name)
	}
	return names
}

// GetAccount returns the named Grok account, or the default account when name is empty.
func (g *GrokCredentials) GetAccount(name string) *GrokAccount {
	if name == "" {
		name = DefaultAccountName
	}
	return g.Accounts[name]
}

// RemoveAccount deletes a Grok account by name.
func (g *GrokCredentials) RemoveAccount(name string) error {
	return removeFromAccounts(g.Accounts, name)
}

// RenameAccount renames a Grok account.
func (g *GrokCredentials) RenameAccount(oldName, newName string) error {
	return renameInAccounts(g.Accounts, oldName, newName)
}

// LoadAccountCredentials loads a provider's credentials behind the generic
// AccountCredentials interface.
func (m *Manager) LoadAccountCredentials(providerID string) (AccountCredentials, error) {
	creds, err := NewCredentials(providerID)
	if err != nil {
		return nil, err
	}
	if err := m.LoadProvider(providerID, creds); err != nil {
		return nil, err
	}
	return creds, nil
}

// removeFromAccounts removes an account from a provider's accounts map.
func removeFromAccounts[T any](accounts map[string]*T, name string) error {
	if accounts == nil || accounts[name] == nil {
		return fmt.Errorf("account '%s' not found", name)
	}
	delete(accounts, name)
	return nil
}

// renameInAccounts renames an account within a provider's accounts map.
func renameInAccounts[T any](accounts map[string]*T, oldName, newName string) error {
	if accounts == nil || accounts[oldName] == nil {
		return fmt.Errorf("account '%s' not found", oldName)
	}
	if accounts[newName] != nil {
		return fmt.Errorf("account '%s' already exists", newName)
	}
	accounts[newName] = accounts[oldName]
	delete(accounts, oldName)
	return nil
}

// ID returns the provider's unique identifier.
func (c *ClaudeCredentials) ID() string { return ProviderClaude }

// Name returns the provider's display name.
func (c *ClaudeCredentials) Name() string { return ProviderDisplayName(ProviderClaude) }

// RemoveAccount deletes an account by name.
func (c *ClaudeCredentials) RemoveAccount(name string) error {
	return removeFromAccounts(c.Accounts, name)
}

// RenameAccount renames an account.
func (c *ClaudeCredentials) RenameAccount(oldName, newName string) error {
	return renameInAccounts(c.Accounts, oldName, newName)
}

// ID returns the provider's unique identifier.
func (k *KimiCredentials) ID() string { return ProviderKimi }

// Name returns the provider's display name.
func (k *KimiCredentials) Name() string { return ProviderDisplayName(ProviderKimi) }

// RemoveAccount deletes an account by name.
func (k *KimiCredentials) RemoveAccount(name string) error {
	return removeFromAccounts(k.Accounts, name)
}

// RenameAccount renames an account.
func (k *KimiCredentials) RenameAccount(oldName, newName string) error {
	return renameInAccounts(k.Accounts, oldName, newName)
}

// ID returns the provider's unique identifier.
func (z *ZAiCredentials) ID() string { return ProviderZAi }

// Name returns the provider's display name.
func (z *ZAiCredentials) Name() string { return ProviderDisplayName(ProviderZAi) }

// RemoveAccount deletes an account by name.
func (z *ZAiCredentials) RemoveAccount(name string) error {
	return removeFromAccounts(z.Accounts, name)
}

// RenameAccount renames an account.
func (z *ZAiCredentials) RenameAccount(oldName, newName string) error {
	return renameInAccounts(z.Accounts, oldName, newName)
}

// ID returns the provider's unique identifier.
func (m *MiniMaxCredentials) ID() string { return ProviderMiniMax }

// Name returns the provider's display name.
func (m *MiniMaxCredentials) Name() string { return ProviderDisplayName(ProviderMiniMax) }

// RemoveAccount deletes an account by name.
func (m *MiniMaxCredentials) RemoveAccount(name string) error {
	return removeFromAccounts(m.Accounts, name)
}

// RenameAccount renames an account.
func (m *MiniMaxCredentials) RenameAccount(oldName, newName string) error {
	return renameInAccounts(m.Accounts, oldName, newName)
}
