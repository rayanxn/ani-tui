package views

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/anilist"
	"github.com/rayanxn/ani-tui/internal/config"
	"github.com/rayanxn/ani-tui/internal/ui"
)

// ViewState identifies which view is currently active.
type ViewState int

const (
	ViewSearch ViewState = iota
	ViewDetail
	ViewTorrents
	ViewPlayer
	ViewLibrary
	ViewAuth
)

// Navigation messages emitted by sub-views.
type (
	NavigateToDetailMsg   struct{ AnimeID int }
	NavigateToTorrentsMsg struct {
		AnimeID    int
		AnimeTitle string
		Episode    int
	}
	NavigateToPlayerMsg struct {
		MagnetURI  string
		AnimeID    int
		AnimeTitle string
		Episode    int
	}
	NavigateToLibraryMsg struct{}
	NavigateBackMsg      struct{}
)

type updateProgressMsg struct {
	err error
}

// AppModel is the root model that routes to sub-views.
type AppModel struct {
	currentView   ViewState
	viewHistory   []ViewState
	width         int
	height        int
	config        config.Config
	anilistClient *anilist.Client
	searchModel   SearchModel
	detailModel   DetailModel
	torrentsModel TorrentsModel
	playerModel   PlayerModel
	libraryModel  LibraryModel
	authModel     AuthModel
	showHelp      bool
	err           error
}

// NewAppModel creates the root model with the given config.
func NewAppModel(cfg config.Config) AppModel {
	client := anilist.NewClient(cfg.AniListToken)
	return AppModel{
		currentView:   ViewSearch,
		config:        cfg,
		anilistClient: client,
		searchModel:   NewSearchModel(client),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.searchModel.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m.propagateMsg(msg)

	case tea.KeyMsg:
		// Dismiss help overlay on any key
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			m.cleanup()
			return m, tea.Quit
		case "q":
			if m.currentView == ViewSearch && !m.searchModel.inputFocused() {
				m.cleanup()
				return m, tea.Quit
			}
			if m.currentView == ViewLibrary && m.libraryModel.list.FilterState() != list.Filtering {
				m.cleanup()
				return m, tea.Quit
			}
		case "esc":
			if m.activeViewHasError() {
				return m.propagateMsg(msg)
			}
			if m.currentView == ViewSearch {
				if m.searchModel.inputFocused() {
					return m.propagateMsg(msg)
				}
				m.cleanup()
				return m, tea.Quit
			}
			if m.currentView == ViewLibrary && m.libraryModel.list.FilterState() == list.Filtering {
				return m.propagateMsg(msg)
			}
			if m.currentView == ViewAuth && (m.authModel.step == authVerifying || m.authModel.step == authSaving) {
				return m, nil
			}
			if m.currentView == ViewPlayer {
				m.playerModel.Cleanup()
			}
			return m.navigateBack()

		case "?":
			// Suppress when text inputs are focused
			if m.currentView == ViewSearch && m.searchModel.inputFocused() {
				return m.propagateMsg(msg)
			}
			if m.currentView == ViewAuth {
				return m.propagateMsg(msg)
			}
			if m.currentView == ViewLibrary && m.libraryModel.list.FilterState() == list.Filtering {
				return m.propagateMsg(msg)
			}
			m.showHelp = true
			return m, nil
		case "tab", "shift+tab":
			if m.currentView == ViewSearch && !m.searchModel.inputFocused() && msg.String() == "tab" {
				if m.config.AniListToken == "" {
					m = m.pushView(ViewAuth)
					m.authModel = NewAuthModel()
					return m, m.authModel.Init()
				}
				m = m.pushView(ViewLibrary)
				m.libraryModel = NewLibraryModel(m.anilistClient, m.config.AniListUserID)
				return m, m.libraryModel.Init()
			}
			if m.currentView == ViewLibrary {
				return m.propagateMsg(msg)
			}
		}

	case NavigateToDetailMsg:
		m = m.pushView(ViewDetail)
		m.detailModel = NewDetailModel(m.anilistClient, msg.AnimeID)
		return m, m.detailModel.Init()

	case NavigateToTorrentsMsg:
		m = m.pushView(ViewTorrents)
		m.torrentsModel = NewTorrentsModel(msg.AnimeTitle, msg.AnimeID, msg.Episode, m.config.PreferredQuality)
		return m, m.torrentsModel.Init()

	case NavigateToPlayerMsg:
		m = m.pushView(ViewPlayer)
		m.playerModel = NewPlayerModel(msg.MagnetURI, msg.AnimeTitle, msg.Episode, msg.AnimeID, m.config)
		return m, m.playerModel.Init()

	case NavigateBackMsg:
		return m.navigateBack()

	case NavigateToLibraryMsg:
		if m.config.AniListToken == "" {
			m = m.pushView(ViewAuth)
			m.authModel = NewAuthModel()
			return m, m.authModel.Init()
		}
		m = m.pushView(ViewLibrary)
		m.libraryModel = NewLibraryModel(m.anilistClient, m.config.AniListUserID)
		return m, m.libraryModel.Init()

	case AuthCompleteMsg:
		m.config.AniListToken = msg.Token
		m.config.AniListUserID = msg.UserID
		m.anilistClient = anilist.NewClient(msg.Token)
		// Pop the auth view and push to library
		if len(m.viewHistory) > 0 {
			m.currentView = m.viewHistory[len(m.viewHistory)-1]
			m.viewHistory = m.viewHistory[:len(m.viewHistory)-1]
		}
		m = m.pushView(ViewLibrary)
		m.libraryModel = NewLibraryModel(m.anilistClient, msg.UserID)
		return m, m.libraryModel.Init()

	case playerReadyMsg:
		// If user navigated away while stream was loading, clean up resources
		if m.currentView != ViewPlayer {
			if msg.session != nil {
				msg.session.Close()
			}
			if msg.torrentClient != nil {
				msg.torrentClient.Close()
			}
			return m, nil
		}
		return m.propagateMsg(msg)

	case PlayerDoneMsg:
		// Navigate back from player
		if len(m.viewHistory) > 0 {
			m.currentView = m.viewHistory[len(m.viewHistory)-1]
			m.viewHistory = m.viewHistory[:len(m.viewHistory)-1]
		}
		// Fire progress update if authenticated
		if m.config.AniListToken != "" && msg.AnimeID > 0 {
			return m, updateProgressCmd(m.anilistClient, msg.AnimeID, msg.Episode)
		}
		return m, nil

	case updateProgressMsg:
		// Silent handler — best-effort sync
		return m, nil
	}

	return m.propagateMsg(msg)
}

func (m AppModel) View() string {
	if m.width == 0 {
		return ""
	}

	header := ui.RenderHeader(m.width)

	// Content area = total height - header (1 line) - status bar (1 line)
	contentHeight := m.height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}

	var content string
	var status string

	switch m.currentView {
	case ViewSearch:
		content = m.searchModel.View(m.width, contentHeight)
		status = "/ search  |  tab library  |  ? help  |  q quit"
	case ViewDetail:
		content = m.detailModel.View(m.width, contentHeight)
		status = "j/k navigate  |  enter select  |  ? help  |  esc back"
	case ViewTorrents:
		content = m.torrentsModel.View(m.width, contentHeight)
		status = "j/k navigate  |  enter stream  |  ? help  |  esc back"
	case ViewPlayer:
		content = m.playerModel.View(m.width, contentHeight)
		status = "? help  |  esc back"
	case ViewLibrary:
		content = m.libraryModel.View(m.width, contentHeight)
		status = "tab category  |  enter select  |  ? help  |  esc back"
	case ViewAuth:
		content = m.authModel.View(m.width, contentHeight)
		status = "AniList login  |  esc back"
	default:
		content = "Not implemented yet"
		status = ""
	}

	if m.showHelp {
		content = m.renderHelpOverlay(m.width, contentHeight)
	}

	statusBar := ui.RenderStatusBar(m.width, status)
	return header + "\n" + content + "\n" + statusBar
}

// pushView saves current view and switches to a new one.
func (m AppModel) pushView(next ViewState) AppModel {
	m.viewHistory = append(m.viewHistory, m.currentView)
	m.currentView = next
	return m
}

// renderHelpOverlay returns a centered help box with context-sensitive keybindings.
func (m AppModel) renderHelpOverlay(width, height int) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary)
	descStyle := lipgloss.NewStyle().Foreground(ui.ColorText)

	type binding struct{ key, desc string }
	var bindings []binding

	switch m.currentView {
	case ViewSearch:
		bindings = []binding{
			{"/", "Focus search input"},
			{"enter", "Search / select anime"},
			{"j/k", "Navigate results"},
			{"tab", "Open library"},
			{"esc", "Unfocus input / quit"},
			{"q", "Quit"},
		}
	case ViewDetail:
		bindings = []binding{
			{"j/k", "Navigate episodes"},
			{"g/G", "First / last episode"},
			{"enter", "Search torrents for episode"},
			{"esc", "Go back"},
		}
	case ViewTorrents:
		bindings = []binding{
			{"j/k", "Navigate torrents"},
			{"enter", "Stream selected torrent"},
			{"esc", "Go back"},
		}
	case ViewPlayer:
		bindings = []binding{
			{"esc", "Stop playback and go back"},
		}
	case ViewLibrary:
		bindings = []binding{
			{"tab", "Next category"},
			{"shift+tab", "Previous category"},
			{"/", "Filter list"},
			{"r", "Refresh library"},
			{"enter", "View anime details"},
			{"q", "Quit"},
			{"esc", "Go back"},
		}
	}

	// Always-available bindings
	bindings = append(bindings,
		binding{"?", "Toggle this help"},
		binding{"ctrl+c", "Force quit"},
	)

	var lines []string
	lines = append(lines, ui.TitleStyle.Render("Keybindings"))
	lines = append(lines, "")
	for _, b := range bindings {
		line := keyStyle.Width(12).Render(b.key) + descStyle.Render(b.desc)
		lines = append(lines, line)
	}
	lines = append(lines, "")
	lines = append(lines, ui.HelpStyle.Render("Press any key to dismiss"))

	boxWidth := 42
	if width-4 < boxWidth {
		boxWidth = width - 4
	}
	box := ui.BorderedBoxStyle.Width(boxWidth).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// activeViewHasError reports whether the current view is showing an error.
func (m AppModel) activeViewHasError() bool {
	switch m.currentView {
	case ViewSearch:
		return m.searchModel.err != nil
	case ViewDetail:
		return m.detailModel.err != nil
	case ViewTorrents:
		return m.torrentsModel.err != nil
	case ViewPlayer:
		return m.playerModel.err != nil
	case ViewLibrary:
		return m.libraryModel.err != nil
	}
	return false
}

// cleanup releases resources before quitting.
func (m *AppModel) cleanup() {
	m.playerModel.Cleanup()
}

// navigateBack pops the view stack and returns to the previous view.
func (m AppModel) navigateBack() (tea.Model, tea.Cmd) {
	if len(m.viewHistory) == 0 {
		m.cleanup()
		return m, tea.Quit
	}
	m.currentView = m.viewHistory[len(m.viewHistory)-1]
	m.viewHistory = m.viewHistory[:len(m.viewHistory)-1]
	return m, nil
}

// propagateMsg forwards the message to the current sub-model.
func (m AppModel) propagateMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewSearch:
		sm, cmd := m.searchModel.Update(msg)
		m.searchModel = sm
		return m, cmd
	case ViewDetail:
		dm, cmd := m.detailModel.Update(msg)
		m.detailModel = dm
		return m, cmd
	case ViewTorrents:
		tm, cmd := m.torrentsModel.Update(msg)
		m.torrentsModel = tm
		return m, cmd
	case ViewPlayer:
		pm, cmd := m.playerModel.Update(msg)
		m.playerModel = pm
		return m, cmd
	case ViewLibrary:
		lm, cmd := m.libraryModel.Update(msg)
		m.libraryModel = lm
		return m, cmd
	case ViewAuth:
		am, cmd := m.authModel.Update(msg)
		m.authModel = am
		return m, cmd
	}
	return m, nil
}

func updateProgressCmd(client *anilist.Client, animeID, episode int) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateProgress(context.Background(), animeID, episode, "CURRENT")
		return updateProgressMsg{err: err}
	}
}
