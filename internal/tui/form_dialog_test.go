package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formMsg stands in for whatever message a view wants once its form is confirmed.
type formMsg struct{ values []string }

var (
	tabKey      = tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTabKey = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	enterKey    = tea.KeyPressMsg{Code: tea.KeyEnter}
	saveKey     = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
)

// newTestingForm returns a form with three fields of one line, the last of them filled in already,
// and one of several lines. It accepts whatever is typed into it, unless the first field says "bad".
func newTestingForm() *FormDialog {
	fields := []FormField{
		{Label: "Shares", Placeholder: "250"},
		{Label: "Price", Placeholder: "396.25"},
		{Label: "Date", Value: "2026-10-02"},
		{Label: "Notes", Lines: 3},
	}
	submit := func(values []string) (tea.Msg, error) {
		if values[0] == "bad" {
			return nil, errors.New("bad is not a number of shares")
		}
		return formMsg{values: values}, nil
	}

	return NewFormDialog("Sell shares", fields, submit, 40, DefaultStyle())
}

// typeIntoForm presses the keys of s in the form, one after the other.
func typeIntoForm(d *FormDialog, s string) {
	for _, msg := range keysPressed(s) {
		d.HandleKeyPress(msg.(tea.KeyPressMsg))
	}
}

// TestFormDialog_Fields covers getting around a form: it opens on its first field, what is typed
// goes into the field that has the focus, and tab and shift+tab move the focus on and back, around
// the ends. A field that was given a value starts out with it.
func TestFormDialog_Fields(t *testing.T) {
	d := newTestingForm()
	assert.Equal(t, 0, d.Focused())
	assert.Equal(t, []string{"", "", "2026-10-02", ""}, d.Values())

	typeIntoForm(d, "50")
	d.HandleKeyPress(tabKey)
	assert.Equal(t, 1, d.Focused())
	typeIntoForm(d, "410.2")
	assert.Equal(t, []string{"50", "410.2", "2026-10-02", ""}, d.Values())

	d.HandleKeyPress(shiftTabKey)
	assert.Equal(t, 0, d.Focused())
	d.HandleKeyPress(shiftTabKey)
	assert.Equal(t, 3, d.Focused(), "shift+tab on the first field should go around to the last")
	d.HandleKeyPress(tabKey)
	assert.Equal(t, 0, d.Focused(), "tab on the last field should go around to the first")
}

// TestFormDialog_Submit covers confirming a form: enter on a field of one line hands the values of
// all fields, without the space around them, to the function the form was made with and sends the
// message that returns. In a field of several lines enter starts a new line, as it does anywhere
// text of several lines is typed, and ctrl+s is what confirms, there and in every other field.
func TestFormDialog_Submit(t *testing.T) {
	d := newTestingForm()
	typeIntoForm(d, " 50 ")
	d.HandleKeyPress(tabKey)
	typeIntoForm(d, "410.2")

	cmd := d.HandleKeyPress(enterKey)
	assert.Equal(t, formMsg{values: []string{"50", "410.2", "2026-10-02", ""}}, runCmd(t, cmd))

	d.HandleKeyPress(tabKey)
	d.HandleKeyPress(tabKey)
	require.Equal(t, 3, d.Focused())
	typeIntoForm(d, "for the kitchen")
	// Enter does not confirm the form here: it starts the second line of the notes, which the
	// values that are confirmed below have.
	d.HandleKeyPress(enterKey)
	typeIntoForm(d, "in two orders")

	cmd = d.HandleKeyPress(saveKey)
	expected := formMsg{values: []string{"50", "410.2", "2026-10-02", "for the kitchen\nin two orders"}}
	assert.Equal(t, expected, runCmd(t, cmd))
}

// TestFormDialog_Refused covers values the form's function refuses: the form stays open with
// everything still in its fields, and says why under them, so that the value can be corrected.
func TestFormDialog_Refused(t *testing.T) {
	d := newTestingForm()
	typeIntoForm(d, "bad")

	cmd := d.HandleKeyPress(enterKey)

	assert.Nil(t, cmd, "refused values should send nothing")
	assert.Equal(t, "bad", d.Values()[0])
	assert.Contains(t, ansi.Strip(d.Layer().GetContent()), "bad is not a number of shares")
}

// TestFormDialog_Cancel covers the way out of a form without confirming it, which is esc, whatever
// has been typed. Ctrl+c quits the program.
func TestFormDialog_Cancel(t *testing.T) {
	d := newTestingForm()
	typeIntoForm(d, "50")

	cmd := d.HandleKeyPress(tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.Equal(t, FormCanceledMsg{}, runCmd(t, cmd))

	cmd = d.HandleKeyPress(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))
}

// TestFormDialog_Layer covers what the form looks like: its title, then every field after its
// label, the labels in a column of their own so that the fields line up, and a field of several
// lines as tall as it was asked to be.
func TestFormDialog_Layer(t *testing.T) {
	d := newTestingForm()
	typeIntoForm(d, "50")

	lines := strings.Split(ansi.Strip(d.Layer().GetContent()), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(strings.Trim(line, "│╭╮╰╯─"), " ")
	}

	// The border above and below, the title, three fields of one line and one of three.
	require.Len(t, lines, 9)
	assert.Equal(t, " Sell shares", lines[1])
	assert.Equal(t, " Shares  50", lines[2])
	assert.True(t, strings.HasPrefix(lines[3], " Price   "), lines[3])
	assert.Equal(t, " Date    2026-10-02", lines[4])
	assert.True(t, strings.HasPrefix(lines[5], " Notes"), lines[5])
}
