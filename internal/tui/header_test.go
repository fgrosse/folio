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

	values := shownValues{account: account, current: portfolio.Gross, potential: portfolio.Gross}

	header := accountHeader("8.5 PANW", "PANW $396.25", values, 74, DefaultStyle())

	expected := "" +
		"  8.5 PANW                                                 Total: $11,293.13\n" +
		"  PANW $396.25       Current (gross) $3,368.13 · Potential (gross) $7,925.00"
	assert.Equal(t, expected, ansi.Strip(header))

	// Both lines end in the column the table's last cell does: the indent and the width given.
	for line := range strings.SplitSeq(expected, "\n") {
		assert.Equal(t, 76, lipgloss.Width(line))
	}
}

// TestPortfolioHeader_PotentialBasis covers the potential value the header shows, which is the
// bank's unless the account asks for it after tax, and the word next to it that says which one it
// is. After tax, the total is the current value and the potential one after tax, so that the
// values on screen add up. Without a tax rate there is nothing to take off, and the header says it
// shows the bank's number.
func TestPortfolioHeader_PotentialBasis(t *testing.T) {
	tests := map[string]struct {
		basis     portfolio.Basis
		noTaxRate bool
		total     string
		parts     string
	}{
		"gross": {
			basis: portfolio.Gross,
			total: "Total: $11,293.13",
			parts: "Current (gross) $3,368.13 · Potential (gross) $7,925.00",
		},
		"net": {
			basis: portfolio.Net,
			total: "Total: $7,782.36",
			parts: "Current (gross) $3,368.13 · Potential (net) $4,414.23",
		},
		"net without a tax rate": {
			basis:     portfolio.Net,
			noTaxRate: true,
			total:     "Total: $11,293.13",
			parts:     "Current (gross) $3,368.13 · Potential (gross) $7,925.00",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := testPortfolio()
			p.PotentialBasis = tt.basis
			if tt.noTaxRate {
				p.TaxRate = decimal.NullDecimal{}
			}

			lines := strings.Split(ansi.Strip(portfolioHeader("8.5 PANW", p, nil, 80, DefaultStyle())), "\n")

			require.Len(t, lines, 2)
			assert.True(t, strings.HasSuffix(lines[0], tt.total), "%q should end in %q", lines[0], tt.total)
			assert.True(t, strings.HasSuffix(lines[1], tt.parts), "%q should end in %q", lines[1], tt.parts)
		})
	}
}

// TestPortfolioHeader_CurrentBasis covers the current value the header shows, which is the bank's
// unless the account asks for it after tax: then it is what the shares that are held would bring
// if they were sold today, less the tax on the gain of each lot, and says so. The total counts the
// value that is shown, next to a potential value that has a basis of its own. Without a gains tax
// rate there is nothing to take off, and the header says it shows the bank's number.
func TestPortfolioHeader_CurrentBasis(t *testing.T) {
	tests := map[string]struct {
		current   portfolio.Basis
		potential portfolio.Basis
		noRate    bool
		total     string
		parts     string
	}{
		"net": {
			current:   portfolio.Net,
			potential: portfolio.Gross,
			total:     "Total: $11,267.58",
			parts:     "Current (net) $3,342.58 · Potential (gross) $7,925.00",
		},
		"both net": {
			current:   portfolio.Net,
			potential: portfolio.Net,
			total:     "Total: $7,756.81",
			parts:     "Current (net) $3,342.58 · Potential (net) $4,414.23",
		},
		"net without a gains tax rate": {
			current:   portfolio.Net,
			potential: portfolio.Gross,
			noRate:    true,
			total:     "Total: $11,293.13",
			parts:     "Current (gross) $3,368.13 · Potential (gross) $7,925.00",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := testPortfolio()
			p.CurrentBasis = tt.current
			p.PotentialBasis = tt.potential
			if tt.noRate {
				p.GainsTaxRate = decimal.NullDecimal{}
			}

			lines := strings.Split(ansi.Strip(portfolioHeader("8.5 PANW", p, nil, 80, DefaultStyle())), "\n")

			require.Len(t, lines, 2)
			assert.True(t, strings.HasSuffix(lines[0], tt.total), "%q should end in %q", lines[0], tt.total)
			assert.True(t, strings.HasSuffix(lines[1], tt.parts), "%q should end in %q", lines[1], tt.parts)
		})
	}
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
