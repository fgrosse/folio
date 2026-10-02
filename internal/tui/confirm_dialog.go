package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ConfirmDialog asks a yes-or-no question before something happens that cannot be taken back, such
// as deleting a lot. It does not do that thing itself: a yes hands the parent the message it was
// created with, and a no hands it a ConfirmCanceledMsg.
type ConfirmDialog struct {
	title    string
	question string
	confirm  tea.Msg
	style    Style
}

// ConfirmCanceledMsg reports that a ConfirmDialog was answered with no.
type ConfirmCanceledMsg struct{}

// NewConfirmDialog returns a dialog that shows title and question, and sends confirm if the answer
// is yes. style is what Layer draws the dialog's frame and title with.
func NewConfirmDialog(title, question string, confirm tea.Msg, style Style) *ConfirmDialog {
	return &ConfirmDialog{
		title:    title,
		question: question,
		confirm:  confirm,
		style:    style,
	}
}

// HandleKeyPress reacts to a key pressed while the dialog is open and returns the command, if any,
// that the parent should run. Only y answers yes. N and esc answer no, ctrl+c quits the program, and
// every other key - enter included, which is pressed out of habit - does nothing.
func (d *ConfirmDialog) HandleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "y":
		confirm := d.confirm
		return func() tea.Msg { return confirm }

	case "n", "esc":
		return func() tea.Msg { return ConfirmCanceledMsg{} }

	case "ctrl+c":
		return tea.Quit
	}

	return nil
}

// Layer renders the dialog - title, question and border - as a compositor layer, for the parent to
// position over its own view.
func (d *ConfirmDialog) Layer() *lipgloss.Layer {
	dialog := d.style.DialogTitle.Render(d.title)
	dialog += "\n"
	dialog += d.question
	dialog = d.style.Dialog.Render(dialog)
	return lipgloss.NewLayer(dialog)
}
