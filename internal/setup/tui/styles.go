// Package tui provides the Bubble Tea TUI for the setup wizard.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Base colors
	titleColor      = lipgloss.Color("86")  // Cyan
	subtitleColor   = lipgloss.Color("244") // Gray
	cursorColor     = lipgloss.Color("213") // Pink
	selectedColor   = lipgloss.Color("226") // Yellow
	successColor    = lipgloss.Color("70")  // Green
	errorColor      = lipgloss.Color("203") // Red
	dimColor        = lipgloss.Color("241") // Dim gray
	mutedColor      = lipgloss.Color("245") // Muted gray
	inputFocusColor = lipgloss.Color("117") // Light blue

	// Background colors
	borderColor = lipgloss.Color("238")
)

// Style definitions
var (
	// Title style
	titleStyle = lipgloss.NewStyle().
			Foreground(titleColor).
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	// Subtitle style
	subtitleStyle = lipgloss.NewStyle().
			Foreground(subtitleColor)

	// Cursor style
	cursorStyle = lipgloss.NewStyle().
			Foreground(cursorColor).
			Bold(true)

	// Selected item style
	selectedStyle = lipgloss.NewStyle().
			Foreground(selectedColor).
			Bold(true)

	// Normal item style
	normalStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Dimmed item style
	dimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Success style
	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	// Input field style
	inputFieldStyle = lipgloss.NewStyle().
			Foreground(inputFocusColor).
			Bold(true)

	// Input placeholder style
	inputPlaceholderStyle = lipgloss.NewStyle().
				Foreground(dimColor)

	// Separator style
	separatorStyle = lipgloss.NewStyle().
			Foreground(borderColor)

	// Provider name style
	providerStyle = lipgloss.NewStyle().
			Foreground(titleColor).
			Bold(true)

	// Warning style
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")). // Orange
			Bold(true)
)

// RenderCursor returns the cursor indicator
func RenderCursor(isActive bool) string {
	if isActive {
		return cursorStyle.Render("▶")
	}
	return " "
}

// RenderMenuItem returns a styled menu item
func RenderMenuItem(label string, isActive bool) string {
	if isActive {
		return selectedStyle.Render(label)
	}
	return normalStyle.Render(label)
}

// RenderInputField returns a styled input field
func RenderInputField(label, value, placeholder string, isActive, isPassword bool) string {
	var renderedValue string
	switch {
	case isPassword && value != "":
		renderedValue = strings.Repeat("*", len(value))
	case value == "":
		renderedValue = inputPlaceholderStyle.Render(placeholder)
	default:
		renderedValue = inputFieldStyle.Render(value)
	}

	var prefix string
	if isActive {
		prefix = cursorStyle.Render("▶ ")
	} else {
		prefix = "  "
	}

	return prefix + normalStyle.Render(label) + ": " + renderedValue
}

// RenderError returns a styled error message
func RenderError(msg string) string {
	return errorStyle.Render("✗ " + msg)
}

// RenderSuccess returns a styled success message
func RenderSuccess(msg string) string {
	return successStyle.Render("✓ " + msg)
}

// RenderWarning returns a styled warning message
func RenderWarning(msg string) string {
	return warningStyle.Render("⚠ " + msg)
}

// RenderSeparator returns a horizontal separator
func RenderSeparator(width int) string {
	return separatorStyle.Render(strings.Repeat("─", width))
}
