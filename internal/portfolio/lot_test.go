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

// TestNewLot_Errors covers the specs NewLot refuses, each with an error that says what to type
// instead: it is shown under the field of the dialog the spec was typed into.
func TestNewLot_Errors(t *testing.T) {
	tests := map[string]struct {
		spec  string
		error string
	}{
		"empty": {
			spec:  "",
			error: `a lot is written as "<shares> <symbol> [YYYY-MM-DD]"`,
		},
		"only shares": {
			spec:  "12",
			error: `a lot is written as "<shares> <symbol> [YYYY-MM-DD]"`,
		},
		"too many fields": {
			spec:  "12 PANW 2026-03-15 vested",
			error: `a lot is written as "<shares> <symbol> [YYYY-MM-DD]"`,
		},
		"shares that are not a number": {
			spec:  "twelve PANW",
			error: `"twelve" is not a number of shares`,
		},
		"a day that is not YYYY-MM-DD": {
			spec:  "12 PANW 15.03.2026",
			error: `"15.03.2026" is not a day written as YYYY-MM-DD`,
		},
		"no shares": {
			spec:  "0 PANW",
			error: "lot of PANW must have more than 0 shares",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewLot(tt.spec)
			assert.EqualError(t, err, tt.error)
		})
	}
}
