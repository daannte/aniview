package internal

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))
)

// AnimeItem represents an item in the list UI
type AnimeItem struct {
	animeEntry AnimeEntry
	index      int // Store the original index in the list
}

func (i AnimeItem) Title() string {
	return i.animeEntry.Title
}

func (i AnimeItem) Description() string {
	var status string

	// Check if the anime is currently airing
	if i.animeEntry.IsAiring {
		status = fmt.Sprintf("%d/%d episodes (Currently airing)", i.animeEntry.Progress, i.animeEntry.Episodes)
	} else if i.animeEntry.Episodes > 0 {
		status = fmt.Sprintf("%d/%d episodes", i.animeEntry.Progress, i.animeEntry.Episodes)
	}
	return status
}

func (i AnimeItem) FilterValue() string {
	return i.animeEntry.Title
}

// Custom delegate for episode items to make them more compact
type compactDelegate struct {
	styles  list.DefaultItemStyles
	height  int
	spacing int
}

// Height returns the height of the item
func (d compactDelegate) Height() int {
	return d.height
}

// Spacing returns the spacing between items
func (d compactDelegate) Spacing() int {
	return d.spacing
}

// Update is called when a message is received
func (d compactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders the item
func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string

	if i, ok := item.(EpisodeItem); ok {
		title = i.Title()
	} else {
		title = "Unknown item"
	}

	if selected := index == m.Index(); selected {
		fmt.Fprint(w, selectedStyle.Render("> "+title))
	} else {
		fmt.Fprint(w, "  "+title)
	}
}

// EpisodeItem represents an episode in the episode list
type EpisodeItem struct {
	number int
}

func (e EpisodeItem) Title() string {
	return fmt.Sprintf("Episode %d", e.number)
}

func (e EpisodeItem) Description() string {
	return ""
}

func (e EpisodeItem) FilterValue() string {
	return strconv.Itoa(e.number)
}

// NewCompactDelegate creates a new compact delegate for episode items
func NewCompactDelegate() list.ItemDelegate {
	styles := list.NewDefaultItemStyles()
	styles.SelectedTitle = styles.SelectedTitle.Foreground(lipgloss.Color("#04B575"))

	return &compactDelegate{
		styles:  styles,
		height:  1, // Single line height
		spacing: 0, // No spacing between items
	}
}

// Model represents the UI state
type Model struct {
	config        *Config
	animeList     list.Model
	episodeList   list.Model
	animeEntries  []AnimeEntry
	loading       bool
	spinner       spinner.Model
	err           error
	selectedAnime *AnimeItem
	state         string // "selecting", "episode", "loading", "error"
}

func NewModel(config *Config) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create anime list
	animeDelegate := list.NewDefaultDelegate()
	animeDelegate.Styles.SelectedTitle = animeDelegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#04B575"))
	animeDelegate.Styles.SelectedDesc = animeDelegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#04B575"))

	animeList := list.New([]list.Item{}, animeDelegate, 0, 0)
	animeList.Title = "Currently Watching"
	animeList.SetShowStatusBar(false)
	animeList.SetFilteringEnabled(true)
	animeList.Styles.Title = titleStyle

	// Create episode list with compact delegate
	episodeList := list.New([]list.Item{}, NewCompactDelegate(), 0, 0)
	episodeList.Title = "Select Episode"
	episodeList.SetShowStatusBar(false)
	episodeList.SetFilteringEnabled(true)
	episodeList.Styles.Title = titleStyle

	return &Model{
		config:      config,
		animeList:   animeList,
		episodeList: episodeList,
		spinner:     s,
		loading:     true,
		state:       "loading",
	}
}

// InitAnimeList initializes the anime list
func (m *Model) InitAnimeList() tea.Cmd {
	return func() tea.Msg {
		animeEntries, err := GetCurrentlyWatching(m.config)
		if err != nil {
			return errMsg{err}
		}
		return animeListMsg{animeEntries}
	}
}

// Init initializes the UI
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.InitAnimeList(),
	)
}

// Update updates the UI state
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := msg.Width-4, msg.Height-4 // Leave some margin
		m.animeList.SetSize(h, v)
		m.episodeList.SetSize(h, v)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			switch m.state {
			case "selecting":
				item, ok := m.animeList.SelectedItem().(AnimeItem)
				if ok {
					m.selectedAnime = &item
					m.state = "episode"

					episodeCount := item.animeEntry.Episodes

					items := make([]list.Item, episodeCount)
					for i := 0; i < episodeCount; i++ {
						items[i] = EpisodeItem{number: i + 1}
					}

					m.episodeList.SetItems(items)

					// Select the next episode by default
					nextEp := item.animeEntry.Progress + 1
					if nextEp >= 0 && nextEp < len(items) {
						m.episodeList.Select(nextEp - 1)
					}

					return m, nil
				}
			case "episode":
				m.state = "loading"
				return m, m.startPlayEpisode()
			}
		case "esc":
			if m.state == "episode" {
				m.state = "selecting"
				return m, nil
			}
		}

	case animeListMsg:
		m.animeEntries = msg.entries
		items := make([]list.Item, len(m.animeEntries))
		for i, entry := range m.animeEntries {
			// Store the index so we can update the correct entry later
			items[i] = AnimeItem{animeEntry: entry, index: i}
		}

		// Update the list with the new items
		m.animeList.SetItems(items)
		m.loading = false
		m.state = "selecting"
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.state = "error"
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case episodePlayedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = "error"
		} else {
			// Get the selected episode number
			epItem, ok := m.episodeList.SelectedItem().(EpisodeItem)
			if ok {
				// Update progress in AniList
				_ = UpdateProgress(m.config, m.selectedAnime.animeEntry.ID, epItem.number)

				// Update local progress if the watched episode is the next one
				if epItem.number == m.selectedAnime.animeEntry.Progress+1 {
					m.animeEntries[m.selectedAnime.index].Progress = epItem.number
					m.selectedAnime.animeEntry.Progress = epItem.number

					// Update the list items to reflect the progress change
					items := make([]list.Item, len(m.animeEntries))
					for i, entry := range m.animeEntries {
						items[i] = AnimeItem{animeEntry: entry, index: i}
					}
					m.animeList.SetItems(items)
				}
			}

			// Return to selection screen
			m.state = "selecting"
		}
		return m, nil
	}

	switch m.state {
	case "selecting":
		var cmd tea.Cmd
		m.animeList, cmd = m.animeList.Update(msg)
		return m, cmd
	case "episode":
		var cmd tea.Cmd
		m.episodeList, cmd = m.episodeList.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the UI
func (m *Model) View() string {
	if m.loading {
		return fmt.Sprintf("\n\n   %s Loading anime list...\n\n", m.spinner.View())
	}

	if m.err != nil {
		return fmt.Sprintf("\n\n   %s\n\n", errorStyle.Render(m.err.Error()))
	}

	switch m.state {
	case "selecting":
		return m.animeList.View()
	case "episode":
		var b strings.Builder
		b.WriteString(fmt.Sprintf("\n   %s\n\n", titleStyle.Render(m.selectedAnime.animeEntry.Title)))

		// Show progress
		progress := m.selectedAnime.animeEntry.Progress
		episodes := m.selectedAnime.animeEntry.Episodes
		if episodes > 0 {
			b.WriteString(fmt.Sprintf("   Progress: %d/%d episodes\n\n", progress, episodes))
		} else {
			b.WriteString(fmt.Sprintf("   Progress: %d episodes watched\n\n", progress))
		}

		// Show the episode list
		b.WriteString(m.episodeList.View())
		b.WriteString("\n\n   Press Enter to watch, Esc to go back\n")
		return b.String()
	case "loading":
		return fmt.Sprintf("\n\n   %s Loading episode...\n\n", m.spinner.View())
	}

	return "Something went wrong"
}

// startPlayEpisode starts playing the selected episode
func (m *Model) startPlayEpisode() tea.Cmd {
	return func() tea.Msg {
		// Get the selected episode
		epItem, ok := m.episodeList.SelectedItem().(EpisodeItem)
		if !ok {
			return episodePlayedMsg{err: fmt.Errorf("failed to get selected episode")}
		}

		epNum := epItem.number
		animeTitle := m.selectedAnime.animeEntry.Title
		animeResults, err := SearchAnime(animeTitle, "sub") // Use "sub" as the default mode
		if err != nil {
			return episodePlayedMsg{err: fmt.Errorf("failed to search anime: %v", err)}
		}

		if len(animeResults) == 0 {
			return episodePlayedMsg{err: fmt.Errorf("no anime found with title: %s", animeTitle)}
		}

		// Get the first result's ID
		var animeID string
		for id := range animeResults {
			animeID = id
			break
		}

		// Get the episode URL
		links, err := GetEpisodeURL(animeID, epNum)
		if err != nil {
			return episodePlayedMsg{err: fmt.Errorf("failed to get episode URL: %v", err)}
		}

		// Update the current episode
		m.selectedAnime.animeEntry.CurrentEpisode = epNum
		GetEpisodeData(m.selectedAnime.animeEntry.MalId, epNum, &m.selectedAnime.animeEntry)

		// Play the episode
		err = PlayEpisode(links, m.selectedAnime.animeEntry)
		return episodePlayedMsg{err: err}
	}
}

// Messages
type animeListMsg struct {
	entries []AnimeEntry
}

type errMsg struct {
	err error
}

// Error implements the error interface
func (e errMsg) Error() string {
	return e.err.Error()
}

type episodePlayedMsg struct {
	err error
}

// StartUI starts the UI
func StartUI(config *Config) error {
	m := NewModel(config)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
