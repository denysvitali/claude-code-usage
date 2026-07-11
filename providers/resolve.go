package providers

import (
	"fmt"
	"slices"
	"strings"

	"github.com/denysvitali/llm-usage/internal/credentials"
	base "github.com/denysvitali/llm-usage/provider"
	"github.com/denysvitali/llm-usage/providers/claude"
	"github.com/denysvitali/llm-usage/providers/codex"
	"github.com/denysvitali/llm-usage/providers/kimi"
	"github.com/denysvitali/llm-usage/providers/minimax"
	"github.com/denysvitali/llm-usage/providers/zai"
)

// Instance is a provider ready for usage retrieval.
type Instance struct {
	base.Provider
	AccountName string
}

// Request controls provider and account selection.
type Request struct {
	Provider, Account            string
	AllAccounts, Debug, Explicit bool
}

// Definition is a compiled provider integration. New providers add one
// definition with metadata and a loader; callers do not need a new switch case.
type Definition struct {
	Capability
	Load func(Request, *credentials.Manager) ([]Instance, []base.Usage)
}

var definitions = []Definition{
	{Capability: Capability{ID: credentials.ProviderClaude, Name: "Claude (Anthropic)", Auth: "OAuth", Implemented: true}, Load: loadClaude},
	{Capability: Capability{ID: credentials.ProviderKimi, Name: "Kimi", Auth: "API key", Implemented: true}, Load: loadKimi},
	{Capability: Capability{ID: credentials.ProviderMiniMax, Name: "MiniMax", Auth: "Cookie + group ID", Implemented: true}, Load: loadMiniMax},
	{Capability: Capability{ID: credentials.ProviderZAi, Name: "Z.AI", Auth: "API key", Implemented: false}, Load: loadZAi},
	{Capability: Capability{ID: credentials.ProviderCodex, Name: "Codex (OpenAI)", Auth: "Codex CLI OAuth", Implemented: true}, Load: loadCodex},
	{Capability: Capability{ID: credentials.ProviderGrok, Name: "Grok (xAI)", Auth: "Quota snapshot", Implemented: true}, Load: loadGrok},
}

func Resolve(request Request, manager *credentials.Manager) ([]Instance, []base.Usage) {
	ids := selectedIDs(request, manager)
	var instances []Instance
	var failures []base.Usage
	for _, id := range ids {
		definition, ok := definitionFor(strings.TrimSpace(id))
		if !ok {
			failures = append(failures, Failure(id, "", fmt.Errorf("unknown provider %q", id)))
			continue
		}
		if !definition.Implemented {
			failures = append(failures, Failure(id, "", fmt.Errorf("provider is not implemented")))
			continue
		}
		loaded, failed := definition.Load(request, manager)
		instances, failures = append(instances, loaded...), append(failures, failed...)
	}
	return instances, failures
}

func selectedIDs(request Request, manager *credentials.Manager) []string {
	if request.Explicit {
		return strings.Split(request.Provider, ",")
	}
	ids := manager.ListAvailable()
	if _, err := credentials.LoadCodexCLI(); err == nil && !slices.Contains(ids, credentials.ProviderCodex) {
		ids = append(ids, credentials.ProviderCodex)
	}
	if _, err := credentials.LoadGrokCLI(); err == nil && !slices.Contains(ids, credentials.ProviderGrok) {
		ids = append(ids, credentials.ProviderGrok)
	}
	if len(ids) == 0 {
		return []string{credentials.ProviderClaude}
	}
	return ids
}

func definitionFor(id string) (Definition, bool) {
	for _, d := range definitions {
		if d.ID == id {
			return d, true
		}
	}
	return Definition{}, false
}

func Name(id string) string {
	if d, ok := definitionFor(id); ok {
		return d.Name
	}
	return strings.ToUpper(id)
}
func ShortName(id string) string {
	if d, ok := definitionFor(id); ok {
		switch id {
		case credentials.ProviderClaude:
			return "Cl"
		case credentials.ProviderCodex:
			return "Cdx"
		case credentials.ProviderMiniMax:
			return "M"
		case credentials.ProviderKimi:
			return "K"
		case credentials.ProviderGrok:
			return "G"
		default:
			return d.ID[:1]
		}
	}
	return strings.ToUpper(id)[:1]
}
func Failure(id, account string, err error) base.Usage {
	u := base.NewUsageError(id, credentials.ProviderDisplayName(id), err)
	if account != "" {
		u.Extra = map[string]any{"account": account}
	}
	return *u
}

func loadCodex(request Request, _ *credentials.Manager) ([]Instance, []base.Usage) {
	creds, err := credentials.LoadCodexCLI()
	if err != nil && !request.Explicit {
		return nil, nil
	}
	if err != nil {
		return nil, []base.Usage{Failure(credentials.ProviderCodex, request.Account, err)}
	}
	if request.Account != "" && request.Account != credentials.DefaultAccountName {
		return nil, []base.Usage{Failure(credentials.ProviderCodex, request.Account, fmt.Errorf("account %q not found", request.Account))}
	}
	return []Instance{{Provider: codex.NewProvider(creds.Tokens.AccessToken, creds.Tokens.AccountID), AccountName: credentials.DefaultAccountName}}, nil
}

func loadGrok(request Request, _ *credentials.Manager) ([]Instance, []base.Usage) {
	creds, err := credentials.LoadGrokCLI()
	if err != nil {
		if request.Explicit {
			return nil, []base.Usage{Failure(credentials.ProviderGrok, request.Account, err)}
		}
		return nil, nil
	}
	if request.Account != "" && request.Account != credentials.DefaultAccountName {
		return nil, []base.Usage{Failure(credentials.ProviderGrok, request.Account, fmt.Errorf("account %q not found", request.Account))}
	}
	return []Instance{{Provider: newGrokAdapter(creds.AccessToken), AccountName: credentials.DefaultAccountName}}, nil
}

func loadClaude(request Request, manager *credentials.Manager) ([]Instance, []base.Usage) {
	cliCreds, cliAccount, cliErr := loadClaudeCLI()
	stored, storedErr := manager.LoadClaude()
	if cliErr != nil && storedErr != nil {
		if request.Explicit {
			return nil, []base.Usage{Failure(credentials.ProviderClaude, "", fmt.Errorf("no credentials found: %w", storedErr))}
		}
		return nil, nil
	}
	if request.Account != "" {
		if storedErr != nil {
			return nil, []base.Usage{Failure(credentials.ProviderClaude, request.Account, storedErr)}
		}
		oauth := stored.GetAccount(request.Account)
		if oauth == nil {
			return nil, []base.Usage{Failure(credentials.ProviderClaude, request.Account, fmt.Errorf("account %q not found", request.Account))}
		}
		oauth, err := refreshClaude(oauth, request.Account, stored, manager)
		if err != nil {
			return nil, []base.Usage{Failure(credentials.ProviderClaude, request.Account, err)}
		}
		return []Instance{{Provider: claude.NewProvider(oauth.AccessToken, request.Debug), AccountName: request.Account}}, nil
	}
	var instances []Instance
	var failures []base.Usage
	cliAdded := false
	if cliErr == nil {
		if claude.IsExpired(cliCreds.ExpiresAt) {
			failures = append(failures, Failure(credentials.ProviderClaude, cliAccount, fmt.Errorf("token from the Claude CLI expired - run 'claude' to re-authenticate")))
		} else {
			instances = append(instances, Instance{Provider: claude.NewProvider(cliCreds.AccessToken, request.Debug), AccountName: cliAccount})
			cliAdded = true
		}
	}
	if storedErr == nil {
		for _, account := range stored.ListAccounts() {
			if cliAdded && account == credentials.DefaultAccountName {
				continue
			}
			oauth := stored.GetAccount(account)
			if oauth == nil {
				continue
			}
			oauth, err := refreshClaude(oauth, account, stored, manager)
			if err != nil {
				failures = append(failures, Failure(credentials.ProviderClaude, account, err))
				continue
			}
			instances = append(instances, Instance{Provider: claude.NewProvider(oauth.AccessToken, request.Debug), AccountName: account})
		}
	}
	return instances, failures
}

func loadClaudeCLI() (*credentials.OAuthCredentials, string, error) {
	creds, err := credentials.Load()
	if err != nil {
		return nil, "", err
	}
	return creds.ClaudeAiOauth, credentials.DefaultAccountName, nil
}
func refreshClaude(oauth *credentials.OAuthCredentials, account string, stored *credentials.ClaudeCredentials, manager *credentials.Manager) (*credentials.OAuthCredentials, error) {
	if !claude.IsExpired(oauth.ExpiresAt) {
		return oauth, nil
	}
	if oauth.RefreshToken == "" {
		return nil, fmt.Errorf("access token expired and no refresh token available")
	}
	refreshed, err := claude.RefreshAccessToken(oauth.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("access token expired and refresh failed: %w", err)
	}
	oauth.AccessToken, oauth.ExpiresAt = refreshed.AccessToken, refreshed.ExpiresAt
	if refreshed.RefreshToken != "" {
		oauth.RefreshToken = refreshed.RefreshToken
	}
	if !manager.UsesCombinedFile() {
		if stored.Accounts != nil {
			if saved := stored.Accounts[account]; saved != nil {
				saved.AccessToken, saved.RefreshToken, saved.ExpiresAt = oauth.AccessToken, oauth.RefreshToken, oauth.ExpiresAt
			}
		} else {
			stored.ClaudeAiOauth = oauth
		}
		_ = manager.SaveProvider(credentials.ProviderClaude, stored)
	}
	return oauth, nil
}

func loadKimi(request Request, manager *credentials.Manager) ([]Instance, []base.Usage) {
	return loadAccounts(credentials.ProviderKimi, request, manager.LoadKimi, func(c *credentials.KimiCredentials) []string { return c.ListAccounts() }, func(c *credentials.KimiCredentials, name string) base.Provider {
		if a := c.GetAccount(name); a != nil {
			return kimi.NewProvider(a.APIKey)
		}
		return nil
	})
}
func loadMiniMax(request Request, manager *credentials.Manager) ([]Instance, []base.Usage) {
	return loadAccounts(credentials.ProviderMiniMax, request, manager.LoadMiniMax, func(c *credentials.MiniMaxCredentials) []string { return c.ListAccounts() }, func(c *credentials.MiniMaxCredentials, name string) base.Provider {
		if a := c.GetAccount(name); a != nil {
			return minimax.NewProvider(a.Cookie, a.GroupID)
		}
		return nil
	})
}
func loadZAi(request Request, manager *credentials.Manager) ([]Instance, []base.Usage) {
	return loadAccounts(credentials.ProviderZAi, request, manager.LoadZAi, func(c *credentials.ZAiCredentials) []string { return c.ListAccounts() }, func(c *credentials.ZAiCredentials, name string) base.Provider {
		if a := c.GetAccount(name); a != nil {
			return zai.NewProvider(a.APIKey)
		}
		return nil
	})
}

func loadAccounts[C any](id string, request Request, load func() (C, error), list func(C) []string, build func(C, string) base.Provider) ([]Instance, []base.Usage) {
	creds, err := load()
	if err != nil {
		return nil, []base.Usage{Failure(id, "", err)}
	}
	if request.AllAccounts || request.Account == "" {
		var instances []Instance
		for _, account := range list(creds) {
			if p := build(creds, account); p != nil {
				instances = append(instances, Instance{Provider: p, AccountName: account})
			}
		}
		return instances, nil
	}
	p := build(creds, request.Account)
	if p == nil {
		return nil, []base.Usage{Failure(id, request.Account, fmt.Errorf("account %q not found", request.Account))}
	}
	return []Instance{{Provider: p, AccountName: request.Account}}, nil
}
