package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/daannte/aniview/internal"
	"github.com/daannte/aniview/internal/ui"
)

func main() {
	// Ensure config exists
	config, err := internal.EnsureConfigExists()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	anilist := internal.NewAniListClient(config.Token)

	// Start the UI
	m := ui.NewModel(config, anilist)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running UI: %v\n", err)
		os.Exit(1)
	}

	defer internal.CleanupSocket("/tmp/iinasocket")
}
