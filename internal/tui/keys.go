package tui

import (
	"charm.land/bubbles/v2/key"
)

// keyMap holds the keys of the views themselves, each of which answers to the ones that make sense
// for it. Moving the selection is left to the table, which has a key map of its own; the help lines
// show keys from both.
type keyMap struct {
	Add       key.Binding
	Edit      key.Binding
	Sell      key.Binding
	Delete    key.Binding
	Release   key.Binding
	Details   key.Binding
	Clear     key.Binding
	ClearYear key.Binding
	Close     key.Binding
	Quit      key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Add:       key.NewBinding(key.WithKeys("a", "insert"), key.WithHelp("a", "add")),
		Edit:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Sell:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sell")),
		Delete:    key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "delete")),
		Release:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "release")),
		Details:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Clear:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear tax")),
		ClearYear: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "clear year")),
		Close:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
