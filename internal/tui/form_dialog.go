package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// labelGap is the space between the column of labels and the fields of a form.
const labelGap = 2

// FormDialog is the floating editor for something that is more than one line of text can say, such
// as a sale: several fields, each after its label, one of which has the focus and takes what is
// typed. Like InputDialog it knows nothing of what its values mean. Confirming hands them to the
// function the form was made with, which either turns them into the message to send to the parent
// or says why they will not do, and esc reports a FormCanceledMsg.
type FormDialog struct {
	title      string
	fields     []formField
	focused    int
	width      int                                    // how many columns a field renders in, cursor included
	labelWidth int                                    // how many columns the longest label takes
	submit     func(values []string) (tea.Msg, error) // what confirming does with the values
	err        error                                  // why the last confirmation was refused, nil unless it was
	style      Style
}

// A FormField describes one field of a FormDialog.
type FormField struct {
	// Label says what the field is for, in front of it.
	Label string

	// Placeholder is shown in the field while it is empty.
	Placeholder string

	// Value is what the field holds when the form opens.
	Value string

	// Lines is how many lines the field takes up. A field of more than one line takes text of
	// several lines, in which enter starts a new line.
	Lines int
}

// formField is a field of a form as it is edited: a text input for a field of one line, and a text
// area for one of several.
type formField struct {
	label     string
	multiline bool
	input     textinput.Model
	area      textarea.Model
}

// FormCanceledMsg reports that a FormDialog was closed without confirming anything.
type FormCanceledMsg struct{}

// NewFormDialog returns a dialog under title with the given fields, the first of which has the
// focus. submit is what confirming hands the values of the fields to, in the order of the fields. It
// returns the message for the parent, or why the values are refused, which keeps the dialog open to
// say so. width is how many columns a field occupies, cursor included, and style is what Layer draws
// the dialog with.
func NewFormDialog(title string, fields []FormField, submit func(values []string) (tea.Msg, error), width int, style Style) *FormDialog {
	d := &FormDialog{
		title:  title,
		width:  width,
		submit: submit,
		style:  style,
	}

	for _, field := range fields {
		d.labelWidth = max(d.labelWidth, lipgloss.Width(field.Label))
		d.fields = append(d.fields, newFormField(field, width))
	}

	d.focus(0)

	return d
}

func newFormField(field FormField, width int) formField {
	f := formField{label: field.Label, multiline: field.Lines > 1}

	if f.multiline {
		f.area = textarea.New()
		f.area.Prompt = ""
		f.area.Placeholder = field.Placeholder
		f.area.ShowLineNumbers = false
		// The area comes with a line of its own color under the cursor and a frame of styles that
		// a dialog with a border of its own has no use for.
		f.area.SetStyles(textarea.Styles{Cursor: f.area.Styles().Cursor})
		f.area.SetWidth(width)
		f.area.SetHeight(field.Lines)
		f.area.SetValue(field.Value)
		f.area.Blur()

		return f
	}

	f.input = textinput.New()
	f.input.Prompt = ""
	f.input.Placeholder = field.Placeholder
	// As in InputDialog: the field draws a virtual cursor after padding to its width, so it is
	// configured one column narrower than it renders.
	f.input.SetWidth(width - 1)
	f.input.SetValue(field.Value)
	f.input.CursorEnd()
	f.input.Blur()

	return f
}

// Init returns the command that starts the cursor blinking. The ticks that keep it blinking come
// back as messages, which the parent has to pass on through Update.
func (d *FormDialog) Init() tea.Cmd {
	return textinput.Blink
}

// Update passes a message on to the dialog, key presses by way of HandleKeyPress. Anything else goes
// to the field that has the focus, which is how the ticks that make its cursor blink reach it.
func (d *FormDialog) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		return d.HandleKeyPress(msg)
	}

	return d.updateFocused(msg)
}

// HandleKeyPress reacts to a key pressed while the dialog is open and returns the command, if any,
// that the parent should run.
//
// Tab and shift+tab move the focus to the next field and the one before, around the ends, and so do
// down and up in a field of one line. Enter confirms the form from a field of one line, and starts a
// new line in a field of several, where ctrl+s is what confirms. Ctrl+s confirms from any field.
// Values that are refused keep the dialog open, and it shows why under the fields. Esc cancels with
// a FormCanceledMsg, and ctrl+c quits the program. Every other key goes to the field that has the
// focus, so that letters such as "q" type rather than trigger a shortcut.
func (d *FormDialog) HandleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	multiline := d.fields[d.focused].multiline

	switch key := msg.String(); {
	case key == "esc":
		return func() tea.Msg { return FormCanceledMsg{} }
	case key == "ctrl+c":
		return tea.Quit
	case key == "ctrl+s", key == "enter" && !multiline:
		return d.submitCmd()
	case msg.Code == tea.KeyTab && msg.Mod == tea.ModShift, key == "up" && !multiline:
		return d.focus(d.focused - 1)
	case msg.Code == tea.KeyTab, key == "down" && !multiline:
		return d.focus(d.focused + 1)
	}

	return d.updateFocused(msg)
}

// OnMultiline reports whether the focus is on a field of several lines, where enter starts a new
// line rather than confirming the form. The parent's help says which key confirms.
func (d *FormDialog) OnMultiline() bool {
	return d.fields[d.focused].multiline
}

// Focused is the index of the field that has the focus.
func (d *FormDialog) Focused() int {
	return d.focused
}

// Values returns what the fields hold, in the order of the fields.
func (d *FormDialog) Values() []string {
	values := make([]string, len(d.fields))
	for i, field := range d.fields {
		if field.multiline {
			values[i] = field.area.Value()
		} else {
			values[i] = field.input.Value()
		}
	}

	return values
}

// focus moves the focus to the field at index, counted around the ends, and returns the command
// that starts its cursor blinking.
func (d *FormDialog) focus(index int) tea.Cmd {
	n := len(d.fields)
	// Adding n first keeps the left side of the % positive, as in AppModel.cycleView.
	index = (index + n) % n

	for i := range d.fields {
		d.fields[i].input.Blur()
		d.fields[i].area.Blur()
	}

	d.focused = index
	if d.fields[index].multiline {
		return d.fields[index].area.Focus()
	}

	return d.fields[index].input.Focus()
}

// updateFocused passes msg on to the field that has the focus.
func (d *FormDialog) updateFocused(msg tea.Msg) tea.Cmd {
	field := &d.fields[d.focused]

	var cmd tea.Cmd
	if field.multiline {
		field.area, cmd = field.area.Update(msg)
	} else {
		field.input, cmd = field.input.Update(msg)
	}

	return cmd
}

// submitCmd confirms the dialog with what its fields hold, without the space around each value. The
// fields are read and handed over right away rather than inside the returned command, which Bubble
// Tea runs on a goroutine of its own while further keys may still be changing them.
func (d *FormDialog) submitCmd() tea.Cmd {
	values := d.Values()
	for i, value := range values {
		values[i] = strings.TrimSpace(value)
	}

	msg, err := d.submit(values)
	if err != nil {
		// Stay open with everything still in the fields, so the user can correct it.
		d.err = err
		return nil
	}

	return func() tea.Msg { return msg }
}

// Layer renders the whole dialog - title, the fields after their labels, why the last confirmation
// was refused, and border - as a compositor layer, for the parent to position over its own view.
func (d *FormDialog) Layer() *lipgloss.Layer {
	lines := []string{d.style.DialogTitle.Render(d.title)}
	for i, field := range d.fields {
		lines = append(lines, d.fieldLines(i, field)...)
	}

	if d.err != nil {
		lines = append(lines, d.style.Error.Render(d.err.Error()))
	}

	return lipgloss.NewLayer(d.style.Dialog.Render(strings.Join(lines, "\n")))
}

// fieldLines renders the field at index after its label. The label is on the first line of a field
// of several, and the lines under it start where the first does. The label of the field that has
// the focus stands out.
func (d *FormDialog) fieldLines(index int, field formField) []string {
	labelStyle := d.style.Hint
	if index == d.focused {
		labelStyle = d.style.DialogTitle
	}

	view := field.input.View()
	if field.multiline {
		view = field.area.View()
	}

	labelColumn := d.labelWidth + labelGap
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		label := ""
		if i == 0 {
			label = field.label
		}

		// A field pads its view to its width, which truncating keeps it to.
		padding := strings.Repeat(" ", labelColumn-lipgloss.Width(label))
		lines[i] = labelStyle.Render(label) + padding + ansi.Truncate(line, d.width, "")
	}

	return lines
}
