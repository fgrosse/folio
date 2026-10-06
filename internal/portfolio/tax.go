package portfolio

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// TaxRateKey is the key of the configuration that the rate vests are taxed at is set under, in
// percent and as ParseTaxRate reads it.
const TaxRateKey = "tax-rate"

// GainsTaxRateKey is the key of the configuration that the rate the gain of a sale is taxed at is
// set under, in percent and as ParseTaxRate reads it.
const GainsTaxRateKey = "gains-tax-rate"

// ShowNetSummaryKey is the key of the configuration that says whether the TUI shows the values in
// its header after tax, as ParseSwitch reads it. They are before tax when it is not set.
const ShowNetSummaryKey = "show-net-summary"

// hundred is all of something, in percent.
var hundred = decimal.NewFromInt(100)

// ParseTaxRate parses a rate that something is taxed at, written as a percentage such as "44.3%" or
// "44.3", and returns it in percent. A vest is taxed as income, at a rate that depends on the rest
// of the year's income and the country, so folio does not work it out but takes one rate for every
// vest, which is the user's estimate of the rate at the top of their income. The gain of a sale is
// taxed at a rate of its own, which in many countries is the same whatever the income.
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

// GainsTax returns the tax that is due on gain, what a sale brought over what the shares cost, at
// rate in percent and to the cent. A loss is not taxed. Neither does it earn anything back here:
// what a loss is good for depends on the other sales of the year, which this does not know.
func GainsTax(gain, rate decimal.Decimal) decimal.Decimal {
	if !gain.IsPositive() {
		return decimal.Zero
	}

	return gain.Mul(rate).Div(hundred).Round(2)
}

// ParseSwitch parses a setting that is on or off, written as "true" or "false".
func ParseSwitch(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%q is neither true nor false", value)
	}
}
