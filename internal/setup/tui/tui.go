// Package tui provides the Bubble Tea TUI for the setup wizard.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/denysvitali/llm-usage/internal/credentials"
)

// Provider represents an LLM provider that can be configured
type Provider struct {
	ID   string
	Name string
}

// All providers that can be configured
var AllProviders = []Provider{
	{ID: "claude", Name: "Claude (Anthropic)"},
	{ID: "kimi", Name: "Kimi"},
	{ID: "zai", Name: "Z.AI"},
}

// Model represents the state of the TUI
type Model struct {
	// Screen state
	screen        screen
	width, height int
	credsMgr      *credentials.Manager

	// Menu navigation
	selectedIdx int

	// Input state
	inputText      string
	cursorPosition int

	// Selection state
	selectedProvider string
	selectedAccount  string
	accountName      string // Name for new account being added
	accounts         []string
	confirmRemove    bool

	// Messages
	successMsg string
	errorMsg   string

	// Screen history for navigation
	screenHistory []screen

	// Key bindings
	keys KeyMap
}

// NewModel creates a new TUI model with the given credentials manager
func NewModel(mgr *credentials.Manager) Model {
	return Model{
		screen:       screenMainMenu,
		credsMgr:     mgr,
		selectedIdx:  0,
		keys:         DefaultKeyMap(),
		screenHistory: []screen{},
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case screenChangeMsg:
		m.screen = msg.screen
		m.selectedIdx = 0
		m.errorMsg = ""
		return m, nil

	case providerSelectedMsg:
		m.selectedProvider = msg.provider
		return m, changeScreen(screenAddAccountName)

	case accountSavedMsg:
		m.successMsg = fmt.Sprintf("Successfully added %s account '%s'", msg.provider, msg.account)
		return m, changeScreen(screenSuccess)

	case accountRemovedMsg:
		m.successMsg = fmt.Sprintf("Successfully removed %s account '%s'", msg.provider, msg.account)
		return m, changeScreen(screenSuccess)

	case errorMsg:
		m.errorMsg = msg.err.Error()
		return m, nil

	case clearErrorMsg:
		m.errorMsg = ""
		return m, nil

	case returnToMainMenuMsg:
		return m.returnToMainMenu()
	}

	return m, nil
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	switch msg.String() {
	case "ctrl+c", "q":
		if m.screen == screenMainMenu {
			return m, tea.Quit
		}
		return m.returnToMainMenu()
	}

	// Handle ESC for back navigation
	if msg.String() == "esc" {
		return m.goBack()
	}

	// Screen-specific handling
	switch m.screen {
	case screenMainMenu:
		return m.updateMainMenu(msg)

	case screenProviderSelect:
		return m.updateProviderSelect(msg)

	case screenAddAccountName:
		return m.updateAddAccountName(msg)

	case screenAddAPIKey:
		return m.updateAddAPIKey(msg)

	case screenListAccounts:
		return m.updateListAccounts(msg)

	case screenRemoveProviderSelect:
		return m.updateRemoveProviderSelect(msg)

	case screenRemoveAccountSelect:
		return m.updateRemoveAccountSelect(msg)

	case screenRemoveConfirm:
		return m.updateRemoveConfirm(msg)

	case screenSuccess:
		// Any key returns to main menu
		return m.returnToMainMenu()
	}

	return m, nil
}

// View renders the current screen
func (m Model) View() string {
	if m.width == 0 {
		m.width = 80
	}

	var content strings.Builder

	switch m.screen {
	case screenMainMenu:
		content.WriteString(m.viewMainMenu())

	case screenProviderSelect:
		content.WriteString(m.viewProviderSelect())

	case screenAddAccountName:
		content.WriteString(m.viewAddAccountName())

	case screenAddAPIKey:
		content.WriteString(m.viewAddAPIKey())

	case screenListAccounts:
		content.WriteString(m.viewListAccounts())

	case screenRemoveProviderSelect:
		content.WriteString(m.viewRemoveProviderSelect())

	case screenRemoveAccountSelect:
		content.WriteString(m.viewRemoveAccountSelect())

	case screenRemoveConfirm:
		content.WriteString(m.viewRemoveConfirm())

	case screenSuccess:
		content.WriteString(m.viewSuccess())
	}

	// Add footer help
	content.WriteString("\n\n")
	content.WriteString(m.viewFooter())

	return lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Width(m.width).Height(m.height).Render(content.String()),
	)
}

// returnToMainMenu returns to the main menu
func (m Model) returnToMainMenu() (tea.Model, tea.Cmd) {
	m.screen = screenMainMenu
	m.selectedIdx = 0
	m.errorMsg = ""
	m.successMsg = ""
	m.inputText = ""
	m.selectedProvider = ""
	m.selectedAccount = ""
	m.accountName = ""
	m.accounts = nil
	m.confirmRemove = false
	m.screenHistory = []screen{}
	return m, nil
}

// goBack navigates to the previous screen
func (m Model) goBack() (tea.Model, tea.Cmd) {
	if len(m.screenHistory) > 0 {
		// Pop the last screen from history
		lastIdx := len(m.screenHistory) - 1
		prevScreen := m.screenHistory[lastIdx]
		m.screenHistory = m.screenHistory[:lastIdx]
		m.screen = prevScreen
		m.selectedIdx = 0
		m.errorMsg = ""
		return m, nil
	}
	return m, nil
}

// pushScreen pushes a screen onto the history stack and returns the updated model
func (m Model) pushScreen(s screen) Model {
	m.screenHistory = append(m.screenHistory, m.screen)
	m.screen = s
	m.selectedIdx = 0
	return m
}

// loadAccounts loads accounts for the selected provider and returns the updated model
func (m Model) loadAccounts() (Model, error) {
	accounts, err := m.credsMgr.ListAccounts(m.selectedProvider)
	if err != nil {
		return m, err
	}
	m.accounts = accounts
	return m, nil
}
