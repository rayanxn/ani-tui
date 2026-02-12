package ui

import "github.com/charmbracelet/lipgloss"

// Color palette — purple/magenta theme with adaptive light/dark support.
var (
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "#7B2FBE", Dark: "#BD93F9"}
	ColorSecondary = lipgloss.AdaptiveColor{Light: "#A855F7", Dark: "#FF79C6"}
	ColorAccent    = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#8BE9FD"}
	ColorSubtle    = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6272A4"}
	ColorText      = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F8F8F2"}
	ColorError     = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#FF5555"}
	ColorSuccess   = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#50FA7B"}
	ColorBg        = lipgloss.AdaptiveColor{Light: "#F9FAFB", Dark: "#282A36"}
	ColorHeaderBg  = lipgloss.AdaptiveColor{Light: "#7B2FBE", Dark: "#44475A"}
)

// Header renders the top bar with the app title.
var HeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FFFFFF")).
	Background(ColorHeaderBg).
	Padding(0, 1)

// StatusBar renders the bottom status bar.
var StatusBarStyle = lipgloss.NewStyle().
	Foreground(ColorSubtle).
	Padding(0, 1)

// Title styles for section headings.
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorPrimary).
	MarginBottom(1)

// Subtitle for secondary headings.
var SubtitleStyle = lipgloss.NewStyle().
	Foreground(ColorSecondary)

// ErrorStyle for error messages.
var ErrorStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorError).
	Padding(0, 1)

// HelpStyle for the help/keybinding text.
var HelpStyle = lipgloss.NewStyle().
	Foreground(ColorSubtle)

// SelectedItemStyle highlights the currently selected list item.
var SelectedItemStyle = lipgloss.NewStyle().
	Foreground(ColorPrimary).
	Bold(true)

// NormalItemStyle for unselected list items.
var NormalItemStyle = lipgloss.NewStyle().
	Foreground(ColorText)

// ActiveTabStyle for the currently active tab.
var ActiveTabStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FFFFFF")).
	Background(ColorPrimary).
	Padding(0, 2)

// InactiveTabStyle for inactive tabs.
var InactiveTabStyle = lipgloss.NewStyle().
	Foreground(ColorSubtle).
	Background(ColorHeaderBg).
	Padding(0, 2)

// BorderedBoxStyle for overlays and modal-like content.
var BorderedBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorPrimary).
	Padding(1, 2)

// SpinnerStyle for loading spinners.
var SpinnerStyle = lipgloss.NewStyle().
	Foreground(ColorSecondary)

// CenterHorizontal centers text horizontally within the given width.
func CenterHorizontal(width int, s string) string {
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, s)
}
