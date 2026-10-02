package portfolio

import (
	"github.com/shopspring/decimal"
)

// An Account is what the lots and grants of an account are worth at some set of quotes, in the
// numbers the bank states for it.
type Account struct {
	// Current is the value of the shares that are held, which are the lots.
	Current decimal.Decimal

	// Potential is the value of the shares still to come, which are the vests that have not been
	// released: the ones that have yet to vest, and the ones that have and are pending release.
	Potential decimal.Decimal
}

// NewAccount works out what lots and grants are worth at quotes, each value to the cent.
func NewAccount(lots []Lot, grants []Grant, quotes map[string]Quote) Account {
	var current, potential decimal.Decimal
	for _, lot := range lots {
		current = current.Add(lot.Shares.Mul(quotes[lot.Symbol].Price))
	}

	for _, grant := range grants {
		for _, vest := range grant.Vests {
			if !vest.Released {
				potential = potential.Add(vest.Shares.Mul(quotes[grant.Symbol].Price))
			}
		}
	}

	return Account{
		Current:   current.Round(2),
		Potential: potential.Round(2),
	}
}
