package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/config"
	"github.com/rayanxn/ani-tui/internal/player"
	"github.com/rayanxn/ani-tui/internal/torrent"
	"github.com/rayanxn/ani-tui/internal/ui"
)

// Messages for the player view lifecycle.
type (
	playerReadyMsg struct {
		torrentClient *torrent.Client
		session   *player.Session
		err       error
	}
	statsTickMsg struct{}
	mpvExitMsg   struct{ err error }
)

// PlayerModel manages torrent streaming and mpv playback.
type PlayerModel struct {
	magnetURI  string
	animeTitle string
	episode    int
	cfg        config.Config
	torrentClient  *torrent.Client
	session    *player.Session
	stats      torrent.Stats
	spinner    spinner.Model
	loading    bool
	done       bool
	err        error
}

// NewPlayerModel creates a player view for the given magnet URI.
func NewPlayerModel(magnetURI, animeTitle string, episode int, cfg config.Config) PlayerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SpinnerStyle

	return PlayerModel{
		magnetURI:  magnetURI,
		animeTitle: animeTitle,
		episode:    episode,
		cfg:        cfg,
		spinner:    s,
		loading:    true,
	}
}

func (m PlayerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		startStreamCmd(m.magnetURI, m.cfg),
	)
}

func (m PlayerModel) Update(msg tea.Msg) (PlayerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case playerReadyMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			return m, nil
		}
		m.loading = false
		m.torrentClient = msg.torrentClient
		m.session = msg.session
		return m, tea.Batch(
			waitForMpvCmd(m.session),
			statsTickCmd(),
		)

	case statsTickMsg:
		if m.done || m.torrentClient == nil {
			return m, nil
		}
		t := m.torrentClient.ActiveTorrent()
		if t != nil {
			m.stats = torrent.GetStats(t)
		}
		return m, statsTickCmd()

	case mpvExitMsg:
		m.done = true
		return m, func() tea.Msg { return NavigateBackMsg{} }

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// View renders the player view.
func (m PlayerModel) View(width, height int) string {
	title := lipgloss.NewStyle().Padding(1, 2).Render(
		ui.TitleStyle.Render(fmt.Sprintf("Playing - %s Episode %d", m.animeTitle, m.episode)),
	)

	var body string
	switch {
	case m.loading:
		body = lipgloss.NewStyle().Padding(1, 2).Render(
			m.spinner.View() + " Starting stream...",
		)
	case m.err != nil:
		body = lipgloss.NewStyle().Padding(1, 0).Render(ui.RenderError(m.err.Error()))
	default:
		body = m.renderStats(width)
	}

	content := title + "\n" + body
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func (m PlayerModel) renderStats(width int) string {
	progress := m.stats.Progress()
	barWidth := width - 8
	if barWidth < 10 {
		barWidth = 10
	}
	filled := int(progress * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(ui.ColorSubtle).Render(strings.Repeat("░", barWidth-filled))

	pct := fmt.Sprintf("%.1f%%", progress*100)
	downloaded := fmt.Sprintf("%s / %s", torrent.FormatBytes(m.stats.BytesCompleted), torrent.FormatBytes(m.stats.BytesTotal))
	peers := fmt.Sprintf("Peers: %d  |  Seeders: %d", m.stats.Peers, m.stats.Seeders)

	lines := []string{
		"",
		"  " + bar,
		"",
		lipgloss.NewStyle().Padding(0, 2).Render(
			lipgloss.NewStyle().Bold(true).Render(pct) + "  " +
				lipgloss.NewStyle().Foreground(ui.ColorSubtle).Render(downloaded),
		),
		lipgloss.NewStyle().Padding(0, 2).Foreground(ui.ColorSubtle).Render(peers),
		"",
		ui.HelpStyle.Render("  mpv is playing in a separate window  |  esc back"),
	}

	return strings.Join(lines, "\n")
}

// Cleanup releases torrent and player resources.
func (m PlayerModel) Cleanup() {
	if m.session != nil {
		m.session.Close()
	}
	if m.torrentClient != nil {
		m.torrentClient.Close()
	}
}

func startStreamCmd(magnetURI string, cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		tc, err := torrent.NewClient(cfg.DownloadDir)
		if err != nil {
			return playerReadyMsg{err: fmt.Errorf("create torrent client: %w", err)}
		}

		reader, filename, err := tc.AddMagnetAndStream(context.Background(), magnetURI)
		if err != nil {
			tc.Close()
			return playerReadyMsg{err: fmt.Errorf("stream torrent: %w", err)}
		}

		session, err := player.Start(cfg.MpvPath, reader, filename)
		if err != nil {
			tc.Close()
			return playerReadyMsg{err: fmt.Errorf("start mpv: %w", err)}
		}

		return playerReadyMsg{torrentClient: tc, session: session}
	}
}

func waitForMpvCmd(s *player.Session) tea.Cmd {
	return func() tea.Msg {
		err := <-s.Wait()
		return mpvExitMsg{err: err}
	}
}

func statsTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return statsTickMsg{}
	})
}
