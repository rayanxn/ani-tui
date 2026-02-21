package views

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/nyaa"
	"github.com/rayanxn/ani-tui/internal/ui"
)

// NyaaResultsMsg carries nyaa search results back to the torrents view.
type NyaaResultsMsg struct {
	Results []nyaa.Item
	Err     error
}

// TorrentListItem wraps a nyaa.Item for bubbles/list rendering.
type TorrentListItem struct {
	item nyaa.Item
}

func (i TorrentListItem) Title() string {
	if i.item.IsTrusted() {
		return "✓ " + i.item.Title
	}
	return i.item.Title
}
func (i TorrentListItem) Description() string { return i.item.Summary() }
func (i TorrentListItem) FilterValue() string { return i.item.Title }

// torrentDelegate wraps DefaultDelegate to highlight trusted torrents.
type torrentDelegate struct {
	list.DefaultDelegate
}

func (d torrentDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(TorrentListItem)
	if ok && ti.item.IsTrusted() {
		d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(ui.ColorSuccess)
		d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(ui.ColorSuccess)
	}
	d.DefaultDelegate.Render(w, m, index, item)
}

// TorrentsModel displays nyaa search results for a selected episode.
type TorrentsModel struct {
	animeID int
	request nyaa.SearchRequest
	list    list.Model
	spinner spinner.Model
	loading bool
	err     error
}

// NewTorrentsModel creates a torrents results view.
func NewTorrentsModel(animeID int, req nyaa.SearchRequest) TorrentsModel {
	base := list.NewDefaultDelegate()
	base.Styles.SelectedTitle = base.Styles.SelectedTitle.
		Foreground(ui.ColorPrimary).
		BorderLeftForeground(ui.ColorPrimary)
	base.Styles.SelectedDesc = base.Styles.SelectedDesc.
		Foreground(ui.ColorSecondary).
		BorderLeftForeground(ui.ColorPrimary)
	delegate := torrentDelegate{DefaultDelegate: base}

	l := list.New(nil, delegate, 0, 0)
	l.Title = fmt.Sprintf("Torrents - Episode %d", req.Episode)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = ui.TitleStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SpinnerStyle

	return TorrentsModel{
		animeID: animeID,
		request: req,
		list:    l,
		spinner: s,
		loading: true,
	}
}

func (m TorrentsModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		searchNyaaCmd(m.request),
	)
}

func (m TorrentsModel) Update(msg tea.Msg) (TorrentsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-6)
		return m, nil

	case NyaaResultsMsg:
		m.loading = false
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

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if m.err != nil {
			if msg.String() == "esc" || msg.String() == "enter" {
				m.err = nil
				return m, nil
			}
		}

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
				return NavigateToPlayerMsg{
					MagnetURI:  magnetURI,
					AnimeID:    m.animeID,
					AnimeTitle: m.request.PrimaryTitle,
					Episode:    m.request.Episode,
				}
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
	queryLabel := nyaa.BuildSearchQuery(m.request.PrimaryTitle, m.request.Episode, m.request.Quality)
	header := lipgloss.NewStyle().Padding(1, 2).Render(
		ui.TitleStyle.Render("Episode Search") + "\n" +
			lipgloss.NewStyle().Foreground(ui.ColorSubtle).Render(queryLabel),
	)

	listHeight := height - lipgloss.Height(header)
	if listHeight < 0 {
		listHeight = 0
	}
	m.list.SetSize(width, listHeight)

	var body string
	switch {
	case m.loading:
		body = lipgloss.NewStyle().Padding(1, 2).Render(m.spinner.View() + " Searching nyaa.si...")
	case m.err != nil:
		body = lipgloss.NewStyle().Padding(1, 0).Render(ui.RenderError(m.err.Error()))
	case len(m.list.Items()) == 0:
		body = ui.HelpStyle.Render("  No torrents found for this episode")
	default:
		body = m.list.View()
	}

	content := header + "\n" + body
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func searchNyaaCmd(req nyaa.SearchRequest) tea.Cmd {
	return func() tea.Msg {
		results, err := nyaa.SearchWithFallback(context.Background(), req)
		return NyaaResultsMsg{Results: results, Err: err}
	}
}
