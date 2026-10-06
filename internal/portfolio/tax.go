package portfolio

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// TaxRateKey is the key of the configuration that the rate vests are taxed at is set under, in
// percent and as ParseTaxRate reads it.
const TaxRateKey = "tax-rate"

// PotentialBasisKey is the key of the configuration that says which potential value the TUI shows
// in its header, as ParseBasis reads it. It is gross when it is not set.
const PotentialBasisKey = "potential"

const (
	// Gross is a value before tax, as the bank states it.
	Gross Basis = "gross"

	// Net is a value after tax, at the rate the account has for it.
	Net Basis = "net"
)

// hundred is all of something, in percent.
var hundred = decimal.NewFromInt(100)

// A Basis is whether a value of an account is shown as the bank states it, before tax, or as what
// is left of it once the tax on it is taken off. The bank's number is the one to compare with its
// web site, and the one after tax is closer to what the shares will bring.
type Basis string

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

// ParseBasis parses which of a value to show, written as "gross" or "net".
func ParseBasis(value string) (Basis, error) {
	switch basis := Basis(strings.ToLower(strings.TrimSpace(value))); basis {
	case Gross, Net:
		return basis, nil
	default:
		return "", fmt.Errorf("%q is neither gross nor net", value)
	}
}
