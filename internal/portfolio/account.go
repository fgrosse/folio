package portfolio

import (
	"maps"
	"slices"

	"github.com/shopspring/decimal"
)

// An Account is what the lots and grants of an account are worth at some set of quotes, in the
// numbers the bank states for it.
type Account struct {
	// Current is the value of the shares that are held: what remains of the lots after their sales.
	Current decimal.Decimal

	// Potential is the value of the shares still to come, which are the vests that have not been
	// released: the ones that have yet to vest, and the ones that have and are pending release.
	Potential decimal.Decimal

	// Unpriced are the symbols of stock in the account that there was no quote of, in alphabetical
	// order. Their shares are missing from the values.
	Unpriced []string
}

// NewAccount works out what lots and grants are worth at quotes, each value to the cent. Stock
// without a quote adds nothing to the values and is listed as unpriced instead.
func NewAccount(lots []Lot, grants []Grant, quotes map[string]Quote) Account {
	unpriced := make(map[string]bool)
	value := func(symbol string, shares decimal.Decimal) decimal.Decimal {
		quote, ok := quotes[symbol]
		if !ok {
			unpriced[symbol] = true
		}

		return shares.Mul(quote.Price)
	}

	var current, potential decimal.Decimal
	for _, lot := range lots {
		current = current.Add(value(lot.Symbol, lot.Remaining()))
	}

	for _, grant := range grants {
		for _, vest := range grant.Vests {
			if !vest.Released {
				potential = potential.Add(value(grant.Symbol, vest.Shares))
			}
		}
	}

	return Account{
		Current:   current.Round(2),
		Potential: potential.Round(2),
		Unpriced:  slices.Sorted(maps.Keys(unpriced)),
	}
}

// Total is the value of the whole account, held and still to come.
func (a Account) Total() decimal.Decimal {
	return a.Current.Add(a.Potential)
}

// Symbols returns the symbols of the stock in lots and grants, each once and in alphabetical order.
// They are what an account needs quotes of.
func Symbols(lots []Lot, grants []Grant) []string {
	symbols := make(map[string]bool)
	for _, lot := range lots {
		symbols[lot.Symbol] = true
	}
	for _, grant := range grants {
		symbols[grant.Symbol] = true
	}

	return slices.Sorted(maps.Keys(symbols))
}
