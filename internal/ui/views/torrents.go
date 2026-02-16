package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/nyaa"
	"github.com/rayanxn/ani-tui/internal/ui"
)

const maxRetries = 3

// NyaaResultsMsg carries nyaa search results back to the torrents view.
type NyaaResultsMsg struct {
	Results []nyaa.Item
	Query   string
	Err     error
}

// NyaaRetryMsg signals that a transient failure occurred and a retry is needed.
type NyaaRetryMsg struct {
	Query   string
	Attempt int
	Err     error
}

// TorrentListItem wraps a nyaa.Item for bubbles/list rendering.
type TorrentListItem struct {
	item nyaa.Item
}

func (i TorrentListItem) Title() string       { return i.item.Title }
func (i TorrentListItem) Description() string { return i.item.Summary() }
func (i TorrentListItem) FilterValue() string { return i.item.Title }

// TorrentsModel displays nyaa search results for a selected episode.
type TorrentsModel struct {
	animeTitle string
	episode    int
	quality    string
	query      string
	list       list.Model
	spinner    spinner.Model
	loading    bool
	retrying   bool
	retryCount int
	err        error
}

// NewTorrentsModel creates a torrents results view.
func NewTorrentsModel(animeTitle string, episode int, preferredQuality string) TorrentsModel {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(ui.ColorPrimary).
		BorderLeftForeground(ui.ColorPrimary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(ui.ColorSecondary).
		BorderLeftForeground(ui.ColorPrimary)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "Torrents"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = ui.TitleStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SpinnerStyle

	query := nyaa.BuildSearchQuery(animeTitle, episode, preferredQuality)

	return TorrentsModel{
		animeTitle: animeTitle,
		episode:    episode,
		quality:    preferredQuality,
		query:      query,
		list:       l,
		spinner:    s,
		loading:    true,
	}
}

func (m TorrentsModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		searchNyaaCmd(m.query),
	)
}

func (m TorrentsModel) Update(msg tea.Msg) (TorrentsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-6)
		return m, nil

	case NyaaResultsMsg:
		m.loading = false
		m.retrying = false
		m.retryCount = 0
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.err = nil
		items := make([]list.Item, len(msg.Results))
		for i, r := range msg.Results {
			items[i] = TorrentListItem{item: r}
		}
		m.list.SetItems(items)
		return m, nil

	case NyaaRetryMsg:
		m.retrying = true
		m.retryCount = msg.Attempt
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Tick, retrySearchCmd(msg.Query, msg.Attempt))

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			item, ok := m.list.SelectedItem().(TorrentListItem)
			if !ok {
				return m, nil
			}
			magnetURI := item.item.MagnetURI()
			if magnetURI == "" {
				m.err = fmt.Errorf("selected item is missing an info hash")
				return m, nil
			}
			return m, func() tea.Msg {
				return NavigateToPlayerMsg{MagnetURI: magnetURI}
			}
		case "r":
			if !m.loading {
				m.loading = true
				m.retrying = false
				m.retryCount = 0
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, searchNyaaCmd(m.query))
			}
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the torrents results view.
func (m TorrentsModel) View(width, height int) string {
	header := lipgloss.NewStyle().Padding(1, 2).Render(
		ui.TitleStyle.Render("Episode Search") + "\n" +
			lipgloss.NewStyle().Foreground(ui.ColorSubtle).Render(strings.TrimSpace(m.query)),
	)

	listHeight := height - lipgloss.Height(header)
	if listHeight < 0 {
		listHeight = 0
	}
	m.list.SetSize(width, listHeight)

	var body string
	switch {
	case m.loading && m.retrying:
		body = lipgloss.NewStyle().Padding(1, 2).Render(
			m.spinner.View() + fmt.Sprintf(" Retrying... (attempt %d/%d)", m.retryCount, maxRetries))
	case m.loading:
		body = lipgloss.NewStyle().Padding(1, 2).Render(m.spinner.View() + " Searching nyaa.si...")
	case m.err != nil:
		body = lipgloss.NewStyle().Padding(1, 0).Render(
			ui.RenderError(m.err.Error()) + "\n" +
				ui.HelpStyle.Render("  press r to retry"))
	case len(m.list.Items()) == 0:
		body = ui.HelpStyle.Render("  No torrents found for this episode")
	default:
		body = m.list.View()
	}

	content := header + "\n" + body
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func searchNyaaCmd(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := nyaa.Search(context.Background(), query)
		if err != nil && nyaa.IsTransient(err) {
			return NyaaRetryMsg{Query: query, Attempt: 1, Err: err}
		}
		return NyaaResultsMsg{Results: results, Query: query, Err: err}
	}
}

func retrySearchCmd(query string, attempt int) tea.Cmd {
	return func() tea.Msg {
		// Exponential backoff: 1s, 2s, 4s
		delay := time.Duration(1<<uint(attempt-1)) * time.Second
		time.Sleep(delay)

		results, err := nyaa.Search(context.Background(), query)
		if err != nil && nyaa.IsTransient(err) && attempt < maxRetries {
			return NyaaRetryMsg{Query: query, Attempt: attempt + 1, Err: err}
		}
		return NyaaResultsMsg{Results: results, Query: query, Err: err}
	}
}
