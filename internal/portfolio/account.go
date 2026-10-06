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

	// TaxableGain is what a sale of the shares that are held would be taxed on at these quotes: the
	// gains of the lots that are worth more than they cost, added up. A lot that lost takes nothing
	// off it, since every lot is taxed on its own, and neither does a lot without a cost, whose gain
	// is not known.
	TaxableGain decimal.Decimal

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

	var current, potential, taxable decimal.Decimal
	for _, lot := range lots {
		current = current.Add(value(lot.Symbol, lot.Remaining()))

		quote, priced := quotes[lot.Symbol]
		if gain, ok := lot.Gain(quote.Price); priced && ok && gain.IsPositive() {
			taxable = taxable.Add(gain)
		}
	}

	for _, grant := range grants {
		for _, vest := range grant.Vests {
			if !vest.Released {
				potential = potential.Add(value(grant.Symbol, vest.Shares))
			}
		}
	}

	return Account{
		Current:     current.Round(2),
		Potential:   potential.Round(2),
		TaxableGain: taxable.Round(2),
		Unpriced:    slices.Sorted(maps.Keys(unpriced)),
	}
}

// Total is the value of the whole account, held and still to come.
func (a Account) Total() decimal.Decimal {
	return a.Current.Add(a.Potential)
}

// AfterTax returns the account with its potential value after tax at rate, in percent: what the
// vests will bring once tax is withheld from them, rather than what the bank states for them. The
// current value stays the same, since the shares that are held were taxed as income when they
// vested, and what a sale of them is taxed at is up to AfterGainsTax. The tax
// is taken off the potential value as a whole, which is the same as off each vest, since every vest
// is taxed at the one rate, but for the rounding of a cent.
func (a Account) AfterTax(rate decimal.Decimal) Account {
	a.Potential = AfterTax(a.Potential, rate)
	return a
}

// AfterGainsTax returns the account with its current value after the tax on the taxable gain at
// rate, in percent: what the shares that are held would bring if they were all sold at these quotes,
// rather than what the bank states for them. The potential value stays the same, since a vest is
// taxed as income and has gained nothing yet. The tax is taken off the taxable gain as a whole,
// as a tax office would, which is the same as the tax on each lot added up but for the rounding of
// a cent: the Holdings view rounds the tax of every lot, and this rounds once.
func (a Account) AfterGainsTax(rate decimal.Decimal) Account {
	a.Current = a.Current.Sub(GainsTax(a.TaxableGain, rate))
	return a
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
