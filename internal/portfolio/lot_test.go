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
			error: `a lot is written as "<shares> <symbol> [YYYY-MM-DD] [@<cost>]"`,
		},
		"only shares": {
			spec:  "12",
			error: `a lot is written as "<shares> <symbol> [YYYY-MM-DD] [@<cost>]"`,
		},
		"too many fields": {
			spec:  "12 PANW 2026-03-15 @380.12 vested",
			error: `a lot is written as "<shares> <symbol> [YYYY-MM-DD] [@<cost>]"`,
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
		"two days": {
			spec:  "12 PANW 2026-03-15 2026-03-16",
			error: `a lot is written as "<shares> <symbol> [YYYY-MM-DD] [@<cost>]"`,
		},
		"two costs": {
			spec:  "12 PANW @380.12 @381",
			error: `a lot is written as "<shares> <symbol> [YYYY-MM-DD] [@<cost>]"`,
		},
		"a cost that is not a number": {
			spec:  "12 PANW @cheap",
			error: `"@cheap" is not a cost such as @380.12`,
		},
		"a cost of nothing": {
			spec:  "12 PANW @0",
			error: `"@0" is not a cost such as @380.12`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewLot(tt.spec)
			assert.EqualError(t, err, tt.error)
		})
	}
}

// TestNewLot_Cost covers the price the shares of a lot were acquired at, which a spec gives after an
// "@": what a share cost on the day it was bought, or what it was worth on the day it vested. It is
// what a gain is measured from, and with it what tax is due on. The cost is optional and may come
// before or after the day.
func TestNewLot_Cost(t *testing.T) {
	tests := map[string]struct {
		spec     string
		expected Lot
	}{
		"a cost after the day": {
			spec:     "12.5 PANW 2026-03-15 @380.12",
			expected: Lot{Symbol: "PANW", Shares: shares("12.5"), Acquired: day("2026-03-15"), Cost: shares("380.12")},
		},
		"a cost before the day": {
			spec:     "12.5 PANW @380.12 2026-03-15",
			expected: Lot{Symbol: "PANW", Shares: shares("12.5"), Acquired: day("2026-03-15"), Cost: shares("380.12")},
		},
		"a cost without a day": {
			spec:     "40 PANW @396",
			expected: Lot{Symbol: "PANW", Shares: shares("40"), Cost: shares("396")},
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

// TestLot_String covers a lot written back as its spec, which is what the dialog that edits a lot
// starts out with: exactly the syntax NewLot reads, so that a lot survives the round trip, and
// without a cost if the lot has none.
func TestLot_String(t *testing.T) {
	tests := map[string]struct {
		lot      Lot
		expected string
	}{
		"with a cost": {
			lot:      Lot{ID: 3, Symbol: "PANW", Shares: shares("12.5"), Acquired: day("2026-03-15"), Cost: shares("380.12"), Grant: "Payout"},
			expected: "12.5 PANW 2026-03-15 @380.12",
		},
		"without a cost": {
			lot:      Lot{Symbol: "PANW", Shares: shares("40"), Acquired: day("2026-03-15")},
			expected: "40 PANW 2026-03-15",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.lot.String())

			parsed, err := NewLot(tt.lot.String())
			require.NoError(t, err)
			assert.Equal(t, tt.lot.Symbol, parsed.Symbol)
			assert.Equal(t, tt.lot.Acquired, parsed.Acquired)
			assert.True(t, tt.lot.Shares.Equal(parsed.Shares))
			assert.True(t, tt.lot.Cost.Equal(parsed.Cost))
		})
	}
}
