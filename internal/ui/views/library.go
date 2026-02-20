package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/anilist"
	"github.com/rayanxn/ani-tui/internal/ui"
)

type libraryFetchedMsg struct {
	lists map[string][]anilist.MediaList
	err   error
}

// LibraryListItem wraps an anilist.MediaList for bubbles/list rendering.
type LibraryListItem struct {
	entry anilist.MediaList
}

func (i LibraryListItem) Title() string       { return i.entry.Media.Title.DisplayTitle() }
func (i LibraryListItem) FilterValue() string  { return i.entry.Media.Title.DisplayTitle() }
func (i LibraryListItem) Description() string {
	parts := []string{fmt.Sprintf("Progress: %d", i.entry.Progress)}
	if i.entry.Media.Episodes > 0 {
		parts[0] = fmt.Sprintf("Progress: %d/%d", i.entry.Progress, i.entry.Media.Episodes)
	}
	if i.entry.Score > 0 {
		parts = append(parts, fmt.Sprintf("Score: %d", i.entry.Score))
	}
	return strings.Join(parts, " · ")
}

var tabLabels = []string{"Watching", "Completed", "Planning", "Dropped", "Paused"}
var tabStatuses = []string{"CURRENT", "COMPLETED", "PLANNING", "DROPPED", "PAUSED"}

// LibraryModel displays the user's anime library with tab-filtered categories.
type LibraryModel struct {
	client    *anilist.Client
	userID    int
	activeTab int
	lists     map[string][]anilist.MediaList
	list      list.Model
	spinner   spinner.Model
	loading   bool
	err       error
}

// NewLibraryModel creates a library view for the given user.
func NewLibraryModel(client *anilist.Client, userID int) LibraryModel {
	base := list.NewDefaultDelegate()
	base.Styles.SelectedTitle = base.Styles.SelectedTitle.
		Foreground(ui.ColorPrimary).
		BorderLeftForeground(ui.ColorPrimary)
	base.Styles.SelectedDesc = base.Styles.SelectedDesc.
		Foreground(ui.ColorSecondary).
		BorderLeftForeground(ui.ColorPrimary)
	l := list.New(nil, base, 0, 0)
	l.Title = "Library"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = ui.TitleStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SpinnerStyle

	return LibraryModel{
		client:  client,
		userID:  userID,
		list:    l,
		spinner: s,
		loading: true,
	}
}

func (m LibraryModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchLibraryCmd(m.client, m.userID),
	)
}

func (m LibraryModel) Update(msg tea.Msg) (LibraryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-6)
		return m, nil

	case libraryFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.lists = msg.lists
		m.updateListItems()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Don't handle tab keys when the list is filtering
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "tab":
			m.activeTab = (m.activeTab + 1) % len(tabLabels)
			m.updateListItems()
			return m, nil
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + len(tabLabels)) % len(tabLabels)
			m.updateListItems()
			return m, nil
		case "r":
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, fetchLibraryCmd(m.client, m.userID))
		case "enter":
			item, ok := m.list.SelectedItem().(LibraryListItem)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg {
				return NavigateToDetailMsg{AnimeID: item.entry.Media.ID}
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

// View renders the library view within the given dimensions.
func (m LibraryModel) View(width, height int) string {
	tabBar := m.renderTabBar(width)
	tabBarHeight := lipgloss.Height(tabBar)

	listHeight := height - tabBarHeight - 1
	if listHeight < 0 {
		listHeight = 0
	}
	m.list.SetSize(width, listHeight)

	var body string
	switch {
	case m.loading:
		body = lipgloss.NewStyle().Padding(1, 2).Render(m.spinner.View() + " Loading library...")
	case m.err != nil:
		body = lipgloss.NewStyle().Padding(1, 0).Render(ui.RenderError(m.err.Error()))
	case len(m.list.Items()) == 0:
		body = ui.HelpStyle.Render("  No anime in this category")
	default:
		body = m.list.View()
	}

	content := tabBar + "\n" + body
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func (m LibraryModel) renderTabBar(width int) string {
	var tabs []string
	for i, label := range tabLabels {
		count := len(m.lists[tabStatuses[i]])
		text := fmt.Sprintf("%s (%d)", label, count)
		if i == m.activeTab {
			tabs = append(tabs, ui.ActiveTabStyle.Render(text))
		} else {
			tabs = append(tabs, ui.InactiveTabStyle.Render(text))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	gap := width - lipgloss.Width(row)
	if gap > 0 {
		fill := lipgloss.NewStyle().Background(ui.ColorHeaderBg).Render(strings.Repeat(" ", gap))
		row += fill
	}
	return row
}

func (m *LibraryModel) updateListItems() {
	status := tabStatuses[m.activeTab]
	entries := m.lists[status]
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = LibraryListItem{entry: e}
	}
	m.list.SetItems(items)
	m.list.ResetSelected()
}

func fetchLibraryCmd(client *anilist.Client, userID int) tea.Cmd {
	return func() tea.Msg {
		collection, err := client.GetUserList(context.Background(), userID)
		if err != nil {
			return libraryFetchedMsg{err: err}
		}

		lists := make(map[string][]anilist.MediaList)
		for _, group := range collection.Lists {
			lists[group.Status] = group.Entries
		}
		return libraryFetchedMsg{lists: lists}
	}
}
