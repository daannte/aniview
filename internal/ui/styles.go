package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// UI styles definitions
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD"))

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	TabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A8A8A8")).
			Padding(0, 1).
			Align(lipgloss.Center).
			Border(lipgloss.NormalBorder(), false, false, false, false) // No bottom border

	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Align(lipgloss.Center).
			Bold(true) // Bold text for active tab

	TabGap = lipgloss.NewStyle().Width(1)
)
