package views

import (
	"context"
	"fmt"
	"html"
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
		m.totalEpisodes = availableEpisodes(msg.Media)
		if m.totalEpisodes > 0 {
			m.selectedEpisode = 1
		} else {
			m.selectedEpisode = 0
		}
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
			if m.totalEpisodes > 0 && m.selectedEpisode < m.totalEpisodes {
				m.selectedEpisode++
			}
			return m, nil
		case "k", "up":
			if m.selectedEpisode > 1 {
				m.selectedEpisode--
			}
			return m, nil
		case "g":
			if m.totalEpisodes > 0 {
				m.selectedEpisode = 1
			}
			return m, nil
		case "G":
			if m.totalEpisodes > 0 {
				m.selectedEpisode = m.totalEpisodes
			}
			return m, nil
		case "enter":
			if m.selectedEpisode <= 0 {
				return m, nil
			}
			return m, func() tea.Msg {
				return NavigateToTorrentsMsg{
					AnimeID:    m.animeID,
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
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary)
	valueStyle := lipgloss.NewStyle().Foreground(ui.ColorText)
	subtleStyle := lipgloss.NewStyle().Foreground(ui.ColorSubtle)
	divider := ui.DimDivider(width)

	var lines []string

	// Title block
	lines = append(lines, ui.TitleStyle.Render(m.media.Title.DisplayTitle()))
	if m.media.Title.Native != "" {
		lines = append(lines, ui.SubtitleStyle.Render(m.media.Title.Native))
	}
	lines = append(lines, divider)

	// Two-column metadata grid
	colWidth := width / 2
	row := func(l1, v1, l2, v2 string) {
		left := labelStyle.Render(l1+": ") + valueStyle.Render(v1)
		padded := lipgloss.NewStyle().Width(colWidth).Render(left)
		if l2 != "" {
			right := labelStyle.Render(l2+": ") + valueStyle.Render(v2)
			lines = append(lines, padded+right)
		} else {
			lines = append(lines, padded)
		}
	}

	format := m.media.Format
	status := formatStatus(m.media.Status)
	if format != "" && status != "" {
		row("Format", format, "Status", status)
	} else if format != "" {
		row("Format", format, "", "")
	} else if status != "" {
		row("Status", status, "", "")
	}

	episodes := ""
	if m.media.Episodes > 0 {
		episodes = fmt.Sprintf("%d", m.media.Episodes)
	}
	duration := ""
	if m.media.Duration > 0 {
		duration = fmt.Sprintf("%d min/ep", m.media.Duration)
	}
	if episodes != "" || duration != "" {
		row("Episodes", episodes, "Duration", duration)
	}

	season := ""
	if m.media.Season != "" && m.media.SeasonYear > 0 {
		season = fmt.Sprintf("%s %d", formatSeason(m.media.Season), m.media.SeasonYear)
	}
	source := formatSource(m.media.Source)
	if season != "" || source != "" {
		row("Season", season, "Source", source)
	}

	if m.media.AverageScore > 0 {
		score := ui.ScoreStyle.Render(fmt.Sprintf("★ %d%%", m.media.AverageScore))
		lines = append(lines, labelStyle.Render("Score: ")+score)
	}

	if m.media.NextAiringEpisode != nil {
		ep := m.media.NextAiringEpisode
		next := fmt.Sprintf("Ep %d", ep.Episode)
		if ep.TimeUntilAiring > 0 {
			next += fmt.Sprintf(" in %s", formatTimeUntil(ep.TimeUntilAiring))
		}
		lines = append(lines, labelStyle.Render("Next Episode: ")+valueStyle.Render(next))
	}

	// Genres / Studio section
	lines = append(lines, divider)

	if len(m.media.Genres) > 0 {
		lines = append(lines, labelStyle.Render("Genres: ")+valueStyle.Render(strings.Join(m.media.Genres, " · ")))
	}
	if len(m.media.Studios.Nodes) > 0 {
		names := make([]string, len(m.media.Studios.Nodes))
		for i, s := range m.media.Studios.Nodes {
			names[i] = s.Name
		}
		lines = append(lines, labelStyle.Render("Studio: ")+valueStyle.Render(strings.Join(names, ", ")))
	}

	// Synopsis
	if m.media.Description != "" {
		lines = append(lines, divider)
		lines = append(lines, labelStyle.Render("Synopsis"))
		desc := cleanDescription(m.media.Description)
		wrapped := wordWrap(desc, width)
		lines = append(lines, subtleStyle.Render(wrapped))
	}

	return strings.Join(lines, "\n")
}

// renderEpisodeSelector renders the episode list with cursor.
func (m DetailModel) renderEpisodeSelector(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary).Padding(0, 1)
	dimStyle := lipgloss.NewStyle().Foreground(ui.ColorSubtle)
	header := titleStyle.Render("Episodes")
	divider := "  " + ui.DimDivider(max(0, width-4))

	// Available lines for episode items (minus header, divider, and padding)
	listHeight := height - 3
	if listHeight < 1 {
		listHeight = 1
	}

	// Compute scroll offset to keep selected episode visible
	m.scrollOffset = 0
	if m.selectedEpisode > listHeight {
		m.scrollOffset = m.selectedEpisode - listHeight
	}

	var items []string
	if m.totalEpisodes <= 0 {
		items = append(items, dimStyle.Render("  No released episodes yet"))
	} else {
		for i := 1; i <= m.totalEpisodes; i++ {
			idx := i - 1
			if idx < m.scrollOffset || idx >= m.scrollOffset+listHeight {
				continue
			}

			if i == m.selectedEpisode {
				items = append(items, ui.SelectedItemStyle.Render(fmt.Sprintf("▸ Episode %d", i)))
			} else {
				items = append(items, dimStyle.Render(fmt.Sprintf("  Episode %d", i)))
			}
		}
	}

	content := header + "\n" + divider + "\n" + strings.Join(items, "\n")

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

// availableEpisodes returns how many episodes should be selectable for torrent search.
// For releasing shows, limit to already-aired episodes.
func availableEpisodes(media anilist.Media) int {
	if media.Status == "RELEASING" && media.NextAiringEpisode != nil {
		aired := media.NextAiringEpisode.Episode - 1
		if aired < 0 {
			return 0
		}
		return aired
	}
	if media.Episodes > 0 {
		return media.Episodes
	}
	return 0
}

func formatTimeUntil(seconds int) string {
	if seconds <= 0 {
		return "soon"
	}
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	switch {
	case days > 0:
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	case hours > 0:
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dm", max(1, minutes))
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

// cleanDescription strips HTML tags and decodes HTML entities from AniList descriptions.
func cleanDescription(s string) string {
	// Convert <br> / <br/> tags to newlines before stripping HTML
	s = strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n").Replace(s)

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
	return html.UnescapeString(strings.TrimSpace(result.String()))
}

// formatSeason converts AniList season enum to title case (e.g. WINTER -> Winter).
func formatSeason(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// formatSource converts AniList source enum to readable text (e.g. LIGHT_NOVEL -> Light Novel).
func formatSource(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
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
