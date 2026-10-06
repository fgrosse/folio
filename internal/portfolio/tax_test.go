package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTaxRate covers how the rate that vests are taxed at is typed: as a percentage, with or
// without its sign, and from none up to all of a vest.
func TestParseTaxRate(t *testing.T) {
	cases := map[string]struct {
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

	for name, tt := range cases {
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

// TestParseSwitch covers how a setting that is on or off is typed: as true or false, however it is
// capitalized, and nothing else.
func TestParseSwitch(t *testing.T) {
	cases := map[string]struct {
		value    string
		expected bool
		error    string
	}{
		"true":       {value: "true", expected: true},
		"false":      {value: "false", expected: false},
		"capitals":   {value: " True ", expected: true},
		"nothing":    {value: "", error: `"" is neither true nor false`},
		"other word": {value: "net", error: `"net" is neither true nor false`},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			on, err := ParseSwitch(c.value)
			if c.error != "" {
				assert.EqualError(t, err, c.error)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, c.expected, on)
		})
	}
}

// TestGainsTax covers the tax that is due on the gain of a sale, at a rate in percent and to the
// cent. A loss is not taxed, and neither is a sale at what the shares cost.
func TestGainsTax(t *testing.T) {
	assert.Equal(t, "25.55", GainsTax(shares("96.78"), shares("26.4")).String())
	assert.Equal(t, "0", GainsTax(shares("96.78"), shares("0")).String())
	assert.Equal(t, "0", GainsTax(shares("-169.83"), shares("26.4")).String())
	assert.Equal(t, "0", GainsTax(shares("0"), shares("26.4")).String())
}
