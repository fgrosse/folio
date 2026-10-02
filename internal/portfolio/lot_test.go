package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLot covers the syntax a lot is typed in, on the command line and in the TUI's dialog:
// "<shares> <symbol> [YYYY-MM-DD]". The symbol is the stock exchange's, which is written in capitals
// however it was typed. The day is optional, and a lot without one leaves it to the caller, who
// knows what today is.
func TestNewLot(t *testing.T) {
	tests := map[string]struct {
		spec     string
		expected Lot
	}{
		"shares, symbol and day": {
			spec:     "12.5 PANW 2026-03-15",
			expected: Lot{Symbol: "PANW", Shares: shares("12.5"), Acquired: day("2026-03-15")},
		},
		"without a day": {
			spec:     "40 PANW",
			expected: Lot{Symbol: "PANW", Shares: shares("40")},
		},
		"a symbol in lower case": {
			spec:     "3 sap.de 2026-01-02",
			expected: Lot{Symbol: "SAP.DE", Shares: shares("3"), Acquired: day("2026-01-02")},
		},
		"space around and between": {
			spec:     "  7   PANW   2026-03-15 ",
			expected: Lot{Symbol: "PANW", Shares: shares("7"), Acquired: day("2026-03-15")},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			lot, err := NewLot(tt.spec)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, lot)
		})
	}
}
