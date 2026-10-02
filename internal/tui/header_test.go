package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestAccountHeader covers the two lines above the table of every view, which are where the account
// values live: the total on the first line and the two values it is made of on the second, at the
// right end where the table's values are. What the view itself has to say goes on the left, with a
// status line under it.
func TestAccountHeader(t *testing.T) {
	account := portfolio.Account{Current: dec("3368.13"), Potential: dec("7925")}

	header := accountHeader("8.5 PANW", "PANW $396.25", account, 58, DefaultStyle())

	expected := "" +
		"  8.5 PANW                                 Total: $11,293.13\n" +
		"  PANW $396.25       Current $3,368.13 · Potential $7,925.00"
	assert.Equal(t, expected, ansi.Strip(header))

	// Both lines end in the column the table's last cell does: the indent and the width given.
	for line := range strings.SplitSeq(expected, "\n") {
		assert.Equal(t, 60, lipgloss.Width(line))
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
