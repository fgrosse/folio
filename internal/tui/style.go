package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Style holds every lipgloss style the TUI renders with, so the look is defined in one place.
type Style struct {
	// Table is the base style of a view's table.
	Table lipgloss.Style

	TableBorder lipgloss.Border

	TableBorderColor color.Color

	Selected lipgloss.Style

	// Tab is a view in the tab bar that is not on display. It is there to be found,
	// not read, so it stays dim.
	Tab lipgloss.Style

	// TabSelected is the view on display, in the accent of the table's selection.
	TabSelected lipgloss.Style

	// Total is the total account value, the one number worth glancing at, so it is the only thing
	// that is gold and bold.
	Total lipgloss.Style

	// Value is what something that is held is worth, where that is the number its surroundings
	// lead up to, such as in the details of a lot. It is a part of the total, so it has the total's
	// gold, without the weight.
	Value lipgloss.Style

	// Gain and Loss are what a value has moved by since it was acquired, in the colors that mean
	// up and down wherever stock is shown.
	Gain lipgloss.Style
	Loss lipgloss.Style

	// Hint is what the header says besides the total: the two values the total is made of, and how
	// old the prices are. It is a footnote to the total, so it stays dim.
	Hint lipgloss.Style

	// Error is what went wrong, such as a quote that could not be fetched, in the header and under
	// the field of a dialog.
	Error lipgloss.Style

	// Dialog frames the editor that floats in front of the table while a lot or grant is being
	// entered.
	Dialog lipgloss.Style

	// DialogTitle says what the dialog is for, since the field itself only shows a placeholder.
	DialogTitle lipgloss.Style
}

// DefaultStyle returns the Style the TUI uses unless it is given another.
func DefaultStyle() Style {
	border := lipgloss.NormalBorder()
	borderColor := lipgloss.Color("240")
	accent := lipgloss.Color("99") // in the same purple family as the table's selection color
	gold := lipgloss.Color("220")  // for the total, which the eye should land on first, and its parts
	return Style{
		Table: lipgloss.NewStyle().
			BorderStyle(border).
			BorderForeground(borderColor),
		TableBorder:      border,
		TableBorderColor: borderColor,
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false),
		Tab: lipgloss.NewStyle().
			Foreground(borderColor),
		TabSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),
		Total: lipgloss.NewStyle().
			Bold(true).
			Foreground(gold),
		Value: lipgloss.NewStyle().
			Foreground(gold),
		Gain: lipgloss.NewStyle().
			Foreground(lipgloss.Color("78")),
		Loss: lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")),
		Hint: lipgloss.NewStyle().
			Foreground(borderColor),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")),
		Dialog: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1),
		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),
	}
}
