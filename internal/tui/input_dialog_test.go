package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// submittedMsg stands in for whatever message a view wants once its dialog is confirmed.
type submittedMsg struct{ value string }

// newTestingDialog returns a dialog that accepts whatever is typed into it, except "bad".
func newTestingDialog() *InputDialog {
	submit := func(value string) (tea.Msg, error) {
		if value == "bad" {
			return nil, errors.New("bad is not a thing")
		}
		return submittedMsg{value: value}, nil
	}

	return NewInputDialog("New thing", "a thing", submit, 20, DefaultStyle())
}

// typeInto presses the keys of s in the dialog, one after the other.
func typeInto(d *InputDialog, s string) {
	for _, msg := range keysPressed(s) {
		d.HandleKeyPress(msg.(tea.KeyPressMsg))
	}
}

// TestInputDialog_Submit covers the dialog doing what it is for: what is typed goes into its field,
// q and digits included, which are shortcuts everywhere else, and enter hands the text, without the
// space around it, to the function the dialog was made with and sends the message that returns.
func TestInputDialog_Submit(t *testing.T) {
	d := newTestingDialog()

	typeInto(d, " 12 quarts ")
	assert.Equal(t, " 12 quarts ", d.Value())

	cmd := d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Equal(t, submittedMsg{value: "12 quarts"}, runCmd(t, cmd))
}

// TestInputDialog_Refused covers a text the dialog's function refuses: the dialog stays open with
// the text still in its field, and says why under it, so that the text can be corrected.
func TestInputDialog_Refused(t *testing.T) {
	d := newTestingDialog()
	typeInto(d, "bad")

	cmd := d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.Nil(t, cmd, "a refused text should send nothing")
	assert.Equal(t, "bad", d.Value())
	assert.Contains(t, ansi.Strip(d.Layer().GetContent()), "bad is not a thing")
}

// TestInputDialog_Cancel covers the ways out of the dialog without submitting anything: esc, and
// enter on a field with nothing in it, which means "never mind". Ctrl+c quits the program.
func TestInputDialog_Cancel(t *testing.T) {
	cases := map[string]struct {
		typed    string
		key      tea.KeyPressMsg
		expected tea.Msg
	}{
		"esc":                     {typed: "12 PANW", key: tea.KeyPressMsg{Code: tea.KeyEscape}, expected: InputCanceledMsg{}},
		"enter on an empty field": {typed: "", key: tea.KeyPressMsg{Code: tea.KeyEnter}, expected: InputCanceledMsg{}},
		"enter on only spaces":    {typed: "  ", key: tea.KeyPressMsg{Code: tea.KeyEnter}, expected: InputCanceledMsg{}},
		"ctrl+c":                  {typed: "12 PANW", key: tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, expected: tea.QuitMsg{}},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			d := newTestingDialog()
			typeInto(d, tt.typed)

			cmd := d.HandleKeyPress(tt.key)
			require.NotNil(t, cmd)
			assert.Equal(t, tt.expected, cmd())
		})
	}
}

// TestInputDialog_SetValue covers a dialog that opens with something in its field already, such as
// the shares of a vest that most likely all arrived: the text is there to be confirmed as it is, and
// the cursor is at its end, where a backspace corrects it.
func TestInputDialog_SetValue(t *testing.T) {
	d := newTestingDialog()

	d.SetValue("10")
	assert.Equal(t, "10", d.Value())

	d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyBackspace})
	typeInto(d, "2")
	cmd := d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Equal(t, submittedMsg{value: "12"}, runCmd(t, cmd))
}
