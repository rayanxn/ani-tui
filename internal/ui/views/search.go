package views

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/ui"
)

// SearchModel handles anime search via text input and results list.
type SearchModel struct {
	input   textinput.Model
	list    list.Model
	focused bool // true when the text input has focus
}

// NewSearchModel creates a search view with a focused text input and empty list.
func NewSearchModel() SearchModel {
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

	return SearchModel{
		input:   ti,
		list:    l,
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

	case tea.KeyMsg:
		if m.focused {
			switch msg.String() {
			case "enter":
				// Will trigger AniList search in Phase 2
				query := m.input.Value()
				if query != "" {
					m.focused = false
					m.input.Blur()
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
			if item := m.list.SelectedItem(); item != nil {
				// Will emit NavigateToDetailMsg in Phase 2
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

	var hint string
	if len(m.list.Items()) == 0 && !m.focused {
		hint = ui.HelpStyle.Render("  Press / to search")
	}

	content := searchBar + "\n" + m.list.View() + hint

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(content)
}

// inputFocused reports whether the text input currently has focus.
func (m SearchModel) inputFocused() bool {
	return m.focused
}
