// Package tui provides the Bubble Tea TUI for the setup wizard.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/denysvitali/llm-usage/internal/credentials"
	"github.com/denysvitali/llm-usage/internal/setup"
)

// updateProviderSelect handles updates for the provider selection screen
func (m Model) updateProviderSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyUp, "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case keyDown, "j":
		if m.selectedIdx < len(AllProviders)-1 {
			m.selectedIdx++
		}
	case keyEnter:
		provider := AllProviders[m.selectedIdx]
		// Claude requires special handling (OAuth)
		if provider.ID == credentials.ProviderClaude {
			m.selectedProvider = provider.ID
			m.errorMsg = "Claude uses OAuth. Please run: llm-usage setup add claude"
			return m, nil
		}
		if provider.ID == credentials.ProviderCodex {
			m.selectedProvider = provider.ID
			m.errorMsg = "Codex uses the local Codex CLI session; no setup is required"
			return m, nil
		}
		if provider.ID == credentials.ProviderGrok {
			m.selectedProvider = provider.ID
			m.errorMsg = "Grok consumer quotas use the grok.json snapshot; no API key is required"
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
	switch msg.Type { //nolint:exhaustive
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
		// Save the account name and clear inputText for the next screen
		m.accountName = accountName
		m.inputText = ""
		// MiniMax needs a group ID before the cookie
		if m.selectedProvider == credentials.ProviderMiniMax {
			return m.pushScreen(screenAddGroupID), nil
		}
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

// updateAddGroupID handles updates for the MiniMax group ID input screen
func (m Model) updateAddGroupID(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type { //nolint:exhaustive
	case tea.KeyEnter:
		if m.inputText == "" {
			m.errorMsg = "group ID is required"
			return m, nil
		}
		m.groupID = m.inputText
		m.inputText = "" // Clear for the cookie input
		return m.pushScreen(screenAddAPIKey), nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		if len(m.inputText) > 0 {
			m.inputText = m.inputText[:len(m.inputText)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			m.inputText += string(msg.Runes)
		}
	}
	return m, nil
}

// viewAddGroupID renders the MiniMax group ID input screen
func (m Model) viewAddGroupID() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Add MiniMax Account"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("Enter your MiniMax Group ID"))
	b.WriteString("\n\n")

	cursor := cursorStyle.Render("▶")
	input := m.inputText
	if input == "" {
		input = dimStyle.Render("(empty)")
	} else {
		input = inputFieldStyle.Render(input)
	}
	b.WriteString(cursor + " Group ID: " + input + "_")

	if m.errorMsg != "" {
		b.WriteString("\n\n" + RenderError(m.errorMsg))
	}

	return b.String()
}

// updateAddAPIKey handles updates for the API key input screen
func (m Model) updateAddAPIKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type { //nolint:exhaustive
	case tea.KeyEnter:
		if m.inputText == "" {
			m.errorMsg = m.secretLabel() + " is required"
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

// secretLabel returns the label for the secret input of the selected provider
func (m Model) secretLabel() string {
	if m.selectedProvider == credentials.ProviderMiniMax {
		return "Cookie"
	}
	return "API key"
}

// saveAccount saves the account credentials via the shared setup save path
func (m Model) saveAccount() (tea.Model, tea.Cmd) {
	var err error
	switch m.selectedProvider {
	case credentials.ProviderKimi:
		err = setup.SaveAPIKeyAccount(m.credsMgr, m.selectedProvider, m.accountName, m.inputText)
	case credentials.ProviderMiniMax:
		err = setup.SaveMiniMaxAccount(m.credsMgr, m.accountName, m.inputText, m.groupID)
	default:
		err = fmt.Errorf("unsupported provider: %s", m.selectedProvider)
	}

	if err != nil {
		m.errorMsg = err.Error()
		return m, nil
	}

	m.successMsg = fmt.Sprintf("Successfully added %s account '%s'", m.selectedProvider, m.accountName)
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
	b.WriteString(normalStyle.Render("Enter your " + m.secretLabel()))
	b.WriteString("\n\n")

	cursor := cursorStyle.Render("▶")
	// Mask the secret for display
	maskedKey := strings.Repeat("*", len(m.inputText))
	if maskedKey == "" {
		maskedKey = dimStyle.Render("(empty)")
	} else {
		maskedKey = inputFieldStyle.Render(maskedKey)
	}
	b.WriteString(cursor + " " + m.secretLabel() + ": " + maskedKey + "_")

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
		switch {
		case err != nil:
			b.WriteString(normalStyle.Render("  (error loading accounts)"))
		case len(accounts) == 0:
			b.WriteString(dimStyle.Render("  (no accounts configured)"))
		default:
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
func (m Model) updateListAccounts(_ tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key goes back to main menu
	return m.goBack()
}
