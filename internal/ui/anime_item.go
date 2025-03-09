package ui

import (
	"fmt"
	"strings"

	"github.com/daannte/aniview/internal"
)

// AnimeItem represents an item in the list UI
type AnimeItem struct {
	AnimeEntry internal.AnimeEntry
	Index      int // Store the original index in the list
}

func (i AnimeItem) Title() string {
	return i.AnimeEntry.Title
}

func (i AnimeItem) Description() string {
	var status string

	// Check if the anime is currently airing
	if i.AnimeEntry.IsAiring {
		status = fmt.Sprintf("%d/%d episodes (Currently airing)", i.AnimeEntry.Progress, i.AnimeEntry.Episodes)
	} else if i.AnimeEntry.Episodes > 0 {
		status = fmt.Sprintf("%d/%d episodes", i.AnimeEntry.Progress, i.AnimeEntry.Episodes)
	}
	return status
}

func (i AnimeItem) FilterValue() string {
	return i.AnimeEntry.Title
}

// DetailedView returns a detailed view of the anime with title, description, and airing info
func (i AnimeItem) DetailedView() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(i.AnimeEntry.Title))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Progress: %d/%d episodes\n", i.AnimeEntry.Progress, i.AnimeEntry.Episodes))

	if i.AnimeEntry.IsAiring {
		b.WriteString("Status: Currently Airing\n")
	} else {
		b.WriteString("Status: Completed\n")
	}

	if i.AnimeEntry.IsAiring {
		timeUntil := FormatTimeUntil(i.AnimeEntry.NextAiringEpisode.TimeUntilAiring)
		b.WriteString(fmt.Sprintf("\nNext Episode: Episode %d\n", i.AnimeEntry.NextAiringEpisode.Episode))
		b.WriteString(fmt.Sprintf("Airing: %s\n", timeUntil))
	}

	// Add a separator before description
	b.WriteString("\nDescription:\n")

	// Format and add the description with word wrapping
	if i.AnimeEntry.Description != "" {
		// Simple word wrapping for description
		wrappedDesc := WordWrap(i.AnimeEntry.Description, 120)
		b.WriteString(wrappedDesc)
	} else {
		b.WriteString("No description available.")
	}

	return b.String()
}
