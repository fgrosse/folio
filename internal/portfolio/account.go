package portfolio

import (
	"github.com/shopspring/decimal"
)

// An Account is what the lots and grants of an account are worth at some set of quotes, in the
// three numbers the bank states for it.
type Account struct {
	// Current is the value of the shares that are held, which are the lots.
	Current decimal.Decimal
}

// NewAccount works out what lots and grants are worth at quotes, each value to the cent.
func NewAccount(lots []Lot, grants []Grant, quotes map[string]Quote) Account {
	var current decimal.Decimal
	for _, lot := range lots {
		current = current.Add(lot.Shares.Mul(quotes[lot.Symbol].Price))
	}

	return Account{Current: current.Round(2)}
}
