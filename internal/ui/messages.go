package ui

import "github.com/daannte/aniview/internal"

// Custom tea.Msg types for the UI

// AnimeListsMsg contains the anime lists data from the API
type AnimeListsMsg struct {
	Watching []internal.AnimeEntry
	Planned  []internal.AnimeEntry
}

// ErrMsg represents an error message
type ErrMsg struct {
	Err error
}

// Error implements the error interface
func (e ErrMsg) Error() string {
	return e.Err.Error()
}

// EpisodePlayedMsg represents the result of playing an episode
type EpisodePlayedMsg struct {
	Err error
}

// StatusChangeMsg represents a confirmation for status change
type StatusChangeMsg struct {
	Confirmed bool
}
