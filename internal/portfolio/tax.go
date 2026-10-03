package portfolio

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// hundred is all of something, in percent.
var hundred = decimal.NewFromInt(100)

// ParseTaxRate parses the rate that a vest is taxed at, written as a percentage such as "44.3%" or
// "44.3", and returns it in percent. A vest is taxed as income, at a rate that depends on the rest
// of the year's income and the country, so folio does not work it out but takes one rate for every
// vest, which is the user's estimate of the rate at the top of their income.
func ParseTaxRate(spec string) (decimal.Decimal, error) {
	number := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(spec), "%"))

	rate, err := decimal.NewFromString(number)
	switch {
	case err != nil:
		return decimal.Zero, fmt.Errorf("%q is not a tax rate such as 44.3%%", spec)
	case rate.IsNegative() || rate.GreaterThan(hundred):
		return decimal.Zero, fmt.Errorf("a tax rate is between 0%% and 100%%, not %s%%", rate)
	}

	return rate, nil
}

// AfterTax returns what is left of value once tax at rate, in percent, is taken off, to the cent.
// It is what a vest is worth to whoever receives it, which is less than the bank states for it.
func AfterTax(value, rate decimal.Decimal) decimal.Decimal {
	return value.Mul(hundred.Sub(rate)).Div(hundred).Round(2)
}
