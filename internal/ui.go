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

	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A8A8A8")).
			Padding(0, 1).
			Align(lipgloss.Center).
			Border(lipgloss.NormalBorder(), false, false, false, false) // No bottom border
	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Align(lipgloss.Center).
			Bold(true) // Bold text for active tab

	tabGap = lipgloss.NewStyle().Width(1)
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

// DetailedView returns a detailed view of the anime with title, description, and airing info
func (i AnimeItem) DetailedView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(i.animeEntry.Title))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Progress: %d/%d episodes\n", i.animeEntry.Progress, i.animeEntry.Episodes))

	if i.animeEntry.IsAiring {
		b.WriteString("Status: Currently Airing\n")
	} else {
		b.WriteString("Status: Completed\n")
	}

	if i.animeEntry.IsAiring {
		timeUntil := formatTimeUntil(i.animeEntry.NextAiringEpisode.TimeUntilAiring)
		b.WriteString(fmt.Sprintf("\nNext Episode: Episode %d\n", i.animeEntry.NextAiringEpisode.Episode))
		b.WriteString(fmt.Sprintf("Airing: %s\n", timeUntil))
	}

	// Add a separator before description
	b.WriteString("\nDescription:\n")

	// Format and add the description with word wrapping
	if i.animeEntry.Description != "" {
		// Simple word wrapping for description (80 chars per line)
		wrappedDesc := wordWrap(i.animeEntry.Description, 80)
		b.WriteString(wrappedDesc)
	} else {
		b.WriteString("No description available.")
	}

	return b.String()
}

// Helper function to format time until airing in a human-readable format
func formatTimeUntil(seconds int) string {
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

// Simple word wrapping function
func wordWrap(text string, lineWidth int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	lineLength := 0

	for _, word := range words {
		// Handle HTML tags by removing them for display
		word = strings.ReplaceAll(word, "<br>", "\n")
		word = strings.ReplaceAll(word, "<i>", "")
		word = strings.ReplaceAll(word, "</i>", "")

		if lineLength+len(word)+1 > lineWidth {
			result.WriteString("\n")
			lineLength = 0
		} else if lineLength > 0 {
			result.WriteString(" ")
			lineLength++
		}

		result.WriteString(word)
		lineLength += len(word)
	}

	return result.String()
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

type statusChangeMsg struct {
	confirmed bool
}

// Model represents the UI state
type Model struct {
	config           *Config
	animeList        list.Model
	plannedList      list.Model
	episodeList      list.Model
	animeEntries     []AnimeEntry
	plannedEntries   []AnimeEntry
	loading          bool
	spinner          spinner.Model
	err              error
	selectedAnime    *AnimeItem
	state            string // "selecting", "episode", "loading", "error"
	activeTab        int    // 0 = watching, 1 = planned
	tabs             []string
	confirmingStatus bool
	detailsContent   string
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
	episodeList.Styles.Title = titleStyle

	return &Model{
		config:      config,
		animeList:   animeList,
		plannedList: plannedList,
		episodeList: episodeList,
		spinner:     s,
		loading:     true,
		state:       "loading",
		activeTab:   0,
		tabs:        []string{"Currently Watching", "Planned"},
	}
}

// InitAnimeList initializes both anime lists
func (m *Model) InitAnimeLists() tea.Cmd {
	return func() tea.Msg {
		// Get currently watching anime
		animeEntries, err := GetCurrentlyWatching(m.config)
		if err != nil {
			return errMsg{err}
		}

		// Get planned anime
		plannedEntries, err := GetPlanned(m.config)
		if err != nil {
			return errMsg{err}
		}

		return animeListsMsg{
			watching: animeEntries,
			planned:  plannedEntries,
		}
	}
}

// Init initializes the UI
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.InitAnimeLists(),
	)
}

// Update updates the UI state
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := msg.Width-4, msg.Height-6 // Leave some margin plus space for tabs
		m.animeList.SetSize(h, v)
		m.plannedList.SetSize(h, v)
		m.episodeList.SetSize(h, v)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == "details" {
				// Return to selection screen from details view
				m.state = "selecting"
				return m, nil
			}
			return m, tea.Quit
		case "i", "I":
			if m.state == "selecting" {
				var selectedItem AnimeItem
				var ok bool

				if m.activeTab == 0 {
					selectedItem, ok = m.animeList.SelectedItem().(AnimeItem)
				} else {
					selectedItem, ok = m.plannedList.SelectedItem().(AnimeItem)
				}

				if ok {
					m.detailsContent = selectedItem.DetailedView()
					m.state = "details"
					return m, nil
				}
			}
		case "esc":
			switch m.state {
			case "details":
				m.state = "selecting"
				return m, nil
			case "episode":
				m.state = "selecting"
				return m, nil
			}
		case "tab", "right", "l":
			if m.state == "selecting" {
				m.activeTab = (m.activeTab + 1) % len(m.tabs)
				return m, nil
			}
		case "shift+tab", "left", "h":
			if m.state == "selecting" {
				m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
				return m, nil
			}
		case "y":
			if m.state == "confirming" {
				return m, func() tea.Msg { return statusChangeMsg{confirmed: true} }
			}
		case "n":
			if m.state == "confirming" {
				return m, func() tea.Msg { return statusChangeMsg{confirmed: false} }
			}
		case "enter":
			switch m.state {
			case "selecting":
				var selectedItem AnimeItem
				var ok bool
				if m.activeTab == 0 {
					selectedItem, ok = m.animeList.SelectedItem().(AnimeItem)
				} else {
					selectedItem, ok = m.plannedList.SelectedItem().(AnimeItem)
				}
				if ok {
					m.selectedAnime = &selectedItem
					m.state = "episode"
					episodeCount := selectedItem.animeEntry.Episodes
					items := make([]list.Item, episodeCount)
					for i := 0; i < episodeCount; i++ {
						items[i] = EpisodeItem{number: i + 1}
					}
					m.episodeList.SetItems(items)
					// Select the next episode by default
					nextEp := selectedItem.animeEntry.Progress + 1
					if nextEp >= 0 && nextEp <= len(items) {
						m.episodeList.Select(nextEp - 1)
					}
					return m, nil
				}
			case "episode":
				m.state = "loading"
				return m, m.startPlayEpisode()
			case "details":
				m.state = "selecting"
				return m, nil
			}
		}

	case animeListsMsg:
		m.animeEntries = msg.watching
		m.plannedEntries = msg.planned

		// Create items for watching list
		watchingItems := make([]list.Item, len(m.animeEntries))
		for i, entry := range m.animeEntries {
			watchingItems[i] = AnimeItem{animeEntry: entry, index: i}
		}
		m.animeList.SetItems(watchingItems)

		// Create items for planned list
		plannedItems := make([]list.Item, len(m.plannedEntries))
		for i, entry := range m.plannedEntries {
			plannedItems[i] = AnimeItem{animeEntry: entry, index: i}
		}
		m.plannedList.SetItems(plannedItems)

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
					if m.activeTab == 0 {
						m.animeEntries[m.selectedAnime.index].Progress = epItem.number
						m.selectedAnime.animeEntry.Progress = epItem.number

						// Update the list items to reflect the progress change
						items := make([]list.Item, len(m.animeEntries))
						for i, entry := range m.animeEntries {
							items[i] = AnimeItem{animeEntry: entry, index: i}
						}
						m.animeList.SetItems(items)
					} else {
						m.plannedEntries[m.selectedAnime.index].Progress = epItem.number
						m.selectedAnime.animeEntry.Progress = epItem.number

						// Update the list items to reflect the progress change
						items := make([]list.Item, len(m.plannedEntries))
						for i, entry := range m.plannedEntries {
							items[i] = AnimeItem{animeEntry: entry, index: i}
						}
						m.plannedList.SetItems(items)
					}
				}
				if m.activeTab == 1 {
					// Create a confirmation model and prompt the user
					return m, m.promptStatusChange()
				}
			}

			// Return to selection screen
			m.state = "selecting"
		}
		return m, nil
	case statusChangeMsg:
		if msg.confirmed {
			// User confirmed, update the anime status to CURRENT
			err := UpdateAnime(m.config, m.selectedAnime.animeEntry.ID, m.selectedAnime.animeEntry.Progress, "CURRENT")
			if err != nil {
				m.err = err
				m.state = "error"
				return m, nil
			}

			// We should refresh the lists since we've moved an item from planned to current
			m.state = "loading"
			m.loading = true
			return m, m.InitAnimeLists()
		}

		// User declined or operation complete, return to selection
		m.state = "selecting"
		return m, nil
	}

	switch m.state {
	case "selecting":
		var cmd tea.Cmd
		if m.activeTab == 0 {
			m.animeList, cmd = m.animeList.Update(msg)
		} else {
			m.plannedList, cmd = m.plannedList.Update(msg)
		}
		return m, cmd
	case "episode":
		var cmd tea.Cmd
		m.episodeList, cmd = m.episodeList.Update(msg)
		return m, cmd
	}

	return m, nil
}

// renderTabs renders the tab bar
func (m *Model) renderTabs() string {
	var renderedTabs []string

	for i, tab := range m.tabs {
		if i == m.activeTab {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, tabStyle.Render(tab))
		}

		if i < len(m.tabs)-1 {
			renderedTabs = append(renderedTabs, tabGap.Render("  "))
		}
	}

	// Join tabs with a horizontal layout
	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

func (m *Model) promptStatusChange() tea.Cmd {
	m.state = "confirming"
	m.confirmingStatus = true

	return func() tea.Msg {
		// This would normally show a prompt, but we'll handle the UI in the View method
		// Just return a placeholder message to force an update
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}}
	}
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
		var b strings.Builder
		// Render tabs
		b.WriteString(fmt.Sprintf("\n   %s\n\n", m.renderTabs()))

		// Render appropriate list
		if m.activeTab == 0 {
			b.WriteString(m.animeList.View())
		} else {
			b.WriteString(m.plannedList.View())
		}
		return b.String()

	case "details":
		var b strings.Builder
		b.WriteString("\n   ")
		b.WriteString(m.detailsContent)
		return b.String()

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

	case "confirming":
		var b strings.Builder
		b.WriteString(fmt.Sprintf("\n\n   %s\n\n", titleStyle.Render("Move to Currently Watching?")))
		b.WriteString(fmt.Sprintf("   Do you want to move '%s' to your Currently Watching list?\n\n", m.selectedAnime.animeEntry.Title))
		b.WriteString("   Press [y] to confirm, [n] to keep in Planned\n")
		return b.String()
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
type animeListsMsg struct {
	watching []AnimeEntry
	planned  []AnimeEntry
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
