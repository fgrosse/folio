package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmDialog(t *testing.T) {
	type confirmed struct{} // stands in for whatever message the parent wants on a yes

	cases := map[string]struct {
		key      tea.KeyPressMsg
		expected tea.Msg // nil if the key should do nothing at all
	}{
		"y confirms":   {key: keyPressed("y"), expected: confirmed{}},
		"n cancels":    {key: keyPressed("n"), expected: ConfirmCanceledMsg{}},
		"esc cancels":  {key: tea.KeyPressMsg{Code: tea.KeyEscape}, expected: ConfirmCanceledMsg{}},
		"ctrl+c quits": {key: tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, expected: tea.QuitMsg{}},

		// Only a deliberate y confirms. Enter in particular is pressed out of habit, and here that
		// habit would delete something.
		"enter does nothing":      {key: tea.KeyPressMsg{Code: tea.KeyEnter}},
		"other keys do nothing":   {key: keyPressed("x")},
		"another delete does not": {key: keyPressed("d")},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			d := NewConfirmDialog("Delete lot", "Delete it?", confirmed{}, DefaultStyle())

			cmd := d.HandleKeyPress(c.key)
			if c.expected == nil {
				assert.Nil(t, cmd)
				return
			}

			require.NotNil(t, cmd)
			assert.Equal(t, c.expected, cmd())
		})
	}
}
