package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/anilist"
	"github.com/rayanxn/ani-tui/internal/ui"
)

// AnimeDetailsMsg carries the result of fetching anime details.
type AnimeDetailsMsg struct {
	Media anilist.Media
	Err   error
}

// DetailModel displays anime metadata and an episode selector.
type DetailModel struct {
	client          *anilist.Client
	animeID         int
	media           anilist.Media
	viewport        viewport.Model
	spinner         spinner.Model
	loading         bool
	err             error
	selectedEpisode int // 1-indexed
	totalEpisodes   int
	scrollOffset    int // for scrolling the episode list
}

// NewDetailModel creates a detail view for the given anime ID.
func NewDetailModel(client *anilist.Client, animeID int) DetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SpinnerStyle

	return DetailModel{
		client:          client,
		animeID:         animeID,
		loading:         true,
		selectedEpisode: 1,
		spinner:         s,
	}
}

func (m DetailModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchAnimeDetailsCmd(m.client, m.animeID),
	)
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case AnimeDetailsMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.media = msg.Media
		m.totalEpisodes = msg.Media.Episodes
		if m.totalEpisodes <= 0 {
			m.totalEpisodes = 1
		}
		m.selectedEpisode = 1
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		switch msg.String() {
		case "j", "down":
			if m.selectedEpisode < m.totalEpisodes {
				m.selectedEpisode++
			}
			return m, nil
		case "k", "up":
			if m.selectedEpisode > 1 {
				m.selectedEpisode--
			}
			return m, nil
		case "g":
			m.selectedEpisode = 1
			return m, nil
		case "G":
			m.selectedEpisode = m.totalEpisodes
			return m, nil
		case "enter":
			return m, func() tea.Msg {
				return NavigateToTorrentsMsg{
					AnimeTitle: m.media.Title.DisplayTitle(),
					Episode:    m.selectedEpisode,
				}
			}
		}
	}

	return m, nil
}

// View renders the detail view within the given dimensions.
func (m DetailModel) View(width, height int) string {
	if m.loading {
		return lipgloss.NewStyle().Padding(1, 2).Render(
			m.spinner.View() + " Loading anime details...")
	}

	if m.err != nil {
		return lipgloss.NewStyle().Padding(1, 0).Render(
			ui.RenderError(m.err.Error()))
	}

	if width < 80 {
		return m.renderVertical(width, height)
	}
	return m.renderHorizontal(width, height)
}

// renderHorizontal renders side-by-side: left 2/3 metadata, right 1/3 episodes.
func (m DetailModel) renderHorizontal(width, height int) string {
	leftWidth := width*2/3 - 2
	rightWidth := width - leftWidth - 1

	metaContent := m.renderMetadata(leftWidth - 4)
	m.viewport = viewport.New(leftWidth, height)
	m.viewport.SetContent(metaContent)
	m.viewport.Style = lipgloss.NewStyle().Padding(0, 2)

	leftPanel := m.viewport.View()

	episodePanel := m.renderEpisodeSelector(rightWidth, height)

	separator := lipgloss.NewStyle().
		Foreground(ui.ColorSubtle).
		Render(strings.Repeat("│\n", height))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, episodePanel)
}

// renderVertical renders stacked: metadata on top, episodes below.
func (m DetailModel) renderVertical(width, height int) string {
	metaHeight := height * 2 / 3
	epHeight := height - metaHeight

	metaContent := m.renderMetadata(width - 4)
	m.viewport = viewport.New(width, metaHeight)
	m.viewport.SetContent(metaContent)
	m.viewport.Style = lipgloss.NewStyle().Padding(0, 2)

	leftPanel := m.viewport.View()

	episodePanel := m.renderEpisodeSelector(width, epHeight)

	return lipgloss.JoinVertical(lipgloss.Left, leftPanel, episodePanel)
}

// renderMetadata formats the anime metadata block.
func (m DetailModel) renderMetadata(width int) string {
	title := ui.TitleStyle.Render(m.media.Title.DisplayTitle())

	var lines []string
	lines = append(lines, title)

	if m.media.Title.Native != "" {
		lines = append(lines, ui.SubtitleStyle.Render(m.media.Title.Native))
	}

	lines = append(lines, "")

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary)
	valueStyle := lipgloss.NewStyle().Foreground(ui.ColorText)

	addField := func(label, value string) {
		if value != "" && value != "0" {
			lines = append(lines, labelStyle.Render(label+": ")+valueStyle.Render(value))
		}
	}

	addField("Format", m.media.Format)
	addField("Status", formatStatus(m.media.Status))

	if m.media.Episodes > 0 {
		addField("Episodes", fmt.Sprintf("%d", m.media.Episodes))
	}
	if m.media.Duration > 0 {
		addField("Duration", fmt.Sprintf("%d min/ep", m.media.Duration))
	}
	if m.media.AverageScore > 0 {
		addField("Score", fmt.Sprintf("%d%%", m.media.AverageScore))
	}
	if len(m.media.Genres) > 0 {
		addField("Genres", strings.Join(m.media.Genres, ", "))
	}
	if len(m.media.Studios.Nodes) > 0 {
		names := make([]string, len(m.media.Studios.Nodes))
		for i, s := range m.media.Studios.Nodes {
			names[i] = s.Name
		}
		addField("Studio", strings.Join(names, ", "))
	}

	if m.media.NextAiringEpisode != nil {
		ep := m.media.NextAiringEpisode
		addField("Next Episode", fmt.Sprintf("Ep %d", ep.Episode))
	}

	if m.media.Description != "" {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Synopsis"))
		desc := cleanDescription(m.media.Description)
		wrapped := wordWrap(desc, width)
		lines = append(lines, valueStyle.Render(wrapped))
	}

	return strings.Join(lines, "\n")
}

// renderEpisodeSelector renders the episode list with cursor.
func (m DetailModel) renderEpisodeSelector(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary).Padding(0, 1)
	header := titleStyle.Render("Episodes")

	// Available lines for episode items (minus header line and padding)
	listHeight := height - 2
	if listHeight < 1 {
		listHeight = 1
	}

	// Compute scroll offset to keep selected episode visible
	m.scrollOffset = 0
	if m.selectedEpisode > listHeight {
		m.scrollOffset = m.selectedEpisode - listHeight
	}

	var items []string
	for i := 1; i <= m.totalEpisodes; i++ {
		idx := i - 1
		if idx < m.scrollOffset || idx >= m.scrollOffset+listHeight {
			continue
		}

		label := fmt.Sprintf("  Episode %d", i)
		if i == m.selectedEpisode {
			label = ui.SelectedItemStyle.Render(fmt.Sprintf("▸ Episode %d", i))
		}
		items = append(items, label)
	}

	content := header + "\n" + strings.Join(items, "\n")

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}

// fetchAnimeDetailsCmd returns a Cmd that fetches anime details from AniList.
func fetchAnimeDetailsCmd(client *anilist.Client, id int) tea.Cmd {
	return func() tea.Msg {
		media, err := client.GetAnimeDetails(context.Background(), id)
		return AnimeDetailsMsg{Media: media, Err: err}
	}
}

// formatStatus converts API status enum to display text.
func formatStatus(status string) string {
	switch status {
	case "FINISHED":
		return "Finished"
	case "RELEASING":
		return "Releasing"
	case "NOT_YET_RELEASED":
		return "Not Yet Released"
	case "CANCELLED":
		return "Cancelled"
	case "HIATUS":
		return "Hiatus"
	default:
		return status
	}
}

// cleanDescription strips HTML tags from AniList descriptions.
func cleanDescription(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

// wordWrap wraps text to the given width at word boundaries.
func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}

	var result strings.Builder
	for _, paragraph := range strings.Split(s, "\n") {
		if result.Len() > 0 {
			result.WriteByte('\n')
		}

		words := strings.Fields(paragraph)
		if len(words) == 0 {
			continue
		}

		lineLen := 0
		for i, word := range words {
			wl := len(word)
			if i > 0 && lineLen+1+wl > width {
				result.WriteByte('\n')
				lineLen = 0
			} else if i > 0 {
				result.WriteByte(' ')
				lineLen++
			}
			result.WriteString(word)
			lineLen += wl
		}
	}

	return result.String()
}
