package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/anilist"
	"github.com/rayanxn/ani-tui/internal/ui"
)

// SearchResultsMsg carries results from an AniList search.
type SearchResultsMsg struct {
	Results []anilist.Media
	Err     error
}

// AnimeListItem wraps a Media for use in a bubbles/list.
type AnimeListItem struct {
	media anilist.Media
}

func (i AnimeListItem) Title() string       { return i.media.Title.DisplayTitle() }
func (i AnimeListItem) FilterValue() string { return i.media.Title.DisplayTitle() }

func (i AnimeListItem) Description() string {
	eps := "?"
	if i.media.Episodes > 0 {
		eps = fmt.Sprintf("%d", i.media.Episodes)
	}
	score := "N/A"
	if i.media.AverageScore > 0 {
		score = fmt.Sprintf("%d%%", i.media.AverageScore)
	}
	return fmt.Sprintf("%s · %s eps · %s · %s",
		i.media.Format, eps, score, i.media.Status)
}

// Media returns the underlying anilist.Media.
func (i AnimeListItem) Media() anilist.Media { return i.media }

// SearchModel handles anime search via text input and results list.
type SearchModel struct {
	client  *anilist.Client
	input   textinput.Model
	list    list.Model
	spinner spinner.Model
	focused bool // true when the text input has focus
	loading bool
	err     error
}

// NewSearchModel creates a search view with a focused text input and empty list.
func NewSearchModel(client *anilist.Client) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search anime..."
	ti.CharLimit = 100
	ti.Width = 40
	ti.Focus()

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(ui.ColorPrimary).
		BorderLeftForeground(ui.ColorPrimary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(ui.ColorSecondary).
		BorderLeftForeground(ui.ColorPrimary)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "Results"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = ui.TitleStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SpinnerStyle

	return SearchModel{
		client:  client,
		input:   ti,
		list:    l,
		spinner: s,
		focused: true,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-6)
		return m, nil

	case SearchResultsMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.err = nil
		items := make([]list.Item, len(msg.Results))
		for i, media := range msg.Results {
			items[i] = AnimeListItem{media: media}
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
		if m.focused {
			switch msg.String() {
			case "enter":
				query := m.input.Value()
				if query != "" {
					m.focused = false
					m.input.Blur()
					m.loading = true
					m.err = nil
					return m, tea.Batch(
						m.spinner.Tick,
						searchAniListCmd(m.client, query),
					)
				}
				return m, nil
			case "esc":
				if m.input.Value() != "" {
					m.focused = false
					m.input.Blur()
					return m, nil
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		// List is focused
		switch msg.String() {
		case "/":
			m.focused = true
			m.input.Focus()
			return m, textinput.Blink
		case "enter":
			if item, ok := m.list.SelectedItem().(AnimeListItem); ok {
				return m, func() tea.Msg {
					return NavigateToDetailMsg{AnimeID: item.media.ID}
				}
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if m.focused {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the search view within the given dimensions.
func (m SearchModel) View(width, height int) string {
	inputStyle := lipgloss.NewStyle().
		Padding(1, 2)

	searchBar := inputStyle.Render(m.input.View())

	listHeight := height - lipgloss.Height(searchBar)
	if listHeight < 0 {
		listHeight = 0
	}
	m.list.SetSize(width, listHeight)

	var body string
	switch {
	case m.loading:
		body = lipgloss.NewStyle().Padding(1, 2).Render(
			m.spinner.View() + " Searching...")
	case m.err != nil:
		body = lipgloss.NewStyle().Padding(1, 0).Render(
			ui.RenderError(m.err.Error()))
	case len(m.list.Items()) == 0 && !m.focused:
		body = ui.HelpStyle.Render("  Press / to search")
	default:
		body = m.list.View()
	}

	content := searchBar + "\n" + body

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(content)
}

// inputFocused reports whether the text input currently has focus.
func (m SearchModel) inputFocused() bool {
	return m.focused
}

// searchAniListCmd returns a Cmd that searches AniList for anime.
func searchAniListCmd(client *anilist.Client, query string) tea.Cmd {
	return func() tea.Msg {
		results, err := client.SearchAnime(context.Background(), query, 1)
		return SearchResultsMsg{Results: results, Err: err}
	}
}
