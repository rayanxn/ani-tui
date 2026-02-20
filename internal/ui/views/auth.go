package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rayanxn/ani-tui/internal/anilist"
	"github.com/rayanxn/ani-tui/internal/config"
	"github.com/rayanxn/ani-tui/internal/ui"
)

// AuthCompleteMsg is emitted when authentication succeeds and config is saved.
type AuthCompleteMsg struct {
	Token  string
	UserID int
}

type tokenVerifiedMsg struct {
	user anilist.User
	err  error
}

type configSavedMsg struct {
	err error
}

type authStep int

const (
	authShowURL authStep = iota
	authPasteToken
	authVerifying
	authSaving
)

// AuthModel handles the OAuth login flow.
type AuthModel struct {
	step    authStep
	input   textinput.Model
	spinner spinner.Model
	token   string
	user    anilist.User
	err     error
}

// NewAuthModel creates a new auth view.
func NewAuthModel() AuthModel {
	ti := textinput.New()
	ti.Placeholder = "Paste your AniList token here..."
	ti.EchoMode = textinput.EchoPassword
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SpinnerStyle

	return AuthModel{
		step:    authShowURL,
		input:   ti,
		spinner: s,
	}
}

func (m AuthModel) Init() tea.Cmd {
	return nil
}

func (m AuthModel) Update(msg tea.Msg) (AuthModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tokenVerifiedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.step = authPasteToken
			m.input.Focus()
			return m, textinput.Blink
		}
		m.user = msg.user
		m.step = authSaving
		return m, saveConfigCmd(m.token, m.user.ID)

	case configSavedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.step = authPasteToken
			m.input.Focus()
			return m, textinput.Blink
		}
		return m, func() tea.Msg {
			return AuthCompleteMsg{
				Token:  m.token,
				UserID: m.user.ID,
			}
		}

	case spinner.TickMsg:
		if m.step == authVerifying || m.step == authSaving {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case authShowURL:
			if msg.String() == "enter" {
				m.step = authPasteToken
				m.input.Focus()
				return m, textinput.Blink
			}

		case authPasteToken:
			switch msg.String() {
			case "esc":
				m.step = authShowURL
				m.err = nil
				m.input.Reset()
				return m, nil
			case "enter":
				token := strings.TrimSpace(m.input.Value())
				if token == "" {
					return m, nil
				}
				m.token = token
				m.err = nil
				m.step = authVerifying
				return m, tea.Batch(m.spinner.Tick, verifyTokenCmd(token))
			}

			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the auth view within the given dimensions.
func (m AuthModel) View(width, height int) string {
	var content string

	switch m.step {
	case authShowURL:
		url := anilist.AuthURL()
		lines := []string{
			ui.TitleStyle.Render("AniList Login"),
			"",
			"Visit this URL to get your access token:",
			"",
			lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent).Render(url),
			"",
			lipgloss.NewStyle().Foreground(ui.ColorSubtle).Render(
				"After authorizing, copy the token from the URL bar."),
			"",
			ui.HelpStyle.Render("Press enter to continue"),
		}
		box := ui.BorderedBoxStyle.Width(min(width-4, 70)).Render(strings.Join(lines, "\n"))
		content = ui.CenterHorizontal(width, box)

	case authPasteToken:
		lines := []string{
			ui.TitleStyle.Render("Paste Token"),
			"",
			m.input.View(),
		}
		if m.err != nil {
			lines = append(lines, "", ui.ErrorStyle.Render(m.err.Error()))
		}
		lines = append(lines, "", ui.HelpStyle.Render("enter submit  |  esc back"))
		box := ui.BorderedBoxStyle.Width(min(width-4, 70)).Render(strings.Join(lines, "\n"))
		content = ui.CenterHorizontal(width, box)

	case authVerifying:
		content = ui.CenterHorizontal(width,
			m.spinner.View()+" Verifying token...")

	case authSaving:
		content = ui.CenterHorizontal(width,
			m.spinner.View()+" Saving configuration...")
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(height/4, 0, 0, 0).
		Render(content)
}

func verifyTokenCmd(token string) tea.Cmd {
	return func() tea.Msg {
		client := anilist.NewClient(token)
		user, err := client.GetViewer(context.Background())
		return tokenVerifiedMsg{user: user, err: err}
	}
}

func saveConfigCmd(token string, userID int) tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return configSavedMsg{err: fmt.Errorf("load config: %w", err)}
		}
		cfg.AniListToken = token
		cfg.AniListUserID = userID
		if err := config.Save(cfg); err != nil {
			return configSavedMsg{err: fmt.Errorf("save config: %w", err)}
		}
		return configSavedMsg{}
	}
}
