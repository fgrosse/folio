package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// testConfigKeys are the keys the tests of the dialog edit: the tax rate, which is typed, and the
// switch for the values after tax, which is picked. They are named here rather than taken as portfolio.ConfigKeys
// so that a key added to folio does not change what these tests see.
func testConfigKeys(t *testing.T) []portfolio.ConfigKey {
	t.Helper()

	taxRate, err := portfolio.ConfigKeyNamed(portfolio.TaxRateKey)
	require.NoError(t, err)
	showNet, err := portfolio.ConfigKeyNamed(portfolio.ShowNetSummaryKey)
	require.NoError(t, err)

	return []portfolio.ConfigKey{taxRate, showNet}
}

// dialogText is what the dialog shows, without its styling.
func dialogText(d *ConfigDialog) string {
	return ansi.Strip(d.Layer().GetContent())
}

// TestConfigDialog_Rows covers what the dialog lists: one row for each key of the configuration,
// with the value it is set to, and under the rows what the selected key is for, which is the first
// when the dialog opens. A key that is not set shows its default, or says so if it has none. A key
// that takes one of a few values shows its value between the arrows that change it.
func TestConfigDialog_Rows(t *testing.T) {
	values := map[string]string{"tax-rate": "44.3%", "show-net-summary": "true"}
	d := NewConfigDialog(testConfigKeys(t), values, dialogWidth, DefaultStyle())

	text := dialogText(d)
	assert.Contains(t, text, "Configuration")
	assert.Contains(t, text, "> tax-rate          44.3%")
	assert.Contains(t, text, "  show-net-summary  ‹ true ›")
	assert.Contains(t, text, "The rate that the shares still to vest are taxed at")
	assert.NotContains(t, text, "Whether the header shows", "only the selected key is described")

	d = NewConfigDialog(testConfigKeys(t), nil, dialogWidth, DefaultStyle())

	text = dialogText(d)
	assert.Contains(t, text, "> tax-rate          not set")
	assert.Contains(t, text, "  show-net-summary  ‹ false ›")
}

// TestConfigDialog_Select covers moving through the keys: down and up select the next key and the
// one before, as do j and k, which move through every table of folio, and both wrap around the
// ends. What is described under the rows is the key that is selected.
func TestConfigDialog_Select(t *testing.T) {
	up, down := tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyDown}

	tests := map[string]struct {
		keys     []tea.KeyPressMsg
		expected string
	}{
		"nothing pressed yet":       {expected: "> tax-rate"},
		"down selects the next":     {keys: []tea.KeyPressMsg{down}, expected: "> show-net-summary"},
		"down wraps around":         {keys: []tea.KeyPressMsg{down, down}, expected: "> tax-rate"},
		"up selects the one before": {keys: []tea.KeyPressMsg{down, up}, expected: "> tax-rate"},
		"up wraps around":           {keys: []tea.KeyPressMsg{up}, expected: "> show-net-summary"},
		"j is down":                 {keys: []tea.KeyPressMsg{keyPressed("j")}, expected: "> show-net-summary"},
		"k is up":                   {keys: []tea.KeyPressMsg{keyPressed("k")}, expected: "> show-net-summary"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			d := NewConfigDialog(testConfigKeys(t), nil, dialogWidth, DefaultStyle())
			for _, key := range tt.keys {
				assert.Nil(t, d.HandleKeyPress(key), "moving the selection sends nothing")
			}

			assert.Contains(t, dialogText(d), tt.expected)
		})
	}

	d := NewConfigDialog(testConfigKeys(t), nil, dialogWidth, DefaultStyle())
	d.HandleKeyPress(down)
	assert.Contains(t, dialogText(d), "Whether the header shows the values after tax")
}

// TestConfigDialog_Choose covers changing a key that takes one of a few values: right and left
// pick the next value and the one before, around the ends, and enter and space pick the next, so
// that there is nothing to type. The dialog asks for the value to be set with a SetConfigMsg and
// goes on showing the value it was given until it is given the one that was stored. A key that
// takes any text has no next value, so the arrows do nothing on it.
func TestConfigDialog_Choose(t *testing.T) {
	left, right := tea.KeyPressMsg{Code: tea.KeyLeft}, tea.KeyPressMsg{Code: tea.KeyRight}
	enter, space := tea.KeyPressMsg{Code: tea.KeyEnter}, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}

	tests := map[string]struct {
		values   map[string]string
		key      tea.KeyPressMsg
		expected string
	}{
		"right picks the next":      {values: map[string]string{"show-net-summary": "false"}, key: right, expected: "true"},
		"right wraps around":        {values: map[string]string{"show-net-summary": "true"}, key: right, expected: "false"},
		"left picks the one before": {values: map[string]string{"show-net-summary": "true"}, key: left, expected: "false"},
		"left wraps around":         {values: map[string]string{"show-net-summary": "false"}, key: left, expected: "true"},
		"from the default":          {key: right, expected: "true"},
		"enter picks the next":      {key: enter, expected: "true"},
		"space picks the next":      {key: space, expected: "true"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			d := NewConfigDialog(testConfigKeys(t), tt.values, dialogWidth, DefaultStyle())
			d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyDown})

			cmd := d.HandleKeyPress(tt.key)
			assert.Equal(t, SetConfigMsg{key: "show-net-summary", value: tt.expected}, runCmd(t, cmd))
		})
	}

	d := NewConfigDialog(testConfigKeys(t), nil, dialogWidth, DefaultStyle())
	assert.Nil(t, d.HandleKeyPress(right), "the tax rate has no next value")
	assert.Nil(t, d.HandleKeyPress(left), "nor one before")

	d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyDown})
	d.HandleKeyPress(right)
	assert.Contains(t, dialogText(d), "‹ false ›", "the value changes once it is stored")

	d.SetValues(map[string]string{"show-net-summary": "true"})
	assert.Contains(t, dialogText(d), "‹ true ›")
}

// typeIntoConfig presses the keys of s in the dialog, one after the other.
func typeIntoConfig(d *ConfigDialog, s string) {
	for _, msg := range keysPressed(s) {
		d.HandleKeyPress(msg.(tea.KeyPressMsg))
	}
}

// TestConfigDialog_Edit covers changing a key that takes any text: enter turns its value into a
// field with the value in it, which takes every key that is typed, the j and k that would move the
// selection included. Enter in the field asks for what was typed to be set, written the way the key
// stores it, and ends the edit.
func TestConfigDialog_Edit(t *testing.T) {
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	backspace := tea.KeyPressMsg{Code: tea.KeyBackspace}

	d := NewConfigDialog(testConfigKeys(t), map[string]string{"tax-rate": "44.3%"}, dialogWidth, DefaultStyle())
	assert.False(t, d.Editing())

	d.HandleKeyPress(enter)
	require.True(t, d.Editing())
	assert.Contains(t, dialogText(d), "> tax-rate          44.3%")

	typeIntoConfig(d, "jk")
	assert.Contains(t, dialogText(d), "> tax-rate          44.3%jk", "a letter is typed rather than moving the selection")

	for range "4.3%jk" {
		d.HandleKeyPress(backspace)
	}
	typeIntoConfig(d, "2.50")

	cmd := d.HandleKeyPress(enter)
	assert.Equal(t, SetConfigMsg{key: "tax-rate", value: "42.5%"}, runCmd(t, cmd))
	assert.False(t, d.Editing())
}

// TestConfigDialog_Refused covers a value the key refuses: the field stays open with the text
// still in it, and the dialog says why, so that the text can be corrected. Once it is, the reason
// is gone.
func TestConfigDialog_Refused(t *testing.T) {
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}

	d := NewConfigDialog(testConfigKeys(t), nil, dialogWidth, DefaultStyle())
	d.HandleKeyPress(enter)
	typeIntoConfig(d, "120")

	assert.Nil(t, d.HandleKeyPress(enter), "a refused value should send nothing")
	assert.True(t, d.Editing())
	assert.Contains(t, dialogText(d), "> tax-rate          120")
	assert.Contains(t, dialogText(d), "a tax rate is between 0% and 100%, not 120%")

	d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyBackspace})
	cmd := d.HandleKeyPress(enter)
	assert.Equal(t, SetConfigMsg{key: "tax-rate", value: "12%"}, runCmd(t, cmd))
	assert.NotContains(t, dialogText(d), "a tax rate is between")
}

// TestConfigDialog_Unset covers taking a value back: enter on a field with nothing in it asks for
// the key to be unset rather than handing nothing to the key, which would refuse it. The empty
// field says what enter will leave the key as.
func TestConfigDialog_Unset(t *testing.T) {
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}

	d := NewConfigDialog(testConfigKeys(t), map[string]string{"tax-rate": "5%"}, dialogWidth, DefaultStyle())
	d.HandleKeyPress(enter)
	d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyBackspace})
	d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyBackspace})
	assert.Contains(t, dialogText(d), "> tax-rate          not set")

	cmd := d.HandleKeyPress(enter)
	assert.Equal(t, UnsetConfigMsg{key: "tax-rate"}, runCmd(t, cmd))
	assert.False(t, d.Editing())
}

// TestConfigDialog_Close covers the ways out. Esc closes the dialog with a ConfigClosedMsg. While
// a value is being typed, esc is about that value instead: it drops what was typed, and why it was
// refused, and leaves the dialog open on the value the key had. Ctrl+c quits the program from
// either.
func TestConfigDialog_Close(t *testing.T) {
	enter, esc := tea.KeyPressMsg{Code: tea.KeyEnter}, tea.KeyPressMsg{Code: tea.KeyEscape}
	ctrlC := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

	d := NewConfigDialog(testConfigKeys(t), map[string]string{"tax-rate": "44.3%"}, dialogWidth, DefaultStyle())
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, d.HandleKeyPress(ctrlC)))

	d.HandleKeyPress(enter)
	typeIntoConfig(d, "000")
	d.HandleKeyPress(enter)
	require.Contains(t, dialogText(d), "is not a tax rate")
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, d.HandleKeyPress(ctrlC)))

	assert.Nil(t, d.HandleKeyPress(esc), "esc in the field should not close the dialog")
	assert.False(t, d.Editing())
	assert.Contains(t, dialogText(d), "> tax-rate          44.3%")
	assert.NotContains(t, dialogText(d), "44.3%000")
	assert.NotContains(t, dialogText(d), "is not a tax rate")

	assert.Equal(t, ConfigClosedMsg{}, runCmd(t, d.HandleKeyPress(esc)))
}

// TestConfigDialog_Help covers the line of keys at the foot of the dialog, which says what the keys
// do to the row that is selected: a value that is typed is edited, one that is picked is changed,
// and while a value is being typed, enter and esc are about that value.
func TestConfigDialog_Help(t *testing.T) {
	d := NewConfigDialog(testConfigKeys(t), nil, dialogWidth, DefaultStyle())
	assert.Contains(t, dialogText(d), "↑/k up • ↓/j down • enter edit • esc close")

	d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Contains(t, dialogText(d), "↑/k up • ↓/j down • ←/→ change • esc close")

	d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyUp})
	d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Contains(t, dialogText(d), "enter save • esc cancel")
	assert.NotContains(t, dialogText(d), "esc close")
}

// TestConfigDialog_Update covers the messages the app passes on to the dialog: a key press is
// handled as HandleKeyPress does, and anything else goes to the field while a value is being
// typed, which is how text that is pasted and the ticks that blink its cursor reach it.
func TestConfigDialog_Update(t *testing.T) {
	d := NewConfigDialog(testConfigKeys(t), nil, dialogWidth, DefaultStyle())

	assert.Nil(t, d.Update(tea.PasteMsg{Content: "44.3"}), "nothing is being typed yet")

	d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.True(t, d.Editing())

	d.Update(tea.PasteMsg{Content: "44.3"})
	assert.Contains(t, dialogText(d), "> tax-rate          44.3")
}
