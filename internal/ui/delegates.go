package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CompactDelegate is a custom delegate for episode items to make them more compact
type CompactDelegate struct {
	styles  list.DefaultItemStyles
	height  int
	spacing int
}

// Height returns the height of the item
func (d CompactDelegate) Height() int {
	return d.height
}

// Spacing returns the spacing between items
func (d CompactDelegate) Spacing() int {
	return d.spacing
}

// Update is called when a message is received
func (d CompactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders the item
func (d CompactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	if i, ok := item.(EpisodeItem); ok {
		title = i.Title()
	} else {
		title = "Unknown item"
	}

	if selected := index == m.Index(); selected {
		fmt.Fprint(w, SelectedStyle.Render("> "+title))
	} else {
		fmt.Fprint(w, "  "+title)
	}
}

// NewCompactDelegate creates a new compact delegate for episode items
func NewCompactDelegate() list.ItemDelegate {
	styles := list.NewDefaultItemStyles()
	styles.SelectedTitle = styles.SelectedTitle.Foreground(lipgloss.Color("#04B575"))

	return &CompactDelegate{
		styles:  styles,
		height:  1, // Single line height
		spacing: 0, // No spacing between items
	}
}
