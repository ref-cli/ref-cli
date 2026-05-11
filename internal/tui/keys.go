package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	EntryUp    key.Binding
	EntryDown  key.Binding
	Copy       key.Binding
	Edit       key.Binding
	Fullscreen key.Binding
	Quit       key.Binding
	Back       key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "ctrl+p"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "ctrl+n"),
		key.WithHelp("↓", "down"),
	),
	EntryUp: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "prev entry"),
	),
	EntryDown: key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "next entry"),
	),
	Copy: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Fullscreen: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "fullscreen"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc", "back"),
	),
}
