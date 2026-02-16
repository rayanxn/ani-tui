package ui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines keybindings available in all views.
type GlobalKeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	Back    key.Binding
	Tab     key.Binding
	Enter   key.Binding
	ForceQuit key.Binding
}

var GlobalKeys = GlobalKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch view"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	ForceQuit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "force quit"),
	),
}

// SearchKeyMap defines keybindings for the search view.
type SearchKeyMap struct {
	Focus key.Binding
}

var SearchKeys = SearchKeyMap{
	Focus: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "focus search"),
	),
}

// TorrentsKeyMap defines keybindings for the torrents view.
type TorrentsKeyMap struct {
	Retry key.Binding
}

var TorrentsKeys = TorrentsKeyMap{
	Retry: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "retry search"),
	),
}
