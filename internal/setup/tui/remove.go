// Package tui provides the Bubble Tea TUI for the setup wizard.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dvitali/llm-usage/internal/credentials"
)

// updateRemoveProviderSelect handles updates for the provider selection (remove) screen
func (m Model) updateRemoveProviderSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		// Only show providers that have accounts
		availableProviders := m.getProvidersWithAccounts()
		if m.selectedIdx < len(availableProviders)-1 {
			m.selectedIdx++
		}
	case "enter":
		availableProviders := m.getProvidersWithAccounts()
		if len(availableProviders) == 0 {
			m.errorMsg = "No providers configured"
			return m, nil
		}
		m.selectedProvider = availableProviders[m.selectedIdx]
		// Load accounts for this provider
		var err error
		m, err = m.loadAccounts()
		if err != nil {
			m.errorMsg = err.Error()
			return m, nil
		}
		if len(m.accounts) == 0 {
			m.errorMsg = "No accounts found for this provider"
			return m, nil
		}
		return m.pushScreen(screenRemoveAccountSelect), nil
	}
	return m, nil
}

// getProvidersWithAccounts returns a list of provider IDs that have accounts
func (m Model) getProvidersWithAccounts() []string {
	var providers []string
	allProviders := m.credsMgr.ListAvailable()
	for _, pid := range allProviders {
		accounts, err := m.credsMgr.ListAccounts(pid)
		if err == nil && len(accounts) > 0 {
			providers = append(providers, pid)
		}
	}
	return providers
}

// viewRemoveProviderSelect renders the provider selection (remove) screen
func (m Model) viewRemoveProviderSelect() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select Provider to Remove Account"))
	b.WriteString("\n\n")

	availableProviders := m.getProvidersWithAccounts()
	if len(availableProviders) == 0 {
		b.WriteString(normalStyle.Render("No providers configured."))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Run 'llm-usage setup' to configure providers."))
		return b.String()
	}

	for i, providerID := range availableProviders {
		providerName := providerID
		for _, p := range AllProviders {
			if p.ID == providerID {
				providerName = p.Name
				break
			}
		}

		cursor := " "
		if i == m.selectedIdx {
			cursor = cursorStyle.Render("▶")
			b.WriteString(cursor + " " + selectedStyle.Render(providerName) + "\n")
		} else {
			b.WriteString(cursor + " " + normalStyle.Render(providerName) + "\n")
		}
	}

	if m.errorMsg != "" {
		b.WriteString("\n" + RenderError(m.errorMsg))
	}

	return b.String()
}

// updateRemoveAccountSelect handles updates for the account selection (remove) screen
func (m Model) updateRemoveAccountSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		if m.selectedIdx < len(m.accounts)-1 {
			m.selectedIdx++
		}
	case "enter":
		m.selectedAccount = m.accounts[m.selectedIdx]
		return m.pushScreen(screenRemoveConfirm), nil
	}
	return m, nil
}

// viewRemoveAccountSelect renders the account selection (remove) screen
func (m Model) viewRemoveAccountSelect() string {
	var b strings.Builder

	providerName := ""
	for _, p := range AllProviders {
		if p.ID == m.selectedProvider {
			providerName = p.Name
			break
		}
	}

	b.WriteString(titleStyle.Render("Select Account to Remove"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render(fmt.Sprintf("Provider: %s", providerName)))
	b.WriteString("\n\n")

	if len(m.accounts) == 0 {
		b.WriteString(normalStyle.Render("No accounts configured for this provider."))
		return b.String()
	}

	for i, account := range m.accounts {
		cursor := " "
		if i == m.selectedIdx {
			cursor = cursorStyle.Render("▶")
			b.WriteString(cursor + " " + selectedStyle.Render(account) + "\n")
		} else {
			b.WriteString(cursor + " " + normalStyle.Render(account) + "\n")
		}
	}

	return b.String()
}

// updateRemoveConfirm handles updates for the remove confirmation screen
func (m Model) updateRemoveConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		m.confirmRemove = false
	case "right", "l":
		m.confirmRemove = true
	case "up", "k":
		m.confirmRemove = !m.confirmRemove
	case "down", "j":
		m.confirmRemove = !m.confirmRemove
	case "enter":
		if m.confirmRemove {
			return m.doRemoveAccount()
		} else {
			return m.goBack()
		}
	}
	return m, nil
}

// doRemoveAccount performs the actual account removal
func (m Model) doRemoveAccount() (tea.Model, tea.Cmd) {
	var err error

	switch m.selectedProvider {
	case "claude":
		var creds credentials.ClaudeCredentials
		if loadErr := m.credsMgr.LoadProvider("claude", &creds); loadErr != nil {
			err = loadErr
		} else {
			// Handle legacy format (single ClaudeAiOauth field)
			if creds.Accounts == nil {
				if creds.ClaudeAiOauth != nil && m.selectedAccount == "default" {
					// Delete the entire provider file for legacy format
					err = m.credsMgr.DeleteProvider("claude")
				} else {
					err = fmt.Errorf("account '%s' not found", m.selectedAccount)
				}
			} else {
				if creds.Accounts[m.selectedAccount] == nil {
					err = fmt.Errorf("account '%s' not found", m.selectedAccount)
				} else {
					delete(creds.Accounts, m.selectedAccount)
					if len(creds.Accounts) == 0 {
						err = m.credsMgr.DeleteProvider("claude")
					} else {
						err = m.credsMgr.SaveProvider("claude", creds)
					}
				}
			}
		}
	case "kimi":
		var creds credentials.KimiCredentials
		if loadErr := m.credsMgr.LoadProvider("kimi", &creds); loadErr != nil {
			err = loadErr
		} else {
			// Handle legacy format (single APIKey field)
			if creds.Accounts == nil {
				if creds.APIKey != "" && m.selectedAccount == "default" {
					// Delete the entire provider file for legacy format
					err = m.credsMgr.DeleteProvider("kimi")
				} else {
					err = fmt.Errorf("account '%s' not found", m.selectedAccount)
				}
			} else {
				if creds.Accounts[m.selectedAccount] == nil {
					err = fmt.Errorf("account '%s' not found", m.selectedAccount)
				} else {
					delete(creds.Accounts, m.selectedAccount)
					if len(creds.Accounts) == 0 {
						err = m.credsMgr.DeleteProvider("kimi")
					} else {
						err = m.credsMgr.SaveProvider("kimi", creds)
					}
				}
			}
		}
	case "zai":
		var creds credentials.ZAiCredentials
		if loadErr := m.credsMgr.LoadProvider("zai", &creds); loadErr != nil {
			err = loadErr
		} else {
			// Handle legacy format (single APIKey field)
			if creds.Accounts == nil {
				if creds.APIKey != "" && m.selectedAccount == "default" {
					// Delete the entire provider file for legacy format
					err = m.credsMgr.DeleteProvider("zai")
				} else {
					err = fmt.Errorf("account '%s' not found", m.selectedAccount)
				}
			} else {
				if creds.Accounts[m.selectedAccount] == nil {
					err = fmt.Errorf("account '%s' not found", m.selectedAccount)
				} else {
					delete(creds.Accounts, m.selectedAccount)
					if len(creds.Accounts) == 0 {
						err = m.credsMgr.DeleteProvider("zai")
					} else {
						err = m.credsMgr.SaveProvider("zai", creds)
					}
				}
			}
		}
	default:
		err = fmt.Errorf("unsupported provider: %s", m.selectedProvider)
	}

	if err != nil {
		m.errorMsg = err.Error()
		return m, nil
	}

	m.successMsg = fmt.Sprintf("Successfully removed %s account '%s'", m.selectedProvider, m.selectedAccount)
	m.screen = screenSuccess
	return m, nil
}

// viewRemoveConfirm renders the remove confirmation screen
func (m Model) viewRemoveConfirm() string {
	var b strings.Builder

	providerName := ""
	for _, p := range AllProviders {
		if p.ID == m.selectedProvider {
			providerName = p.Name
			break
		}
	}

	b.WriteString(titleStyle.Render("Confirm Removal"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render(fmt.Sprintf("Remove account '%s' from %s?", m.selectedAccount, providerName)))
	b.WriteString("\n\n")

	// Yes/No options
	yesCursor := " "
	noCursor := " "
	if m.confirmRemove {
		yesCursor = cursorStyle.Render("▶")
	} else {
		noCursor = cursorStyle.Render("▶")
	}

	b.WriteString(yesCursor + " " + RenderMenuItem("Yes", m.confirmRemove) + "\n")
	b.WriteString(noCursor + " " + RenderMenuItem("No", !m.confirmRemove))

	return b.String()
}

// viewSuccess renders the success screen
func (m Model) viewSuccess() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Success"))
	b.WriteString("\n\n")
	b.WriteString(successStyle.Render(m.successMsg))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Press any key to continue..."))

	return b.String()
}
