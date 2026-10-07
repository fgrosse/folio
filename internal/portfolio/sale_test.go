package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSale_ProceedsAndGain covers the two amounts a sale comes to: what it brought in, and how much
// of that is gain over what the shares cost. A sale from a lot whose cost is not known has proceeds
// but no gain to state.
func TestSale_ProceedsAndGain(t *testing.T) {
	sale := Sale{Shares: shares("50"), Price: shares("410.2"), Cost: shares("162.5")}

	assert.Equal(t, "20510", sale.Proceeds().String())

	gain, known := sale.Gain()
	assert.True(t, known)
	assert.Equal(t, "12385", gain.String())

	loss := Sale{Shares: shares("230"), Price: shares("131.1961"), Cost: shares("162.5")}
	gain, known = loss.Gain()
	assert.True(t, known)
	assert.Equal(t, "-7199.897", gain.String())

	_, known = Sale{Shares: shares("50"), Price: shares("410.2")}.Gain()
	assert.False(t, known, "a sale without a cost has no gain to state")
}

// TestNewRealized covers what all sales come to together: the money that was realized, and the gain
// in it, each to the cent. The gain leaves out the sales whose cost is not known, and the count of
// those says that it is not the whole gain.
func TestNewRealized(t *testing.T) {
	sales := []Sale{
		{Shares: shares("50"), Price: shares("410.2"), Cost: shares("162.5")},
		{Shares: shares("10.5"), Price: shares("400.333"), Cost: shares("231.48")},
		{Shares: shares("3"), Price: shares("390")},
	}

	realized := NewRealized(sales)

	// 20510 + 4203.4965 + 1170, and 12385 + 1772.9565
	assert.Equal(t, "25883.5", realized.Proceeds.String())
	assert.Equal(t, "14157.96", realized.Gain.String())
	assert.Equal(t, 1, realized.Uncosted)

	assert.Equal(t, "0", NewRealized(nil).Proceeds.String())
}

// TestSale_Tax covers what a sale costs in tax: the gains tax on its own gain, at the rate of the
// account. A sale at a loss is not taxed, and a sale of a lot without a cost has no gain to tax, so
// its tax is not known.
func TestSale_Tax(t *testing.T) {
	rate := shares("26.4")

	// A gain of 50 * 247.7 = 12385.
	tax, known := Sale{Shares: shares("50"), Price: shares("410.2"), Cost: shares("162.5")}.Tax(rate)
	assert.True(t, known)
	assert.Equal(t, "3269.64", tax.String())

	tax, known = Sale{Shares: shares("10"), Price: shares("150"), Cost: shares("162.5")}.Tax(rate)
	assert.True(t, known, "a loss is known not to be taxed")
	assert.Equal(t, "0", tax.String())

	_, known = Sale{Shares: shares("3"), Price: shares("390")}.Tax(rate)
	assert.False(t, known)
}
