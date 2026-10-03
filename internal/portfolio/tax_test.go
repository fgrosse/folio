package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTaxRate covers how the rate that vests are taxed at is typed: as a percentage, with or
// without its sign, and from none up to all of a vest.
func TestParseTaxRate(t *testing.T) {
	tests := map[string]struct {
		spec  string
		rate  string
		error string
	}{
		"with the percent sign":    {spec: "44.3%", rate: "44.3"},
		"without the percent sign": {spec: "44.3", rate: "44.3"},
		"with space":               {spec: " 44.3 % ", rate: "44.3"},
		"none":                     {spec: "0%", rate: "0"},
		"all":                      {spec: "100%", rate: "100"},
		"nothing":                  {spec: "", error: `"" is not a tax rate such as 44.3%`},
		"not a number":             {spec: "high", error: `"high" is not a tax rate such as 44.3%`},
		"less than none":           {spec: "-5%", error: "a tax rate is between 0% and 100%, not -5%"},
		"more than all":            {spec: "120%", error: "a tax rate is between 0% and 100%, not 120%"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rate, err := ParseTaxRate(tt.spec)
			if tt.error != "" {
				assert.EqualError(t, err, tt.error)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.rate, rate.String())
		})
	}
}

// TestAfterTax covers what is left of a value once a rate in percent is taken off, to the cent.
func TestAfterTax(t *testing.T) {
	assert.Equal(t, "2207.11", AfterTax(shares("3962.50"), shares("44.3")).String())
	assert.Equal(t, "3962.5", AfterTax(shares("3962.50"), shares("0")).String())
	assert.Equal(t, "0", AfterTax(shares("3962.50"), shares("100")).String())
}
