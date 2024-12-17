package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
)

type KeyMap struct {
	TableKeyMap table.KeyMap
	Help key.Binding
	Quit key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k KeyMap) ShortHelp() []key.Binding {
	tableHelp := k.TableKeyMap.ShortHelp()
	return append( []key.Binding{k.Help, k.Quit}, tableHelp...)
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	tableHelp := k.TableKeyMap.FullHelp()
	return append([][]key.Binding{{k.Help, k.Quit}}, tableHelp... )
}

var keymap = KeyMap{
	TableKeyMap: table.DefaultKeyMap(),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}