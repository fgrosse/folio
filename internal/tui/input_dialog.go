package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// InputDialog is the floating editor a line of text is typed into, such as the spec of a new lot. It
// holds a single text field and knows nothing of what the text means: enter hands it to the function
// the dialog was made with, which either turns it into the message to send to the parent or says why
// it will not do. It reports a way out without a text through InputCanceledMsg, and never touches
// the store itself.
type InputDialog struct {
	title  string
	input  textinput.Model
	width  int                                 // how many columns the field renders in, cursor included
	submit func(value string) (tea.Msg, error) // what enter does with the text of the field
	err    error                               // why the last enter was refused, nil unless it was
	style  Style
}

// InputCanceledMsg reports that an InputDialog was closed without submitting anything.
type InputCanceledMsg struct{}

// NewInputDialog returns a dialog under title whose text field is empty, focused and ready for
// typing, and shows placeholder until something is typed. submit is what enter hands the text to. It
// returns the message for the parent, or why the text is refused, which keeps the dialog open to say
// so. width is how many columns the field occupies, cursor included, and style is what Layer draws
// the dialog's frame and title with.
func NewInputDialog(title, placeholder string, submit func(value string) (tea.Msg, error), width int, style Style) *InputDialog {
	d := &InputDialog{
		title:  title,
		input:  textinput.New(),
		width:  width,
		submit: submit,
		style:  style,
	}

	d.input.Prompt = ""
	d.input.Placeholder = placeholder
	// The field draws a virtual cursor into the string View returns, which is what lets it be
	// composited into a dialog layer - a real cursor, via Model.Cursor, would need the field's
	// absolute position on screen. View pads to the configured width and only then draws that
	// cursor, so the field is configured one column narrower than it renders.
	d.input.SetWidth(width - 1)
	d.input.Focus()

	return d
}

// Init returns the command that starts the cursor blinking. The ticks that keep it blinking come
// back as messages, which the parent has to pass on through Update.
func (d *InputDialog) Init() tea.Cmd {
	return textinput.Blink
}

// Update passes a message on to the dialog, key presses by way of HandleKeyPress. Anything else goes
// to the text field, which is how the ticks that make its cursor blink reach it.
func (d *InputDialog) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		return d.HandleKeyPress(msg)
	}

	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

// HandleKeyPress reacts to a key pressed while the dialog is open and returns the command, if any,
// that the parent should run. Enter submits the text, and on an empty field counts as canceling. A
// text that is refused keeps the dialog open instead, and the dialog shows why under the field. Esc
// cancels with an InputCanceledMsg, and ctrl+c quits the program. Every other key goes to the text
// field, so that letters such as "q" type rather than trigger a shortcut.
func (d *InputDialog) HandleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		return d.submitCmd()
	case "esc":
		return func() tea.Msg { return InputCanceledMsg{} }
	case "ctrl+c":
		return tea.Quit
	}

	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

// submitCmd confirms the dialog with what has been typed so far. The field is read and handed over
// right away rather than inside the returned command, which Bubble Tea runs on a goroutine of its own
// while further keys may still be changing the field.
func (d *InputDialog) submitCmd() tea.Cmd {
	value := strings.TrimSpace(d.input.Value())
	if value == "" {
		// Nothing was typed, so enter means "never mind".
		return func() tea.Msg { return InputCanceledMsg{} }
	}

	msg, err := d.submit(value)
	if err != nil {
		// Stay open with the text still in the field, so the user can correct it.
		d.err = err
		return nil
	}

	return func() tea.Msg { return msg }
}

// Value returns the text typed into the field so far.
func (d *InputDialog) Value() string {
	return d.input.Value()
}

// Layer renders the whole dialog - title, text field, why the last text was refused, and border - as
// a compositor layer, for the parent to position over its own view.
func (d *InputDialog) Layer() *lipgloss.Layer {
	dialog := d.style.DialogTitle.Render(d.title)
	dialog += "\n"
	// The field pads its view to its width, which truncating keeps it to.
	dialog += ansi.Truncate(d.input.View(), d.width, "")
	if d.err != nil {
		dialog += "\n"
		dialog += d.style.Error.Render(d.err.Error())
	}
	dialog = d.style.Dialog.Render(dialog)
	return lipgloss.NewLayer(dialog)
}
