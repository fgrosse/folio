package tui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
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
	err      error             // why the last value was refused or not stored, nil unless it was
	help     help.Model        // renders the keys at the foot of the dialog
	width    int               // how many columns the dialog has inside its border
	style    Style
}

// SetConfigMsg asks for the key of the configuration to be set to value, which the key has checked
// and written the way it is stored.
type SetConfigMsg struct {
	key   string
	value string
}

// UnsetConfigMsg asks for the value of the key of the configuration to be taken back.
type UnsetConfigMsg struct {
	key string
}

// ConfigClosedMsg reports that a ConfigDialog was closed.
type ConfigClosedMsg struct{}

// NewConfigDialog returns a dialog over keys, the first of which is selected. values is what the
// keys are set to, by name, without the ones that are not set. width is how many columns the
// dialog occupies inside its border, and style is what Layer draws it with.
func NewConfigDialog(keys []portfolio.ConfigKey, values map[string]string, width int, style Style) *ConfigDialog {
	return &ConfigDialog{
		keys:   keys,
		values: values,
		width:  width,
		help:   help.New(),
		style:  style,
	}
}

// Update passes a message on to the dialog, key presses by way of HandleKeyPress. Anything else goes
// to the field while a value is being typed, which is how the ticks that make its cursor blink
// reach it.
func (d *ConfigDialog) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		return d.HandleKeyPress(msg)
	}

	if !d.editing {
		return nil
	}

	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

// HandleKeyPress reacts to a key pressed while the dialog is open and returns the command, if any,
// that the parent should run. Down and up select the next key and the one before, around the ends,
// and so do j and k, as in the tables. On a key that takes one of a few values, right and left ask
// for the next of them and the one before to be set, and so do enter and space for the next. Esc
// closes the dialog with a ConfigClosedMsg, and ctrl+c quits the program.
//
// On a key that takes any text, enter turns the value into a field to type it into, and
// handleEditKeyPress has the keys while that is open.
func (d *ConfigDialog) HandleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if d.editing {
		return d.handleEditKeyPress(msg)
	}

	choices := len(d.keys[d.selected].Choices) > 0

	switch pressed := msg.String(); {
	case pressed == "down", pressed == "j":
		d.selectKey(d.selected + 1)
	case pressed == "up", pressed == "k":
		d.selectKey(d.selected - 1)
	case pressed == "enter" && !choices:
		return d.editCmd()
	case pressed == "right", pressed == "enter", pressed == "space":
		return d.chooseCmd(1)
	case pressed == "left":
		return d.chooseCmd(-1)
	case pressed == "esc":
		return func() tea.Msg { return ConfigClosedMsg{} }
	case pressed == "ctrl+c":
		return tea.Quit
	}

	return nil
}

// handleEditKeyPress reacts to a key pressed while the value of the selected key is being typed.
// Enter asks for what the field holds to be set, or for the key to be unset if it holds nothing. A
// value that the key refuses keeps the field open, and the dialog shows why under the description.
// Esc drops what was typed and leaves the key as it was, and ctrl+c quits the program. Every other
// key goes to the field, so that letters such as "j" type rather than move the selection.
func (d *ConfigDialog) handleEditKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		return d.submitCmd()
	case "esc":
		d.err = nil
		d.editing = false
		return nil
	case "ctrl+c":
		return tea.Quit
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
	// An empty field unsets the key, so it shows what the row will say then.
	d.input.Placeholder = notSet
	// As in InputDialog: the field draws a virtual cursor after padding to its width, so it is
	// configured one column narrower than it renders.
	d.input.SetWidth(d.valueWidth() - 1)
	d.input.SetValue(d.values[d.keys[d.selected].Name])
	d.input.CursorEnd()
	d.editing = true

	return d.input.Focus()
}

// submitCmd ends the edit and returns the command that asks for the selected key to be set to what
// was typed, written the way the key stores it, or to be unset if nothing was. The field is read right away rather than inside
// the returned command, as in InputDialog.
func (d *ConfigDialog) submitCmd() tea.Cmd {
	k := d.keys[d.selected]

	typed := strings.TrimSpace(d.input.Value())
	if typed == "" {
		// Nothing is no value of any key, so it is how to say that the key should have none.
		d.err = nil
		d.editing = false
		unset := UnsetConfigMsg{key: k.Name}

		return func() tea.Msg { return unset }
	}

	value, err := k.Parse(typed)
	if err != nil {
		// Stay in the field with the text still in it, so the user can correct it.
		d.err = err
		return nil
	}

	d.err = nil
	d.editing = false
	set := SetConfigMsg{key: k.Name, value: value}

	return func() tea.Msg { return set }
}

// SetError has the dialog say that err went wrong, such as storing a value, where it otherwise says
// why a value was refused. A nil err takes back what it said.
func (d *ConfigDialog) SetError(err error) {
	d.err = err
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
	k := d.keys[d.selected]
	n := len(k.Choices)
	if n == 0 {
		return nil
	}

	// A value that is none of the choices counts as the one before the first, so that right picks
	// the first of them.
	current, _ := d.value(k)
	next := (slices.Index(k.Choices, current) + step + n) % n
	set := SetConfigMsg{key: k.Name, value: k.Choices[next]}

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

// Layer renders the whole dialog - title, the rows of the keys, what the selected key is for, why
// the last value was refused, the keys, and border - as a compositor layer, for the parent to position over its own view.
func (d *ConfigDialog) Layer() *lipgloss.Layer {
	lines := []string{d.style.DialogTitle.Render("Configuration")}
	for i := range d.keys {
		lines = append(lines, d.rowView(i))
	}

	// The description takes the width of the dialog, on as many lines as that needs.
	description := lipgloss.NewStyle().Width(d.width).Render(d.keys[d.selected].Description)
	lines = append(lines, "", d.style.Hint.Render(description))

	if d.err != nil {
		lines = append(lines, d.style.Error.Width(d.width).Render(d.err.Error()))
	}

	lines = append(lines, "", d.helpView())

	return lipgloss.NewLayer(d.style.Dialog.Render(strings.Join(lines, "\n")))
}

// helpView renders the keys worth knowing for the row that is selected. A view shows the keys of
// its dialog in its own help lines, under the table. This dialog is the app's and has no such
// lines, so it carries its keys itself.
func (d *ConfigDialog) helpView() string {
	if d.editing {
		return d.help.ShortHelpView([]key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		})
	}

	change := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit"))
	if len(d.keys[d.selected].Choices) > 0 {
		change = key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "change"))
	}

	return d.help.ShortHelpView([]key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		change,
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
	})
}

// rowView renders the row of the key at index: a mark if it is the selected one, its name, and its
// value. The names are padded to the longest of them, so that the values line up in a column.
func (d *ConfigDialog) rowView(index int) string {
	k := d.keys[index]

	mark, nameStyle := strings.Repeat(" ", len(rowMark)), d.style.Hint
	if index == d.selected {
		mark, nameStyle = rowMark, d.style.DialogTitle
	}

	padding := strings.Repeat(" ", d.nameWidth()-lipgloss.Width(k.Name)+labelGap)

	value := d.valueView(k)
	if d.editing && index == d.selected {
		// The field pads its view to its width, which truncating keeps it to.
		value = ansi.Truncate(d.input.View(), d.valueWidth(), "")
	}

	return mark + nameStyle.Render(k.Name) + padding + value
}

// valueWidth is how many columns of a row are left for the value, after the mark and the names.
func (d *ConfigDialog) valueWidth() int {
	return d.width - len(rowMark) - d.nameWidth() - labelGap
}

// nameWidth is how many columns the longest name of a key takes.
func (d *ConfigDialog) nameWidth() int {
	width := 0
	for _, k := range d.keys {
		width = max(width, lipgloss.Width(k.Name))
	}

	return width
}

// valueView renders what key is set to. A key that takes one of a few values has it between the
// arrows that change it, and a key without a value says so, dimmed, since that is not a value.
func (d *ConfigDialog) valueView(k portfolio.ConfigKey) string {
	value, ok := d.value(k)
	switch {
	case !ok:
		return d.style.Hint.Render(notSet)
	case len(k.Choices) > 0:
		return "‹ " + value + " ›"
	default:
		return value
	}
}

// value returns what applies for key: the value it is set to, or else its default. It reports
// false for a key that has neither.
func (d *ConfigDialog) value(k portfolio.ConfigKey) (string, bool) {
	if value, ok := d.values[k.Name]; ok {
		return value, true
	}

	return k.Default, k.Default != ""
}
