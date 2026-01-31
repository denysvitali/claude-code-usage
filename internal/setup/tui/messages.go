// Package tui provides the Bubble Tea TUI for the setup wizard.
package tui

import tea "github.com/charmbracelet/bubbletea"

// screen represents the different screens in the TUI
type screen int

const (
	screenMainMenu screen = iota
	screenProviderSelect
	screenAddAccountName
	screenAddAPIKey
	screenListAccounts
	screenRemoveProviderSelect
	screenRemoveAccountSelect
	screenRemoveConfirm
	screenSuccess
)

// screenChangeMsg is a custom message to change screens
type screenChangeMsg struct {
	screen screen
}

// changeScreen returns a command to change to the specified screen
func changeScreen(s screen) tea.Cmd {
	return func() tea.Msg {
		return screenChangeMsg{screen: s}
	}
}

// providerSelectedMsg is sent when a provider is selected
type providerSelectedMsg struct {
	provider string
}

// accountSavedMsg is sent when an account is successfully saved
type accountSavedMsg struct {
	provider string
	account  string
}

// accountRemovedMsg is sent when an account is successfully removed
type accountRemovedMsg struct {
	provider string
	account  string
}

// errorMsg is a custom error message
type errorMsg struct {
	err error
}

// clearErrorMsg is a message to clear the current error
type clearErrorMsg struct{}

// returnToMainMenuMsg is a message to return to the main menu
type returnToMainMenuMsg struct{}
