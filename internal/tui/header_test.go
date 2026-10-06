package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestAccountHeader covers the two lines above the table of every view, which are where the account
// values live: the total on the first line and the two values it is made of on the second, at the
// right end where the table's values are. What the view itself has to say goes on the left, with a
// status line under it.
func TestAccountHeader(t *testing.T) {
	account := portfolio.Account{Current: dec("3368.13"), Potential: dec("7925")}

	values := shownValues{account: account}

	header := accountHeader("8.5 PANW", "PANW $396.25", values, 66, DefaultStyle())

	expected := "" +
		"  8.5 PANW                                         Total: $11,293.13\n" +
		"  PANW $396.25       Current $3,368.13 · Potential (gross) $7,925.00"
	assert.Equal(t, expected, ansi.Strip(header))

	// Both lines end in the column the table's last cell does: the indent and the width given.
	for line := range strings.SplitSeq(expected, "\n") {
		assert.Equal(t, 68, lipgloss.Width(line))
	}
}

// TestPortfolioHeader_ShowNet covers the values the header shows, which are the bank's unless the
// account asks for them after tax. Then the potential value is what is left of the vests after tax
// at the tax rate, and the current value what the shares that are held would bring if they were
// sold today, less the tax on the gain of each lot. Each says that it is net, and the total is the
// two added up, so that the values on screen add up. A value without a rate to take off stays the
// bank's and does not say net. The current value says nothing while it is the bank's: the line has
// the prices to fit in as well.
func TestPortfolioHeader_ShowNet(t *testing.T) {
	cases := map[string]struct {
		showNet        bool
		noTaxRate      bool
		noGainsTaxRate bool
		total          string
		parts          string
	}{
		"gross": {
			total: "Total: $11,293.13",
			parts: "Current $3,368.13 · Potential (gross) $7,925.00",
		},
		"net": {
			showNet: true,
			total:   "Total: $7,756.81",
			parts:   "Current (net) $3,342.58 · Potential (net) $4,414.23",
		},
		"net without a tax rate": {
			showNet:   true,
			noTaxRate: true,
			total:     "Total: $11,267.58",
			parts:     "Current (net) $3,342.58 · Potential (gross) $7,925.00",
		},
		"net without a gains tax rate": {
			showNet:        true,
			noGainsTaxRate: true,
			total:          "Total: $7,782.36",
			parts:          "Current $3,368.13 · Potential (net) $4,414.23",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := testPortfolio()
			p.ShowNet = c.showNet
			if c.noTaxRate {
				p.TaxRate = decimal.NullDecimal{}
			}
			if c.noGainsTaxRate {
				p.GainsTaxRate = decimal.NullDecimal{}
			}

			lines := strings.Split(ansi.Strip(portfolioHeader("8.5 PANW", p, nil, 80, DefaultStyle())), "\n")

			require.Len(t, lines, 2)
			assert.True(t, strings.HasSuffix(lines[0], c.total), "%q should end in %q", lines[0], c.total)
			assert.True(t, strings.HasSuffix(lines[1], c.parts), "%q should end in %q", lines[1], c.parts)
		})
	}
}

// TestPortfolioHeader_CutsThePricesShort covers a header with more prices than its line has room
// for next to the account values, which a second stock or a value marked net is enough for in a
// narrow window. The prices are cut short rather than pushing the values past the end of the table,
// where the terminal would wrap them into the frame.
func TestPortfolioHeader_CutsThePricesShort(t *testing.T) {
	p := testPortfolio()
	p.ShowNet = true
	p.TaxRate = decimal.NullDecimal{}
	p.Lots = append(p.Lots, portfolio.Lot{ID: 3, Symbol: "AAPL", Shares: dec("3"), Acquired: day("2025-11-02")})
	p.Quotes["AAPL"] = portfolio.Quote{Symbol: "AAPL", Price: dec("330.32"), PreviousClose: dec("325")}

	lines := strings.Split(ansi.Strip(portfolioHeader("3 AAPL · 8.5 PANW", p, nil, 78, DefaultStyle())), "\n")

	require.Len(t, lines, 2)
	assert.Equal(t, "  AAPL $330.32 ▲ 1.6% · …  Current (net) $4,333.54 · Potential (gross) $7,925.00", lines[1])
	assert.Equal(t, 80, lipgloss.Width(lines[1]))
}

// TestQuoteStatus covers the status line under a view's own: the price each stock of the account is
// valued at and how it moved since the previous close, the way the ticker in the status bar reads.
// A stock without a quote says so, since its shares are missing from the values.
func TestQuoteStatus(t *testing.T) {
	quotes := map[string]portfolio.Quote{
		"PANW": {Symbol: "PANW", Price: dec("396.25"), PreviousClose: dec("397.31")},
		"AAPL": {Symbol: "AAPL", Price: dec("330.32"), PreviousClose: dec("325")},
		"NEW":  {Symbol: "NEW", Price: dec("12")}, // has no previous close to have moved from
	}

	tests := map[string]struct {
		symbols  []string
		expected string
	}{
		"down":                     {symbols: []string{"PANW"}, expected: "PANW $396.25 ▼ 0.3%"},
		"up":                       {symbols: []string{"AAPL"}, expected: "AAPL $330.32 ▲ 1.6%"},
		"without a previous close": {symbols: []string{"NEW"}, expected: "NEW $12.00"},
		"without a quote":          {symbols: []string{"SAP.DE"}, expected: "SAP.DE has no price"},
		"several":                  {symbols: []string{"AAPL", "PANW"}, expected: "AAPL $330.32 ▲ 1.6% · PANW $396.25 ▼ 0.3%"},
		"no stock at all":          {symbols: nil, expected: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, quoteStatus(tt.symbols, quotes))
		})
	}
}
