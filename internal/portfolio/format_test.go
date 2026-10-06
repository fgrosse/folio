package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFormatUSD covers how an amount of money reads everywhere folio shows one: in dollars and
// always to the cent, with the thousands set apart, since an account value is a number of five or
// six digits that is otherwise hard to take in at a glance.
func TestFormatUSD(t *testing.T) {
	cases := map[string]struct {
		amount   string
		expected string
	}{
		"nothing":                    {amount: "0", expected: "$0.00"},
		"less than a dollar":         {amount: "0.5", expected: "$0.50"},
		"hundreds":                   {amount: "792.5", expected: "$792.50"},
		"thousands":                  {amount: "4359.09", expected: "$4,359.09"},
		"a round thousand":           {amount: "1000", expected: "$1,000.00"},
		"millions":                   {amount: "1234567.8", expected: "$1,234,567.80"},
		"rounded to the cent":        {amount: "12.345", expected: "$12.35"},
		"rounded up to a new digit":  {amount: "999.999", expected: "$1,000.00"},
		"an amount that is negative": {amount: "-1234.5", expected: "-$1,234.50"},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatUSD(shares(tt.amount)))
		})
	}
}
