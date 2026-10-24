package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Back      key.Binding
	Tab       key.Binding
	Search    key.Binding
	Timeline  key.Binding
	Dashboard key.Binding
	Links     key.Binding
	NextItem  key.Binding
	PrevItem  key.Binding
	OpenItem  key.Binding
	Refresh   key.Binding
	Quit      key.Binding
	Help      key.Binding
}

var Keys = KeyMap{
	NextItem: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next issue/PR"),
	),
	PrevItem: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "previous issue/PR"),
	),
	OpenItem: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open issue/PR in browser"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k/up", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/down", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch view"),
	),
	Search: key.NewBinding(
		key.WithKeys("/", "s"),
		key.WithHelp("/", "search"),
	),
	Timeline: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "timeline"),
	),
	Dashboard: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "dashboard"),
	),
	Links: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "links"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}
