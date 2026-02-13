package views

import (
	"github.com/charmbracelet/bubbletea"

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
	NavigateToTorrentsMsg struct{ AnimeTitle string; Episode int }
	NavigateToPlayerMsg   struct{ MagnetURI string }
	NavigateToLibraryMsg  struct{}
	NavigateToAuthMsg     struct{}
	NavigateBackMsg       struct{}
)

// AppModel is the root model that routes to sub-views.
type AppModel struct {
	currentView    ViewState
	viewHistory    []ViewState
	width          int
	height         int
	config         config.Config
	anilistClient  *anilist.Client
	searchModel    SearchModel
	detailModel    DetailModel
	err            error
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
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.currentView == ViewSearch && !m.searchModel.inputFocused() {
				return m, tea.Quit
			}
		case "esc":
			if m.currentView == ViewSearch {
				if m.searchModel.inputFocused() {
					return m.propagateMsg(msg)
				}
				return m, tea.Quit
			}
			return m.navigateBack()
		case "tab":
			// Toggle between Search and Library (Phase 5)
		}

	case NavigateToDetailMsg:
		m = m.pushView(ViewDetail)
		m.detailModel = NewDetailModel(m.anilistClient, msg.AnimeID)
		return m, m.detailModel.Init()

	case NavigateBackMsg:
		return m.navigateBack()
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
		status = "Search anime  |  / search  |  ? help  |  q quit"
	case ViewDetail:
		content = m.detailModel.View(m.width, contentHeight)
		status = "j/k navigate  |  enter select episode  |  esc back"
	default:
		content = "Not implemented yet"
		status = ""
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

// navigateBack pops the view stack and returns to the previous view.
func (m AppModel) navigateBack() (tea.Model, tea.Cmd) {
	if len(m.viewHistory) == 0 {
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
	}
	return m, nil
}
