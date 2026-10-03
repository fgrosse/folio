package tui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/fgrosse/folio/internal/portfolio"
)

// rowMark stands in front of the row of the selected key.
const rowMark = "> "

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
	editing  bool              // whether the value of the selected key is being typed into input
	input    textinput.Model   // the field the value of the selected key is typed into
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
//
// On a key that takes any text, enter turns the value into a field to type it into. While that is
// open, enter asks for what it holds to be set, and every other key goes to the field, so that
// letters such as "j" type rather than move the selection.
func (d *ConfigDialog) HandleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if d.editing {
		return d.handleEditKeyPress(msg)
	}

	choices := len(d.keys[d.selected].Choices) > 0

	switch key := msg.String(); {
	case key == "down", key == "j":
		d.selectKey(d.selected + 1)
	case key == "up", key == "k":
		d.selectKey(d.selected - 1)
	case key == "enter" && !choices:
		return d.editCmd()
	case key == "right", key == "enter", key == "space":
		return d.chooseCmd(1)
	case key == "left":
		return d.chooseCmd(-1)
	}

	return nil
}

// handleEditKeyPress reacts to a key pressed while the value of the selected key is being typed.
func (d *ConfigDialog) handleEditKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if msg.String() == "enter" {
		return d.submitCmd()
	}

	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

// editCmd opens the field that the value of the selected key is typed into, with the value the key
// is set to in it, and returns the command that starts its cursor blinking.
func (d *ConfigDialog) editCmd() tea.Cmd {
	d.input = textinput.New()
	d.input.Prompt = ""
	// As in InputDialog: the field draws a virtual cursor after padding to its width, so it is
	// configured one column narrower than it renders.
	d.input.SetWidth(d.valueWidth() - 1)
	d.input.SetValue(d.values[d.keys[d.selected].Name])
	d.input.CursorEnd()
	d.editing = true

	return d.input.Focus()
}

// submitCmd ends the edit and returns the command that asks for the selected key to be set to what
// was typed, written the way the key stores it. The field is read right away rather than inside
// the returned command, as in InputDialog.
func (d *ConfigDialog) submitCmd() tea.Cmd {
	key := d.keys[d.selected]

	value, err := key.Parse(strings.TrimSpace(d.input.Value()))
	if err != nil {
		return nil
	}

	d.editing = false
	set := SetConfigMsg{key: key.Name, value: value}

	return func() tea.Msg { return set }
}

// Editing reports whether the value of the selected key is being typed, which is when esc and enter
// are about that value rather than about the dialog.
func (d *ConfigDialog) Editing() bool {
	return d.editing
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

	mark, nameStyle := strings.Repeat(" ", len(rowMark)), d.style.Hint
	if index == d.selected {
		mark, nameStyle = rowMark, d.style.DialogTitle
	}

	padding := strings.Repeat(" ", d.nameWidth()-lipgloss.Width(key.Name)+labelGap)

	value := d.valueView(key)
	if d.editing && index == d.selected {
		// The field pads its view to its width, which truncating keeps it to.
		value = ansi.Truncate(d.input.View(), d.valueWidth(), "")
	}

	return mark + nameStyle.Render(key.Name) + padding + value
}

// valueWidth is how many columns of a row are left for the value, after the mark and the names.
func (d *ConfigDialog) valueWidth() int {
	return d.width - len(rowMark) - d.nameWidth() - labelGap
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
