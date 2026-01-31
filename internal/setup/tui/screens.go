// Package tui provides the Bubble Tea TUI for the setup wizard.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/denysvitali/llm-usage/internal/credentials"
)

// updateProviderSelect handles updates for the provider selection screen
func (m Model) updateProviderSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		if m.selectedIdx < len(AllProviders)-1 {
			m.selectedIdx++
		}
	case "enter":
		provider := AllProviders[m.selectedIdx]
		// Claude requires special handling (OAuth)
		if provider.ID == "claude" {
			m.selectedProvider = provider.ID
			// For Claude, we show a special screen explaining OAuth
			m.errorMsg = "Claude uses OAuth. Please run: llm-usage setup add claude"
			return m, nil
		}
		m.selectedProvider = provider.ID
		return m.pushScreen(screenAddAccountName), nil
	}
	return m, nil
}

// viewProviderSelect renders the provider selection screen
func (m Model) viewProviderSelect() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select Provider"))
	b.WriteString("\n\n")

	for i, provider := range AllProviders {
		cursor := " "
		if i == m.selectedIdx {
			cursor = cursorStyle.Render("▶")
			b.WriteString(cursor + " " + selectedStyle.Render(provider.Name) + "\n")
		} else {
			b.WriteString(cursor + " " + normalStyle.Render(provider.Name) + "\n")
		}
	}

	if m.errorMsg != "" {
		b.WriteString("\n" + RenderError(m.errorMsg))
	}

	return b.String()
}

// updateAddAccountName handles updates for the account name input screen
func (m Model) updateAddAccountName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Use default name if empty
		accountName := m.inputText
		if accountName == "" {
			accountName = "default"
		}
		// Check if account already exists
		if err := m.checkAccountExists(accountName); err != nil {
			m.errorMsg = err.Error()
			return m, nil
		}
		// Save the account name and clear inputText for the API key screen
		m.accountName = accountName
		m.inputText = "" // Clear for API key input
		return m.pushScreen(screenAddAPIKey), nil
	case tea.KeyBackspace:
		if len(m.inputText) > 0 {
			m.inputText = m.inputText[:len(m.inputText)-1]
		}
	case tea.KeyCtrlH:
		if len(m.inputText) > 0 {
			m.inputText = m.inputText[:len(m.inputText)-1]
		}
	default:
		// Accept runes (character input including paste)
		if len(msg.Runes) > 0 {
			m.inputText += string(msg.Runes)
		}
	}
	return m, nil
}

// checkAccountExists checks if an account with the same name already exists
func (m Model) checkAccountExists(accountName string) error {
	if m.credsMgr.ProviderExists(m.selectedProvider) {
		accounts, err := m.credsMgr.ListAccounts(m.selectedProvider)
		if err != nil {
			return err
		}
		for _, acc := range accounts {
			if acc == accountName {
				return fmt.Errorf("account '%s' already exists", accountName)
			}
		}
	}
	return nil
}

// viewAddAccountName renders the account name input screen
func (m Model) viewAddAccountName() string {
	var b strings.Builder

	providerName := ""
	for _, p := range AllProviders {
		if p.ID == m.selectedProvider {
			providerName = p.Name
			break
		}
	}

	b.WriteString(titleStyle.Render(fmt.Sprintf("Add %s Account", providerName)))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("Enter a name for this account"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("(Leave empty for 'default')"))
	b.WriteString("\n\n")

	cursor := cursorStyle.Render("▶")
	input := m.inputText
	if input == "" {
		input = dimStyle.Render("default")
	} else {
		input = inputFieldStyle.Render(input)
	}
	b.WriteString(cursor + " Name: " + input + "_")

	if m.errorMsg != "" {
		b.WriteString("\n\n" + RenderError(m.errorMsg))
	}

	return b.String()
}

// updateAddAPIKey handles updates for the API key input screen
func (m Model) updateAddAPIKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.inputText == "" {
			m.errorMsg = "API key is required"
			return m, nil
		}
		// Save the account
		return m.saveAccount()
	case tea.KeyBackspace:
		if len(m.inputText) > 0 {
			m.inputText = m.inputText[:len(m.inputText)-1]
		}
	case tea.KeyCtrlH:
		if len(m.inputText) > 0 {
			m.inputText = m.inputText[:len(m.inputText)-1]
		}
	default:
		// Accept runes (character input including paste)
		// This handles clipboard paste and all keyboard input
		if len(msg.Runes) > 0 {
			m.inputText += string(msg.Runes)
		}
	}
	return m, nil
}

// saveAccount saves the account credentials
func (m Model) saveAccount() (tea.Model, tea.Cmd) {
	accountName := m.accountName
	apiKey := m.inputText
	var err error

	switch m.selectedProvider {
	case "kimi":
		var creds credentials.KimiCredentials
		if m.credsMgr.ProviderExists("kimi") {
			m.credsMgr.LoadProvider("kimi", &creds)
		}
		if creds.Accounts == nil {
			creds.Accounts = make(map[string]*credentials.KimiAccount)
		}
		creds.Accounts[accountName] = &credentials.KimiAccount{APIKey: apiKey}
		err = m.credsMgr.SaveProvider("kimi", creds)
	case "zai":
		var creds credentials.ZAiCredentials
		if m.credsMgr.ProviderExists("zai") {
			m.credsMgr.LoadProvider("zai", &creds)
		}
		if creds.Accounts == nil {
			creds.Accounts = make(map[string]*credentials.ZAiAccount)
		}
		creds.Accounts[accountName] = &credentials.ZAiAccount{APIKey: apiKey}
		err = m.credsMgr.SaveProvider("zai", creds)
	default:
		err = fmt.Errorf("unsupported provider: %s", m.selectedProvider)
	}

	if err != nil {
		m.errorMsg = err.Error()
		return m, nil
	}

	m.successMsg = fmt.Sprintf("Successfully added %s account '%s'", m.selectedProvider, accountName)
	m.screen = screenSuccess
	return m, nil
}

// viewAddAPIKey renders the API key input screen
func (m Model) viewAddAPIKey() string {
	var b strings.Builder

	providerName := ""
	for _, p := range AllProviders {
		if p.ID == m.selectedProvider {
			providerName = p.Name
			break
		}
	}

	b.WriteString(titleStyle.Render(fmt.Sprintf("Add %s Account", providerName)))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("Enter your API key"))
	b.WriteString("\n\n")

	cursor := cursorStyle.Render("▶")
	// Mask the API key for display
	maskedKey := strings.Repeat("*", len(m.inputText))
	if maskedKey == "" {
		maskedKey = dimStyle.Render("(empty)")
	} else {
		maskedKey = inputFieldStyle.Render(maskedKey)
	}
	b.WriteString(cursor + " API Key: " + maskedKey + "_")

	if m.errorMsg != "" {
		b.WriteString("\n\n" + RenderError(m.errorMsg))
	}

	return b.String()
}

// viewListAccounts renders the list accounts screen
func (m Model) viewListAccounts() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Configured Accounts"))
	b.WriteString("\n\n")

	providers := m.credsMgr.ListAvailable()
	if len(providers) == 0 {
		b.WriteString(normalStyle.Render("No providers configured."))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Run 'llm-usage setup' to configure providers."))
		return b.String()
	}

	for _, providerID := range providers {
		providerName := providerID
		for _, p := range AllProviders {
			if p.ID == providerID {
				providerName = p.Name
				break
			}
		}

		b.WriteString(providerStyle.Render(providerName))
		b.WriteString("\n")

		accounts, err := m.credsMgr.ListAccounts(providerID)
		if err != nil {
			b.WriteString(normalStyle.Render("  (error loading accounts)"))
		} else if len(accounts) == 0 {
			b.WriteString(dimStyle.Render("  (no accounts configured)"))
		} else {
			for _, acc := range accounts {
				b.WriteString(normalStyle.Render("  • " + acc))
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// updateListAccounts handles updates for the list accounts screen
func (m Model) updateListAccounts(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key goes back to main menu
	return m.goBack()
}
