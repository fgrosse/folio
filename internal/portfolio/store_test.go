package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewTestingStore returns a Store backed by a real SQLite database that only ever lives in
// memory, so tests exercise the same SQL the production code runs without touching disk or
// needing cleanup. Unlike NewStore, it also applies every migration before returning, since a
// fresh in-memory database only exists for the lifetime of this call and there is no reason a
// test would want to see it schema-less.
func NewTestingStore() *SQLiteStore {
	s, err := NewStore(":memory:")
	if err != nil {
		panic(err)
	}

	if err := s.Migrate(); err != nil {
		panic(err)
	}

	return s
}

// TestStore_SaveLot covers the first thing folio is for, recording shares that are held: a fresh
// database has none, and a lot that is saved comes back as it was, with an ID of its own.
func TestStore_SaveLot(t *testing.T) {
	s := NewTestingStore()

	lots, err := s.Lots()
	require.NoError(t, err)
	assert.Empty(t, lots)

	err = s.SaveLot(Lot{Symbol: "PANW", Shares: shares("12.5"), Acquired: day("2026-03-15")})
	require.NoError(t, err)

	lots, err = s.Lots()
	require.NoError(t, err)
	expected := []Lot{
		{ID: 1, Symbol: "PANW", Shares: shares("12.5"), Acquired: day("2026-03-15")},
	}
	assert.Equal(t, expected, lots)
}

// TestStore_LotsAreListedByDay covers the order lots come back in: by the day they were acquired,
// whatever order they were saved in, which is the order an account's history reads in. Lots of the
// same day stay in the order they were saved.
func TestStore_LotsAreListedByDay(t *testing.T) {
	s := NewTestingStore()
	require.NoError(t, s.SaveLot(Lot{Symbol: "PANW", Shares: shares("3"), Acquired: day("2026-03-15")}))
	require.NoError(t, s.SaveLot(Lot{Symbol: "AAPL", Shares: shares("1"), Acquired: day("2025-11-02")}))
	require.NoError(t, s.SaveLot(Lot{Symbol: "PANW", Shares: shares("2"), Acquired: day("2026-03-15")}))

	lots, err := s.Lots()
	require.NoError(t, err)
	expected := []Lot{
		{ID: 2, Symbol: "AAPL", Shares: shares("1"), Acquired: day("2025-11-02")},
		{ID: 1, Symbol: "PANW", Shares: shares("3"), Acquired: day("2026-03-15")},
		{ID: 3, Symbol: "PANW", Shares: shares("2"), Acquired: day("2026-03-15")},
	}
	assert.Equal(t, expected, lots)
}

// TestStore_SaveLotRefusesInvalidLots covers what a lot has to have to be worth anything: a symbol
// to look its price up by, shares to multiply it with, and the day they arrived.
func TestStore_SaveLotRefusesInvalidLots(t *testing.T) {
	tests := map[string]struct {
		lot   Lot
		error string
	}{
		"no symbol": {
			lot:   Lot{Shares: shares("1"), Acquired: day("2026-03-15")},
			error: "lot has no symbol",
		},
		"no shares": {
			lot:   Lot{Symbol: "PANW", Acquired: day("2026-03-15")},
			error: "lot of PANW must have more than 0 shares",
		},
		"negative shares": {
			lot:   Lot{Symbol: "PANW", Shares: shares("-2"), Acquired: day("2026-03-15")},
			error: "lot of PANW must have more than 0 shares",
		},
		"no day": {
			lot:   Lot{Symbol: "PANW", Shares: shares("1")},
			error: "lot of PANW has no day it was acquired on",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewTestingStore()

			assert.EqualError(t, s.SaveLot(tt.lot), tt.error)

			lots, err := s.Lots()
			require.NoError(t, err)
			assert.Empty(t, lots)
		})
	}
}

// TestStore_DeleteLot covers taking back a lot that was entered by mistake: it is gone from the
// list and the others stay. A lot that does not exist cannot be deleted, which says so rather than
// passing for done.
func TestStore_DeleteLot(t *testing.T) {
	s := NewTestingStore()
	require.NoError(t, s.SaveLot(Lot{Symbol: "PANW", Shares: shares("3"), Acquired: day("2026-03-15")}))
	require.NoError(t, s.SaveLot(Lot{Symbol: "AAPL", Shares: shares("1"), Acquired: day("2026-04-01")}))

	require.NoError(t, s.DeleteLot(1))

	lots, err := s.Lots()
	require.NoError(t, err)
	expected := []Lot{
		{ID: 2, Symbol: "AAPL", Shares: shares("1"), Acquired: day("2026-04-01")},
	}
	assert.Equal(t, expected, lots)

	assert.EqualError(t, s.DeleteLot(7), "no lot with ID 7")
}

// TestStore_SaveQuote covers the database as the cache of prices: a quote that is saved comes back
// under its symbol, and saving another for the same symbol replaces it, since only the latest price
// of a stock is worth anything.
func TestStore_SaveQuote(t *testing.T) {
	s := NewTestingStore()

	quotes, err := s.Quotes()
	require.NoError(t, err)
	assert.Empty(t, quotes)

	panw := Quote{
		Symbol:        "PANW",
		Price:         shares("396.25"),
		PreviousClose: shares("397.31"),
		Currency:      "USD",
		At:            timestamp("2026-10-02T08:30"),
	}
	aapl := Quote{
		Symbol:        "AAPL",
		Price:         shares("330.32"),
		PreviousClose: shares("333.02"),
		Currency:      "USD",
		At:            timestamp("2026-10-02T08:30"),
	}
	require.NoError(t, s.SaveQuote(panw))
	require.NoError(t, s.SaveQuote(aapl))

	quotes, err = s.Quotes()
	require.NoError(t, err)
	assert.Equal(t, map[string]Quote{"PANW": panw, "AAPL": aapl}, quotes)

	panw.Price = shares("401.5")
	panw.At = timestamp("2026-10-02T08:35")
	require.NoError(t, s.SaveQuote(panw))

	quotes, err = s.Quotes()
	require.NoError(t, err)
	assert.Equal(t, map[string]Quote{"PANW": panw, "AAPL": aapl}, quotes)
}

// TestStore_SaveGrant covers recording a grant: it comes back with its vests, each with an ID of
// its own, and a database that has none returns none.
func TestStore_SaveGrant(t *testing.T) {
	s := NewTestingStore()

	grants, err := s.Grants()
	require.NoError(t, err)
	assert.Empty(t, grants)

	err = s.SaveGrant(Grant{
		Name:   "Payout",
		Symbol: "PANW",
		Vests:  Repeating(day("2026-01-15"), 1, 3, shares("10")),
	})
	require.NoError(t, err)

	grants, err = s.Grants()
	require.NoError(t, err)
	expected := []Grant{
		{
			ID:     1,
			Name:   "Payout",
			Symbol: "PANW",
			Vests: []Vest{
				{ID: 1, Date: day("2026-01-15"), Shares: shares("10")},
				{ID: 2, Date: day("2026-02-15"), Shares: shares("10")},
				{ID: 3, Date: day("2026-03-15"), Shares: shares("10")},
			},
		},
	}
	assert.Equal(t, expected, grants)
}
