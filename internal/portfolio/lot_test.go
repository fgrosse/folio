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
		// The "@" sets the cost apart by itself, so the space before it is optional, and one after
		// it does no harm either.
		"no space before the cost": {
			spec:     "12.5 PANW 2026-03-15@380.12",
			expected: Lot{Symbol: "PANW", Shares: shares("12.5"), Acquired: day("2026-03-15"), Cost: shares("380.12")},
		},
		"no space before the cost, without a day": {
			spec:     "40 PANW@396",
			expected: Lot{Symbol: "PANW", Shares: shares("40"), Cost: shares("396")},
		},
		"space after the @": {
			spec:     "40 PANW @ 396",
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

// TestParseRelease covers what is typed when a vest is released, "<shares> [@<cost>]": the shares
// that arrived and what one was worth that day, with or without space around the "@". A release
// without a cost has none.
func TestParseRelease(t *testing.T) {
	tests := map[string]struct {
		spec   string
		shares string
		cost   string
		error  string
	}{
		"shares and cost":        {spec: "250 @162.50", shares: "250", cost: "162.5"},
		"no space before the @":  {spec: "250@162.50", shares: "250", cost: "162.5"},
		"space around the @":     {spec: " 250 @ 162.50 ", shares: "250", cost: "162.5"},
		"only shares":            {spec: "42", shares: "42", cost: "0"},
		"nothing":                {spec: "", error: `a release is written as "<shares> [@<cost>]"`},
		"too much":               {spec: "250 @162.50 net", error: `a release is written as "<shares> [@<cost>]"`},
		"only a cost":            {spec: "@162.50", error: `"@162.50" is not a number of shares`},
		"shares not a number":    {spec: "many @162.50", error: `"many" is not a number of shares`},
		"no shares":              {spec: "0 @162.50", error: "a vest must release more than 0 shares"},
		"a cost without its @":   {spec: "250 162.50", error: `"162.50" is not a cost such as @380.12`},
		"a cost that is nothing": {spec: "250 @0", error: `"@0" is not a cost such as @380.12`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			released, cost, err := ParseRelease(tt.spec)
			if tt.error != "" {
				assert.EqualError(t, err, tt.error)
				return
			}

			require.NoError(t, err)
			assert.True(t, shares(tt.shares).Equal(released), "shares: %s", released)
			assert.True(t, shares(tt.cost).Equal(cost), "cost: %s", cost)
		})
	}
}

// TestLot_GrowthPercent covers how far the price of a share has moved from what the lot cost, in percent
// of that cost: up, down, or not at all. A lot without a cost has nothing to measure from, and says
// so rather than passing for a lot that has not moved.
func TestLot_GrowthPercent(t *testing.T) {
	cases := map[string]struct {
		cost     string
		price    string
		expected string
		unknown  bool
	}{
		"up":             {cost: "200", price: "446", expected: "123"},
		"down":           {cost: "400", price: "350", expected: "-12.5"},
		"where it was":   {cost: "380.12", price: "380.12", expected: "0"},
		"without a cost": {cost: "0", price: "396.25", unknown: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			lot := Lot{Symbol: "PANW", Shares: shares("6"), Cost: shares(c.cost)}

			growth, ok := lot.GrowthPercent(shares(c.price))

			assert.Equal(t, !c.unknown, ok)
			if ok {
				assert.Equal(t, c.expected, growth.String())
			}
		})
	}
}

// TestLot_Gain covers what the shares left of a lot are worth over what they cost, which is what a
// sale of them at that price would be taxed on: the difference a share made, for every share that
// is still held. It is less than zero for a lot that lost. A lot without a cost has no gain to
// state, and says so.
func TestLot_Gain(t *testing.T) {
	cases := map[string]struct {
		lot      Lot
		expected string
		unknown  bool
	}{
		"up":             {lot: Lot{Shares: shares("6"), Cost: shares("380.12")}, expected: "96.78"},
		"down":           {lot: Lot{Shares: shares("3"), Cost: shares("452.86")}, expected: "-169.83"},
		"partly sold":    {lot: Lot{Shares: shares("6"), Sold: shares("2"), Cost: shares("380.12")}, expected: "64.52"},
		"without a cost": {lot: Lot{Shares: shares("2.5")}, unknown: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			gain, ok := c.lot.Gain(shares("396.25"))

			assert.Equal(t, !c.unknown, ok)
			if ok {
				assert.Equal(t, c.expected, gain.String())
			}
		})
	}
}
