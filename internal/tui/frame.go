package tui

import (
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
	// failure of the two - the table skips zero-width columns entirely.
	minTableWidth = 60

	// maxTableWidth stops the table from spanning an ultra-wide display, where the eye has to
	// travel the whole screen to get from the start of a row to its value.
	maxTableWidth = 120
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

// tableWidth is how wide the table should be inside a terminal of the given width, clamped so it
// neither overflows a small window nor stretches across a very large one.
func tableWidth(terminalWidth int) int {
	return min(max(terminalWidth-borderWidth, minTableWidth), maxTableWidth)
}
