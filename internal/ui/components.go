package ui

import (
	"github.com/charmbracelet/lipgloss"
)

const appTitle = " ani-tui "

// RenderHeader returns a full-width header bar with the app title.
func RenderHeader(width int) string {
	title := HeaderStyle.Render(appTitle)
	gap := width - lipgloss.Width(title)
	if gap < 0 {
		gap = 0
	}
	fill := lipgloss.NewStyle().
		Background(ColorHeaderBg).
		Render(repeatChar(' ', gap))
	return title + fill
}

// RenderStatusBar returns a full-width status bar with the given text.
func RenderStatusBar(width int, text string) string {
	return StatusBarStyle.Width(width).Render(text)
}

// RenderError returns a styled error message.
func RenderError(msg string) string {
	return ErrorStyle.Render("Error: " + msg)
}

func repeatChar(c byte, n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}
