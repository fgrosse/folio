package tui

import (
	"strings"

	"charm.land/bubbles/v2/table"
)

// The measures of the frame every view renders in: two header lines, a table in a box, and two help
// lines. They are the same in every view, so that switching tabs moves nothing but what is in the
// table.
const (
	// cellPadding is the single space the table renders either side of every cell.
	cellPadding = 2

	// borderWidth is the frame the style draws around the table.
	borderWidth = 2

	// chromeHeight is everything the window spends on something other than rows: the two header
	// lines above the table, the border above and below, and the two help lines underneath. The
	// table's own header and the rule under it come out of the height it is given, not out of this.
	chromeHeight = 6

	// minTableWidth is the point at which the table stops shrinking. A window narrower than this
	// wraps rather than squeezing the flexible column away to nothing, which is the more useful
	// failure of the two - the table skips zero-width columns entirely. It is as wide as it is for
	// the Holdings table, whose number columns alone take most of it.
	minTableWidth = 80

	// maxTableWidth stops the table from spanning an ultra-wide display, where the eye has to
	// travel the whole screen to get from the start of a row to its value.
	maxTableWidth = 120

	// dialogWidth is how many columns the text field of a dialog occupies. A grant spec is a long
	// line, and this fits one with a name of some length.
	dialogWidth = 56

	// flyoutWidth is how many columns a flyout occupies, border included. It is half of the
	// narrowest table, which leaves the columns that say which row is selected in sight, and fits a
	// label next to a value in the millions.
	flyoutWidth = minTableWidth / 2
)

// newTable returns a focused table with the given columns, styled the way every table in the TUI is:
// a rule under the headings, and the selected row in the selection colors of style.
func newTable(style Style, cols []table.Column) table.Model {
	t := table.New(table.WithFocused(true), table.WithColumns(cols))

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(style.TableBorder).
		BorderForeground(style.TableBorderColor).
		BorderBottom(true).
		Bold(false)
	ts.Selected = style.Selected.Inherit(ts.Selected)
	t.SetStyles(ts)

	return t
}

// setRows gives the table its rows and sees to it that one of them is selected. A table that is
// given no rows moves its selection off the end of them, to before the first, and leaves it there
// when rows arrive, where the keys that act on the selected row would find none.
func setRows(t *table.Model, rows []table.Row) {
	t.SetRows(rows)
	if t.Cursor() < 0 && len(rows) > 0 {
		t.SetCursor(0)
	}
}

// tableWidth is how wide the table should be inside a terminal of the given width, clamped so it
// neither overflows a small window nor stretches across a very large one.
func tableWidth(terminalWidth int) int {
	return min(max(terminalWidth-borderWidth, minTableWidth), maxTableWidth)
}

// trimTrailingSpace strips the padding the compositor leaves behind. It draws onto a grid as wide
// as the widest line in the frame - the table's box - and fills everything a shorter line does not
// reach with real spaces, so the header and the help line come back padded out to the full width.
// None of it is visible on a terminal. It is visible in the golden files, where trailing whitespace
// is both invisible to a reviewer and liable to be stripped by an editor.
func trimTrailingSpace(frame string) string {
	lines := strings.Split(frame, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}

	return strings.Join(lines, "\n")
}
