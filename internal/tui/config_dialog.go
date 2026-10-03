package tui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/fgrosse/folio/internal/portfolio"
)

// notSet is what the dialog shows as the value of a key that has none and no default either.
const notSet = "not set"

// ConfigDialog is the floating editor of the configuration of the account: one row for each key,
// with the value it is set to, and what the selected key is for under the rows. It belongs to the
// app rather than to one of its views, since the configuration applies to all of them, and floats
// in front of whichever view is on display, where a changed value shows right away. Like the other
// dialogs it never touches the store itself.
type ConfigDialog struct {
	keys     []portfolio.ConfigKey
	values   map[string]string // what each key is set to, by its name, without the keys that are not set
	selected int               // the index of the selected key
	width    int               // how many columns the dialog has inside its border
	style    Style
}

// SetConfigMsg asks for the key of the configuration to be set to value, which the key has checked
// and written the way it is stored.
type SetConfigMsg struct {
	key   string
	value string
}

// NewConfigDialog returns a dialog over keys, the first of which is selected. values is what the
// keys are set to, by name, without the ones that are not set. width is how many columns the
// dialog occupies inside its border, and style is what Layer draws it with.
func NewConfigDialog(keys []portfolio.ConfigKey, values map[string]string, width int, style Style) *ConfigDialog {
	return &ConfigDialog{
		keys:   keys,
		values: values,
		width:  width,
		style:  style,
	}
}

// HandleKeyPress reacts to a key pressed while the dialog is open and returns the command, if any,
// that the parent should run. Down and up select the next key and the one before, around the ends,
// and so do j and k, as in the tables. On a key that takes one of a few values, right and left ask
// for the next of them and the one before to be set, and so do enter and space for the next.
func (d *ConfigDialog) HandleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "down", "j":
		d.selectKey(d.selected + 1)
	case "up", "k":
		d.selectKey(d.selected - 1)
	case "right", "enter", "space":
		return d.chooseCmd(1)
	case "left":
		return d.chooseCmd(-1)
	}

	return nil
}

// chooseCmd returns the command that asks for the selected key to be set to the choice that is
// step choices on from the one that applies now, counted around the ends. It is nil for a key that
// does not have choices. The dialog goes on showing the value it has until SetValues gives it the
// one that was stored, so that it never shows a value the store did not take.
func (d *ConfigDialog) chooseCmd(step int) tea.Cmd {
	key := d.keys[d.selected]
	n := len(key.Choices)
	if n == 0 {
		return nil
	}

	// A value that is none of the choices counts as the one before the first, so that right picks
	// the first of them.
	current, _ := d.value(key)
	next := (slices.Index(key.Choices, current) + step + n) % n
	set := SetConfigMsg{key: key.Name, value: key.Choices[next]}

	return func() tea.Msg { return set }
}

// SetValues gives the dialog what the keys are set to now, in place of what it had, by name and
// without the keys that are not set.
func (d *ConfigDialog) SetValues(values map[string]string) {
	d.values = values
}

// selectKey selects the key at index, counted around the ends.
func (d *ConfigDialog) selectKey(index int) {
	n := len(d.keys)
	// Adding n first keeps the left side of the % positive, as in AppModel.cycleView.
	d.selected = (index + n) % n
}

// Layer renders the whole dialog - title, the rows of the keys, what the selected key is for, and
// border - as a compositor layer, for the parent to position over its own view.
func (d *ConfigDialog) Layer() *lipgloss.Layer {
	lines := []string{d.style.DialogTitle.Render("Configuration")}
	for i := range d.keys {
		lines = append(lines, d.rowView(i))
	}

	// The description takes the width of the dialog, on as many lines as that needs.
	description := lipgloss.NewStyle().Width(d.width).Render(d.keys[d.selected].Description)
	lines = append(lines, "", d.style.Hint.Render(description))

	return lipgloss.NewLayer(d.style.Dialog.Render(strings.Join(lines, "\n")))
}

// rowView renders the row of the key at index: a mark if it is the selected one, its name, and its
// value. The names are padded to the longest of them, so that the values line up in a column.
func (d *ConfigDialog) rowView(index int) string {
	key := d.keys[index]

	mark, nameStyle := "  ", d.style.Hint
	if index == d.selected {
		mark, nameStyle = "> ", d.style.DialogTitle
	}

	padding := strings.Repeat(" ", d.nameWidth()-lipgloss.Width(key.Name)+labelGap)

	return mark + nameStyle.Render(key.Name) + padding + d.valueView(key)
}

// nameWidth is how many columns the longest name of a key takes.
func (d *ConfigDialog) nameWidth() int {
	width := 0
	for _, key := range d.keys {
		width = max(width, lipgloss.Width(key.Name))
	}

	return width
}

// valueView renders what key is set to. A key that takes one of a few values has it between the
// arrows that change it, and a key without a value says so, dimmed, since that is not a value.
func (d *ConfigDialog) valueView(key portfolio.ConfigKey) string {
	value, ok := d.value(key)
	switch {
	case !ok:
		return d.style.Hint.Render(notSet)
	case len(key.Choices) > 0:
		return "‹ " + value + " ›"
	default:
		return value
	}
}

// value returns what applies for key: the value it is set to, or else its default. It reports
// false for a key that has neither.
func (d *ConfigDialog) value(key portfolio.ConfigKey) (string, bool) {
	if value, ok := d.values[key.Name]; ok {
		return value, true
	}

	return key.Default, key.Default != ""
}
