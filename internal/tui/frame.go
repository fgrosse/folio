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

	// minTableWidth is the point at which the table stops shrinking. A window narrower than this
	// wraps rather than squeezing the flexible column away to nothing, which is the more useful
	// failure of the two - the table skips zero-width columns entirely.
	minTableWidth = 60
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
