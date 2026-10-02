package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/table"

	"github.com/fgrosse/folio/internal/portfolio"
)

// Column widths of the Holdings table. Every column except the symbol is bounded by its own
// content, so the symbol is the one that flexes to fill whatever room the window leaves.
const (
	// sharesColumnWidth fits a number of shares up to 9,999,999, or fewer with a fraction.
	sharesColumnWidth = 10

	// priceColumnWidth fits the price of one share up to $99,999.99.
	priceColumnWidth = 10

	// valueColumnWidth fits a value up to $9,999,999.99, which is as much as the header's total
	// will ever have to add up.
	valueColumnWidth = 14

	// noValue stands where a price or value would be if there was a quote to work it out from.
	noValue = "-"
)

// lotRow renders a lot as a row of the Holdings table, valued at quote, which is the zero Quote if
// there is none of the lot's stock. The numbers are right-aligned for their digits to line up down
// the column. The table has no alignment of its own, so the values are padded out here.
func lotRow(lot portfolio.Lot, quote portfolio.Quote) table.Row {
	price, value := noValue, noValue
	if quote.Symbol != "" {
		price = portfolio.FormatUSD(quote.Price)
		value = portfolio.FormatUSD(lot.Shares.Mul(quote.Price))
	}

	return table.Row{
		lot.Acquired.Format(time.DateOnly),
		lot.Symbol,
		fmt.Sprintf("%*s", sharesColumnWidth, lot.Shares),
		fmt.Sprintf("%*s", priceColumnWidth, price),
		fmt.Sprintf("%*s", valueColumnWidth, value),
	}
}
