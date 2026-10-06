package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// portfolioHeader renders the two lines above the table of a view from the portfolio it shows: left
// is the view's own line, with the prices the portfolio is valued at under it, and the account
// values are on the right. If the portfolio could not be loaded or its quotes not refreshed, err
// says why, and stands where the prices otherwise are.
func portfolioHeader(left string, p Portfolio, err error, width int, style Style) string {
	return notedHeader(left, "", p, err, width, style)
}

// notedHeader renders the header of a portfolio like portfolioHeader, with note in the place of the
// prices unless it is empty. A view puts there what it has to say about the row that is selected.
// An error still comes first, and prices or a note too long for the room left of the account values
// are cut short.
func notedHeader(left, note string, p Portfolio, err error, width int, style Style) string {
	values := shownAccount(p)

	// The values on the right are about as wide as their labels and two amounts, and what is left
	// of them is all the room the prices or the note have.
	room := width - lipgloss.Width(accountParts(values)) - headerGap

	status := style.Hint.Render(ansi.Truncate(quoteStatus(portfolio.Symbols(p.Lots, p.Grants), p.Quotes), room, "…"))
	switch {
	case err != nil:
		// Errors of several quotes come joined by newlines, and the header has one line for them.
		status = style.Error.Render(strings.ReplaceAll(err.Error(), "\n", " · "))
	case note != "":
		status = style.Hint.Render(ansi.Truncate(note, room, "…"))
	}

	return accountHeader(left, status, values, width, style)
}

// shownValues are the values of an account as the header shows them, and whether the current and
// the potential one are before or after tax.
type shownValues struct {
	account   portfolio.Account
	current   portfolio.Basis
	potential portfolio.Basis
}

// shownAccount returns the values of p that the header shows. Each of the two the total is made of
// is after tax only if the account asks for that and has a rate to take off, and the bank's number
// otherwise, so that the header never calls a value net that is not. The two have a tax each: a
// vest is taxed as income, and what is held on the gain it made since.
func shownAccount(p Portfolio) shownValues {
	values := shownValues{
		account:   portfolio.NewAccount(p.Lots, p.Grants, p.Quotes),
		current:   portfolio.Gross,
		potential: portfolio.Gross,
	}

	if p.CurrentBasis == portfolio.Net && p.GainsTaxRate.Valid {
		values.account = values.account.AfterGainsTax(p.GainsTaxRate.Decimal)
		values.current = portfolio.Net
	}
	if p.PotentialBasis == portfolio.Net && p.TaxRate.Valid {
		values.account = values.account.AfterTax(p.TaxRate.Decimal)
		values.potential = portfolio.Net
	}

	return values
}

// headerGap is the least space between the two halves of a header line.
const headerGap = 2

// accountParts renders the two values the total is made of, for the second line of the header. The
// potential value says whether it is gross or net, since the two differ by the tax on the vests and
// the header is where the user compares folio with the bank. The current value only says so when
// it is net: it is the bank's unless the account asks otherwise, and the line has the prices to fit
// in as well.
func accountParts(values shownValues) string {
	current := "Current"
	if values.current == portfolio.Net {
		current += " (net)"
	}

	return fmt.Sprintf("%s %s · Potential (%s) %s",
		current, portfolio.FormatUSD(values.account.Current),
		values.potential, portfolio.FormatUSD(values.account.Potential))
}

// accountHeader renders the two lines above the table of a view. The right half is the same in
// every view and is what folio is for: the total account value on the first line, in the one accent
// of the header, and the current and potential value it is made of underneath, marked as before
// or after tax. The left half is the view's own: what it has to say about the table below,
// and a status line under it.
//
// width is that of the table without the padding of its cells, so that the values end where the
// last column's do.
func accountHeader(left, status string, values shownValues, width int, style Style) string {
	total := style.Total.Render("Total: " + portfolio.FormatUSD(values.account.Total()))
	parts := style.Hint.Render(accountParts(values))

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
