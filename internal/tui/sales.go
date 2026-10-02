package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/table"
	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// realizedSummary sums up sales as the money they brought in and the gain in it, such as "Realized
// $20,510.00 · gain +$12,385.00". It is what the Sales view says above its table. The gain is that
// of the sales whose cost is known, and the summary says how many it leaves out.
func realizedSummary(sales []portfolio.Sale) string {
	if len(sales) == 0 {
		return "Nothing sold yet"
	}

	realized := portfolio.NewRealized(sales)
	summary := "Realized " + portfolio.FormatUSD(realized.Proceeds) + " · gain "

	switch {
	case realized.Uncosted == len(sales):
		return summary + "unknown"
	case realized.Uncosted > 0:
		return summary + formatGain(realized.Gain) + " without " + count(realized.Uncosted, "sale") + " of unknown cost"
	default:
		return summary + formatGain(realized.Gain)
	}
}

// saleRow renders a sale as a row of the Sales table: its day, the stock and the grant of the lot it
// was sold from, the shares and the price of one, and what the sale brought in and gained. The
// numbers are right-aligned and padded out like those of the other tables.
func saleRow(sale portfolio.Sale) table.Row {
	gain := noValue
	if amount, known := sale.Gain(); known {
		gain = formatGain(amount)
	}

	return table.Row{
		sale.Date.Format(time.DateOnly),
		sale.Symbol,
		sale.Grant,
		fmt.Sprintf("%*s", sharesColumnWidth, sale.Shares),
		fmt.Sprintf("%*s", priceColumnWidth, portfolio.FormatUSD(sale.Price)),
		fmt.Sprintf("%*s", valueColumnWidth, portfolio.FormatUSD(sale.Proceeds())),
		fmt.Sprintf("%*s", valueColumnWidth, gain),
	}
}

// formatGain renders a gain in dollars with its sign in front, a plus as well as a minus, so that
// it reads as a change rather than as an amount.
func formatGain(gain decimal.Decimal) string {
	if gain.IsNegative() {
		return portfolio.FormatUSD(gain)
	}

	return "+" + portfolio.FormatUSD(gain)
}
