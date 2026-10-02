package tui

import (
	"charm.land/bubbles/v2/key"
)

// keyMap holds the keys of the views themselves, each of which answers to the ones that make sense
// for it. Moving the selection is left to the table, which has a key map of its own; the help lines
// show keys from both.
type keyMap struct {
	Add     key.Binding
	Edit    key.Binding
	Sell    key.Binding
	Delete  key.Binding
	Release key.Binding
	Quit    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Add:     key.NewBinding(key.WithKeys("a", "insert"), key.WithHelp("a", "add")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Sell:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sell")),
		Delete:  key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "delete")),
		Release: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "release")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
