// Package tui provides the Bubble Tea TUI for the setup wizard.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Main menu items
var mainMenuItems = []string{
	"Add Account",
	"List Accounts",
	"Remove Account",
	"Quit",
}

// updateMainMenu handles updates for the main menu
func (m Model) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		if m.selectedIdx < len(mainMenuItems)-1 {
			m.selectedIdx++
		}
	case "enter", " ":
		return m.handleMainMenuSelection()
	}

	return m, nil
}

// handleMainMenuSelection handles the selection from the main menu
func (m Model) handleMainMenuSelection() (tea.Model, tea.Cmd) {
	switch m.selectedIdx {
	case 0: // Add Account
		return m.pushScreen(screenProviderSelect), nil
	case 1: // List Accounts
		return m.pushScreen(screenListAccounts), nil
	case 2: // Remove Account
		return m.pushScreen(screenRemoveProviderSelect), nil
	case 3: // Quit
		return m, tea.Quit
	}
	return m, nil
}

// viewMainMenu renders the main menu
func (m Model) viewMainMenu() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("LLM Usage Setup"))
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render("Configure your LLM provider credentials"))
	b.WriteString("\n\n")

	// Menu items
	for i, item := range mainMenuItems {
		cursor := " "
		if i == m.selectedIdx {
			cursor = cursorStyle.Render("▶")
			b.WriteString(cursor + " " + selectedStyle.Render(item) + "\n")
		} else {
			b.WriteString(cursor + " " + normalStyle.Render(item) + "\n")
		}
	}

	return b.String()
}

// viewFooter renders the footer help text
func (m Model) viewFooter() string {
	var bindings []string

	switch m.screen {
	case screenMainMenu:
		bindings = []string{"↑/k", "↓/j", "enter", "q"}
	case screenProviderSelect, screenRemoveProviderSelect:
		bindings = []string{"↑/k", "↓/j", "enter", "esc"}
	case screenAddAccountName, screenAddAPIKey:
		bindings = []string{"type", "enter", "esc"}
	case screenListAccounts:
		bindings = []string{"esc"}
	case screenRemoveAccountSelect, screenRemoveConfirm:
		bindings = []string{"↑/k", "↓/j", "enter", "esc"}
	case screenSuccess:
		bindings = []string{"any key"}
	default:
		bindings = []string{"↑/k", "↓/j", "enter", "esc"}
	}

	helpParts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		helpParts = append(helpParts, dimStyle.Render(binding))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, helpParts...)
}
