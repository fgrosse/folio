package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/fgrosse/folio/internal/portfolio"
)

// accountHeader renders the two lines above the table of a view. The right half is the same in
// every view and is what folio is for: the total account value on the first line, in the one accent
// of the header, and the current and potential value it is made of underneath. The left half is the
// view's own: what it has to say about the table below, and a status line under it.
//
// width is that of the table without the padding of its cells, so that the values end where the
// last column's do.
func accountHeader(left, status string, account portfolio.Account, width int, style Style) string {
	total := style.Total.Render("Total: " + portfolio.FormatUSD(account.Total()))
	parts := style.Hint.Render(
		"Current " + portfolio.FormatUSD(account.Current) + " · Potential " + portfolio.FormatUSD(account.Potential),
	)

	return spread(left, total, width) + "\n" + spread(status, parts, width)
}

// spread renders left and right at either end of a header line of the given width, indented like
// every other view's content.
func spread(left, right string, width int) string {
	// At least one space, so a window too narrow for both halves runs them together rather than
	// overlapping them.
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)

	return viewIndent + left + strings.Repeat(" ", gap) + right
}
