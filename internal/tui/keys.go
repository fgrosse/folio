package tui

import (
	"charm.land/bubbles/v2/key"
)

// keyMap holds the keys of the views themselves, each of which answers to the ones that make sense
// for it. Moving the selection is left to the table, which has a key map of its own; the help lines
// show keys from both.
type keyMap struct {
	Add     key.Binding
	Delete  key.Binding
	Release key.Binding
	Quit    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Add:     key.NewBinding(key.WithKeys("a", "insert"), key.WithHelp("a", "add")),
		Delete:  key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "delete")),
		Release: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "release")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
