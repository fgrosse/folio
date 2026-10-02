package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/shopspring/decimal"

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

// quoteStatus renders the price of each of symbols and how it moved since the previous close, such
// as "PANW $396.25 ▼ 0.3%". A symbol that quotes has none of says so instead.
func quoteStatus(symbols []string, quotes map[string]portfolio.Quote) string {
	parts := make([]string, len(symbols))
	for i, symbol := range symbols {
		quote, ok := quotes[symbol]
		if !ok {
			parts[i] = symbol + " has no price"
			continue
		}

		parts[i] = symbol + " " + portfolio.FormatUSD(quote.Price) + dayChange(quote)
	}

	return strings.Join(parts, " · ")
}

// dayChange renders how far the price of quote is from the previous close, as an arrow and a
// percentage to follow the price, or nothing if the quote has no previous close.
func dayChange(quote portfolio.Quote) string {
	if quote.PreviousClose.IsZero() {
		return ""
	}

	change := quote.Price.Sub(quote.PreviousClose).Div(quote.PreviousClose).Mul(decimal.NewFromInt(100))
	arrow := "▲"
	if change.IsNegative() {
		arrow = "▼"
	}

	return " " + arrow + " " + change.Abs().StringFixed(1) + "%"
}

// spread renders left and right at either end of a header line of the given width, indented like
// every other view's content.
func spread(left, right string, width int) string {
	// At least one space, so a window too narrow for both halves runs them together rather than
	// overlapping them.
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)

	return viewIndent + left + strings.Repeat(" ", gap) + right
}
