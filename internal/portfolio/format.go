package portfolio

import (
	"strings"

	"github.com/shopspring/decimal"
)

// FormatUSD renders an amount of money in dollars, to the cent and with the thousands set apart by
// commas, such as "$12,345.60".
func FormatUSD(amount decimal.Decimal) string {
	sign := ""
	if amount.IsNegative() {
		sign = "-"
	}

	dollars, cents, _ := strings.Cut(amount.Abs().StringFixed(2), ".")

	// The commas go in from the right, before every group of three digits but the first.
	var grouped strings.Builder
	for i, digit := range dollars {
		if i > 0 && (len(dollars)-i)%3 == 0 {
			grouped.WriteRune(',')
		}
		grouped.WriteRune(digit)
	}

	return sign + "$" + grouped.String() + "." + cents
}
