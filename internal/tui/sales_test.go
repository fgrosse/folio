package tui

import (
	"testing"

	"charm.land/bubbles/v2/table"
	"github.com/stretchr/testify/assert"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestSaleRow covers how one sale reads as a row of the Sales table: its day, the stock and the
// grant its lot was from, and the numbers right-aligned - how many shares, what one sold for, what
// that brought in, and how much of it is gain over what the shares cost. A gain says which way it
// goes with a sign, and a sale of a lot without a cost has none to state.
func TestSaleRow(t *testing.T) {
	tests := map[string]struct {
		sale     portfolio.Sale
		expected table.Row
	}{
		"sold at a gain": {
			sale: portfolio.Sale{
				Date: day("2026-09-15"), Symbol: "PANW", Grant: "Payout",
				Shares: dec("50"), Price: dec("410.2"), Cost: dec("162.5"),
			},
			expected: table.Row{"2026-09-15", "PANW", "Payout", "        50", "   $410.20", "    $20,510.00", "   +$12,385.00"},
		},
		"sold at a loss": {
			sale: portfolio.Sale{
				Date: day("2025-08-01"), Symbol: "PANW", Grant: "Payout",
				Shares: dec("230"), Price: dec("131.1961"), Cost: dec("162.5"),
			},
			expected: table.Row{"2025-08-01", "PANW", "Payout", "       230", "   $131.20", "    $30,175.10", "    -$7,199.90"},
		},
		"a lot without a cost, entered by hand": {
			sale: portfolio.Sale{
				Date: day("2026-09-20"), Symbol: "AAPL",
				Shares: dec("2.5"), Price: dec("330"),
			},
			expected: table.Row{"2026-09-20", "AAPL", "", "       2.5", "   $330.00", "       $825.00", "             -"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, saleRow(tt.sale))
		})
	}
}
