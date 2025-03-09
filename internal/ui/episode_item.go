package ui

import (
	"fmt"
	"strconv"
)

// EpisodeItem represents an episode in the episode list
type EpisodeItem struct {
	Number int
}

func (e EpisodeItem) Title() string {
	return fmt.Sprintf("Episode %d", e.Number)
}

func (e EpisodeItem) Description() string {
	return ""
}

func (e EpisodeItem) FilterValue() string {
	return strconv.Itoa(e.Number)
}
