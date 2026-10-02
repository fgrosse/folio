package tui

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// grantRow renders a grant as a row of the Grants table: how many of its shares are still to come,
// how many it had in all, and what the ones to come are worth at quote, which is the zero Quote if
// there is none of the grant's stock. The numbers are right-aligned and padded out like those of the
// other tables.
func grantRow(grant portfolio.Grant, quote portfolio.Quote) table.Row {
	var unreleased, granted decimal.Decimal
	for _, vest := range grant.Vests {
		granted = granted.Add(vest.Shares)
		if !vest.Released {
			unreleased = unreleased.Add(vest.Shares)
		}
	}

	value := noValue
	if quote.Symbol != "" {
		value = portfolio.FormatUSD(unreleased.Mul(quote.Price))
	}

	return table.Row{
		grant.Name,
		grant.Symbol,
		fmt.Sprintf("%*s", sharesColumnWidth, unreleased),
		fmt.Sprintf("%*s", sharesColumnWidth, granted),
		fmt.Sprintf("%*s", valueColumnWidth, value),
	}
}
