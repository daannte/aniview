package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daannte/aniview/internal"
)

// UIState represents the possible states of the UI
type UIState string

const (
	StateLoading    UIState = "loading"
	StateSelecting  UIState = "selecting"
	StateEpisode    UIState = "episode"
	StateDetails    UIState = "details"
	StateError      UIState = "error"
	StateConfirming UIState = "confirming"
)

// Model represents the UI state
type Model struct {
	Config           *internal.Config
	Anilist          *internal.AniListClient
	AnimeList        list.Model
	PlannedList      list.Model
	EpisodeList      list.Model
	AnimeEntries     []internal.AnimeEntry
	PlannedEntries   []internal.AnimeEntry
	Loading          bool
	Spinner          spinner.Model
	Err              error
	SelectedAnime    *AnimeItem
	State            UIState
	ActiveTab        int // 0 = watching, 1 = planned
	Tabs             []string
	ConfirmingStatus bool
	Viewport         viewport.Model
}

// NewModel creates a new UI model
func NewModel(config *internal.Config, anilist *internal.AniListClient) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create anime list
	animeDelegate := list.NewDefaultDelegate()
	animeDelegate.Styles.SelectedTitle = animeDelegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#04B575"))
	animeDelegate.Styles.SelectedDesc = animeDelegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#04B575"))

	animeList := list.New([]list.Item{}, animeDelegate, 0, 0)
	animeList.SetShowStatusBar(false)
	animeList.SetFilteringEnabled(true)
	animeList.SetShowTitle(false)

	// Create planned list
	plannedList := list.New([]list.Item{}, animeDelegate, 0, 0)
	plannedList.SetShowStatusBar(false)
	plannedList.SetFilteringEnabled(true)
	plannedList.SetShowTitle(false)

	// Create episode list with compact delegate
	episodeList := list.New([]list.Item{}, NewCompactDelegate(), 0, 0)
	episodeList.Title = "Select Episode"
	episodeList.SetShowStatusBar(false)
	episodeList.SetFilteringEnabled(true)
	episodeList.Styles.Title = TitleStyle

	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().Padding(1, 2)

	return &Model{
		Config:      config,
		Anilist:     anilist,
		AnimeList:   animeList,
		PlannedList: plannedList,
		EpisodeList: episodeList,
		Spinner:     s,
		Loading:     true,
		State:       StateLoading,
		ActiveTab:   0,
		Tabs:        []string{"Currently Watching", "Planned"},
		Viewport:    vp,
	}
}

// InitAnimeLists initializes both anime lists
func (m *Model) InitAnimeLists() tea.Cmd {
	return func() tea.Msg {
		// Get currently watching anime
		animeEntries, err := m.Anilist.GetCurrentlyWatching(m.Config.UserID)
		if err != nil {
			return ErrMsg{Err: err}
		}

		// Get planned anime
		plannedEntries, err := m.Anilist.GetPlanned(m.Config.UserID)
		if err != nil {
			return ErrMsg{Err: err}
		}

		return AnimeListsMsg{
			Watching: animeEntries,
			Planned:  plannedEntries,
		}
	}
}

// Init initializes the UI
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.Spinner.Tick,
		m.InitAnimeLists(),
	)
}

// PromptStatusChange prompts the user to confirm a status change
func (m *Model) PromptStatusChange() tea.Cmd {
	m.State = StateConfirming
	m.ConfirmingStatus = true

	return func() tea.Msg {
		// This would normally show a prompt, but we'll handle the UI in the View method
		// Just return a placeholder message to force an update
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}}
	}
}

// StartPlayEpisode starts playing the selected episode
func (m *Model) StartPlayEpisode() tea.Cmd {
	return func() tea.Msg {
		// Get the selected episode
		epItem, ok := m.EpisodeList.SelectedItem().(EpisodeItem)
		if !ok {
			return EpisodePlayedMsg{Err: fmt.Errorf("failed to get selected episode")}
		}

		epNum := epItem.Number
		animeTitle := m.SelectedAnime.AnimeEntry.Title
		animeResults, err := internal.SearchAnime(animeTitle, "sub") // Use "sub" as the default mode
		if err != nil {
			return EpisodePlayedMsg{Err: fmt.Errorf("failed to search anime: %v", err)}
		}

		if len(animeResults) == 0 {
			return EpisodePlayedMsg{Err: fmt.Errorf("no anime found with title: %s", animeTitle)}
		}

		// Get the first result's ID
		var animeID string
		for id := range animeResults {
			animeID = id
			break
		}

		// Get the episode URL
		links, err := internal.GetEpisodeURL(animeID, epNum)
		if err != nil {
			return EpisodePlayedMsg{Err: fmt.Errorf("failed to get episode URL: %v", err)}
		}

		// Update the current episode
		m.SelectedAnime.AnimeEntry.CurrentEpisode = epNum
		internal.GetEpisodeData(m.SelectedAnime.AnimeEntry.MalId, epNum, &m.SelectedAnime.AnimeEntry)

		// Play the episode
		err = internal.PlayEpisode(links, m.SelectedAnime.AnimeEntry)
		return EpisodePlayedMsg{Err: err}
	}
}

// Update updates the UI state
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := msg.Width-4, msg.Height-6 // Leave some margin plus space for tabs
		m.AnimeList.SetSize(h, v)
		m.PlannedList.SetSize(h, v)
		m.EpisodeList.SetSize(h, v)

		m.Viewport.Width = h
		m.Viewport.Height = v
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case AnimeListsMsg:
		m.AnimeEntries = msg.Watching
		m.PlannedEntries = msg.Planned

		// Create items for watching list
		watchingItems := make([]list.Item, len(m.AnimeEntries))
		for i, entry := range m.AnimeEntries {
			watchingItems[i] = AnimeItem{AnimeEntry: entry, Index: i}
		}
		m.AnimeList.SetItems(watchingItems)

		// Create items for planned list
		plannedItems := make([]list.Item, len(m.PlannedEntries))
		for i, entry := range m.PlannedEntries {
			plannedItems[i] = AnimeItem{AnimeEntry: entry, Index: i}
		}
		m.PlannedList.SetItems(plannedItems)

		m.Loading = false
		m.State = StateSelecting
		return m, nil

	case ErrMsg:
		m.Err = msg.Err
		m.Loading = false
		m.State = StateError
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case EpisodePlayedMsg:
		return m.handleEpisodePlayed(msg)

	case StatusChangeMsg:
		return m.handleStatusChange(msg)
	}

	// Handle state-specific updates
	switch m.State {
	case StateSelecting:
		var cmd tea.Cmd
		if m.ActiveTab == 0 {
			m.AnimeList, cmd = m.AnimeList.Update(msg)
		} else {
			m.PlannedList, cmd = m.PlannedList.Update(msg)
		}
		return m, cmd

	case StateEpisode:
		var cmd tea.Cmd
		m.EpisodeList, cmd = m.EpisodeList.Update(msg)
		return m, cmd

	case StateDetails:
		var cmd tea.Cmd
		m.Viewport, cmd = m.Viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.State == StateDetails {
			// Return to selection screen from details view
			m.State = StateSelecting
			return m, nil
		}
		return m, tea.Quit

	case "i", "I":
		if m.State == StateSelecting {
			var selectedItem AnimeItem
			var ok bool

			if m.ActiveTab == 0 {
				selectedItem, ok = m.AnimeList.SelectedItem().(AnimeItem)
			} else {
				selectedItem, ok = m.PlannedList.SelectedItem().(AnimeItem)
			}

			if ok {
				m.Viewport.SetContent(selectedItem.DetailedView())
				m.Viewport.GotoTop()
				m.State = StateDetails
				return m, nil
			}
		}

	case "esc":
		switch m.State {
		case StateDetails, StateEpisode:
			m.State = StateSelecting
			return m, nil
		}

	case "tab", "right":
		if m.State == StateSelecting {
			m.ActiveTab = (m.ActiveTab + 1) % len(m.Tabs)
			return m, nil
		}

	case "shift+tab", "left":
		if m.State == StateSelecting {
			m.ActiveTab = (m.ActiveTab - 1 + len(m.Tabs)) % len(m.Tabs)
			return m, nil
		}

	case "y":
		if m.State == StateConfirming {
			return m, func() tea.Msg { return StatusChangeMsg{Confirmed: true} }
		}

	case "n":
		if m.State == StateConfirming {
			return m, func() tea.Msg { return StatusChangeMsg{Confirmed: false} }
		}

	case "enter":
		switch m.State {
		case StateSelecting:
			var selectedItem AnimeItem
			var ok bool
			if m.ActiveTab == 0 {
				selectedItem, ok = m.AnimeList.SelectedItem().(AnimeItem)
			} else {
				selectedItem, ok = m.PlannedList.SelectedItem().(AnimeItem)
			}
			if ok {
				m.SelectedAnime = &selectedItem
				m.State = StateEpisode
				episodeCount := selectedItem.AnimeEntry.Episodes
				items := make([]list.Item, episodeCount)
				for i := 0; i < episodeCount; i++ {
					items[i] = EpisodeItem{Number: i + 1}
				}
				m.EpisodeList.SetItems(items)
				// Select the next episode by default
				nextEp := selectedItem.AnimeEntry.Progress + 1
				if nextEp >= 0 && nextEp <= len(items) {
					m.EpisodeList.Select(nextEp - 1)
				}
				return m, nil
			}
		case StateEpisode:
			m.State = StateLoading
			return m, m.StartPlayEpisode()
		case StateDetails:
			m.State = StateSelecting
			return m, nil
		}
	}

	// Pass key events to the appropriate list based on the current state
	var cmd tea.Cmd

	switch m.State {
	case StateSelecting:
		if m.ActiveTab == 0 {
			m.AnimeList, cmd = m.AnimeList.Update(msg)
		} else {
			m.PlannedList, cmd = m.PlannedList.Update(msg)
		}
	case StateEpisode:
		m.EpisodeList, cmd = m.EpisodeList.Update(msg)
	case StateDetails:
		m.Viewport, cmd = m.Viewport.Update(msg)
	}

	return m, cmd
}

// handleEpisodePlayed handles the result of playing an episode
func (m *Model) handleEpisodePlayed(msg EpisodePlayedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.Err = msg.Err
		m.State = StateError
	} else {
		// Get the selected episode number
		epItem, ok := m.EpisodeList.SelectedItem().(EpisodeItem)
		if ok {
			// Update progress in AniList
			_ = m.Anilist.UpdateProgress(m.SelectedAnime.AnimeEntry.ID, epItem.Number)

			// Update local progress if the watched episode is the next one
			if epItem.Number == m.SelectedAnime.AnimeEntry.Progress+1 {
				if m.ActiveTab == 0 {
					m.AnimeEntries[m.SelectedAnime.Index].Progress = epItem.Number
					m.SelectedAnime.AnimeEntry.Progress = epItem.Number

					// Update the list items to reflect the progress change
					items := make([]list.Item, len(m.AnimeEntries))
					for i, entry := range m.AnimeEntries {
						items[i] = AnimeItem{AnimeEntry: entry, Index: i}
					}
					m.AnimeList.SetItems(items)
				} else {
					m.PlannedEntries[m.SelectedAnime.Index].Progress = epItem.Number
					m.SelectedAnime.AnimeEntry.Progress = epItem.Number

					// Update the list items to reflect the progress change
					items := make([]list.Item, len(m.PlannedEntries))
					for i, entry := range m.PlannedEntries {
						items[i] = AnimeItem{AnimeEntry: entry, Index: i}
					}
					m.PlannedList.SetItems(items)
				}
			}
			if m.ActiveTab == 1 {
				// Create a confirmation model and prompt the user
				return m, m.PromptStatusChange()
			}
		}

		// Return to selection screen
		m.State = StateSelecting
	}
	return m, nil
}

// handleStatusChange handles the user's response to a status change prompt
func (m *Model) handleStatusChange(msg StatusChangeMsg) (tea.Model, tea.Cmd) {
	if msg.Confirmed {
		// User confirmed, update the anime status to CURRENT
		err := m.Anilist.UpdateAnime(m.SelectedAnime.AnimeEntry.ID, m.SelectedAnime.AnimeEntry.Progress, "CURRENT")
		if err != nil {
			m.Err = err
			m.State = StateError
			return m, nil
		}

		// We should refresh the lists since we've moved an item from planned to current
		m.State = StateLoading
		m.Loading = true
		return m, m.InitAnimeLists()
	}

	// User declined or operation complete, return to selection
	m.State = StateSelecting
	return m, nil
}

// View renders the UI
func (m *Model) View() string {
	if m.Loading {
		return fmt.Sprintf("\n\n   %s Loading anime list...\n\n", m.Spinner.View())
	}

	if m.Err != nil {
		return fmt.Sprintf("\n\n   %s\n\n", ErrorStyle.Render(m.Err.Error()))
	}

	switch m.State {
	case StateSelecting:
		var b strings.Builder
		// Render tabs
		b.WriteString(fmt.Sprintf("\n   %s\n\n", RenderTabs(m.Tabs, m.ActiveTab)))

		// Render appropriate list
		if m.ActiveTab == 0 {
			b.WriteString(m.AnimeList.View())
		} else {
			b.WriteString(m.PlannedList.View())
		}
		return b.String()

	case StateDetails:
		var b strings.Builder
		b.WriteString("\n   " + TitleStyle.Render("Anime Details") + "\n\n")
		b.WriteString(m.Viewport.View())
		return b.String()

	case StateEpisode:
		var b strings.Builder
		b.WriteString(fmt.Sprintf("\n   %s\n\n", TitleStyle.Render(m.SelectedAnime.AnimeEntry.Title)))

		// Show progress
		progress := m.SelectedAnime.AnimeEntry.Progress
		episodes := m.SelectedAnime.AnimeEntry.Episodes
		if episodes > 0 {
			b.WriteString(fmt.Sprintf("   Progress: %d/%d episodes\n\n", progress, episodes))
		} else {
			b.WriteString(fmt.Sprintf("   Progress: %d episodes watched\n\n", progress))
		}

		// Show the episode list
		b.WriteString(m.EpisodeList.View())
		b.WriteString("\n\n   Press Enter to watch, Esc to go back\n")
		return b.String()

	case StateLoading:
		return fmt.Sprintf("\n\n   %s Loading episode...\n\n", m.Spinner.View())

	case StateConfirming:
		var b strings.Builder
		b.WriteString(fmt.Sprintf("\n\n   %s\n\n", TitleStyle.Render("Move to Currently Watching?")))
		b.WriteString(fmt.Sprintf("   Do you want to move '%s' to your Currently Watching list?\n\n", m.SelectedAnime.AnimeEntry.Title))
		b.WriteString("   Press [y] to confirm, [n] to keep in Planned\n")
		return b.String()
	}

	return "Something went wrong"
}
