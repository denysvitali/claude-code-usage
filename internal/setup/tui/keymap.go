// Package tui provides the Bubble Tea TUI for the setup wizard.
package tui

import "github.com/charmbracelet/lipgloss"

// KeyMap defines key bindings for the TUI
type KeyMap struct {
	Up       keyBinding
	Down     keyBinding
	Enter    keyBinding
	Escape   keyBinding
	Quit     keyBinding
	Back     keyBinding
	Confirm  keyBinding
	Cancel   keyBinding
	Help     keyBinding
}

// keyBinding represents a single key binding
type keyBinding struct {
	keys []string
	help string
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: keyBinding{
			keys: []string{"↑", "k"},
			help: "up",
		},
		Down: keyBinding{
			keys: []string{"↓", "j"},
			help: "down",
		},
		Enter: keyBinding{
			keys: []string{"enter"},
			help: "select",
		},
		Escape: keyBinding{
			keys: []string{"esc"},
			help: "back",
		},
		Quit: keyBinding{
			keys: []string{"q", "ctrl+c"},
			help: "quit",
		},
		Back: keyBinding{
			keys: []string{"esc"},
			help: "back",
		},
		Confirm: keyBinding{
			keys: []string{"enter"},
			help: "confirm",
		},
		Cancel: keyBinding{
			keys: []string{"esc"},
			help: "cancel",
		},
		Help: keyBinding{
			keys: []string{"?"},
			help: "help",
		},
	}
}

// HelpView returns a formatted help view
func (k KeyMap) HelpView(bindings ...keyBinding) string {
	var helpItems []string
	for _, b := range bindings {
		helpItems = append(helpItems, lipgloss.NewStyle().
			Foreground(dimColor).
			Render(b.help))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, helpItems...)
}

// ShortHelp returns short help for the footer
func (k KeyMap) ShortHelp() []keyBinding {
	return []keyBinding{k.Up, k.Down, k.Enter, k.Escape}
}
