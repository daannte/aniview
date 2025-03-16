package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FormatTimeUntil formats time until airing in a human-readable format
func FormatTimeUntil(seconds int) string {
	if seconds <= 0 {
		return "Airing now"
	}

	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("in %d days, %d hours", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("in %d hours, %d minutes", hours, minutes)
	} else {
		return fmt.Sprintf("in %d minutes", minutes)
	}
}

// WordWrap provides simple word wrapping functionality
func WordWrap(text string, lineWidth int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	lineLength := 0

	for _, word := range words {
		word = strings.ReplaceAll(word, "<br>", "\n")
		word = strings.ReplaceAll(word, "<i>", "")
		word = strings.ReplaceAll(word, "</i>", "")
		word = strings.ReplaceAll(word, "<b>", "")
		word = strings.ReplaceAll(word, "</b>", "")

		parts := strings.Split(word, "\n")
		for i, part := range parts {
			if i > 0 {
				result.WriteString("\n")
				lineLength = 0
			}
			if len(part) == 0 {
				continue // Skip empty parts from consecutive \n
			}

			if lineLength+len(part)+1 > lineWidth {
				result.WriteString("\n")
				lineLength = 0
			} else if lineLength > 0 {
				result.WriteString(" ")
				lineLength++
			}

			result.WriteString(part)
			lineLength += len(part)
		}
	}

	return result.String()
}

// RenderTabs renders the tab bar for the UI
func RenderTabs(tabs []string, activeTab int) string {
	var renderedTabs []string

	for i, tab := range tabs {
		if i == activeTab {
			renderedTabs = append(renderedTabs, ActiveTabStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, TabStyle.Render(tab))
		}

		if i < len(tabs)-1 {
			renderedTabs = append(renderedTabs, TabGap.Render("  "))
		}
	}

	// Join tabs with a horizontal layout
	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

func parseAnimeTitle(fullTitle string) (string, string) {
	// Default values
	title := fullTitle
	episodeCount := "Unknown episodes"

	// Check if the title contains episode information in parentheses
	if idx := strings.LastIndex(fullTitle, " ("); idx != -1 {
		// Extract the title without the episode count
		title = strings.TrimSpace(fullTitle[:idx])

		// Extract episode count text
		episodePart := fullTitle[idx+2:] // Skip the " (" part
		episodeCount = strings.TrimSuffix(episodePart, ")")
	}

	return title, episodeCount
}
