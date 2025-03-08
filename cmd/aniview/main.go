package main

import (
	"fmt"
	"os"

	"github.com/daannte/aniview/internal"
)

func main() {
	// Ensure config exists
	config, err := internal.EnsureConfigExists()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Start the UI
	if err := internal.StartUI(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error running UI: %v\n", err)
		os.Exit(1)
	}

	defer internal.CleanupSocket("/tmp/iinasocket")
}
