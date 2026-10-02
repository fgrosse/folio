package portfolio

import (
	"path/filepath"
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

// TestStore_ReleaseVest covers the step that turns potential into current: releasing a vest makes a
// lot of its grant's stock, acquired on the day of the vest, and marks the vest as released. The
// lot holds the shares that actually arrived, which are fewer than vested whenever some were
// withheld for tax.
func TestStore_ReleaseVest(t *testing.T) {
	s := NewTestingStore()
	require.NoError(t, s.SaveGrant(Grant{
		Name:   "Payout",
		Symbol: "PANW",
		Vests:  Repeating(day("2026-01-15"), 1, 2, shares("10")),
	}))

	require.NoError(t, s.ReleaseVest(1, shares("6")))

	lots, err := s.Lots()
	require.NoError(t, err)
	// The lot says which grant it came from, which a lot entered by hand has nothing to say about.
	expectedLots := []Lot{
		{ID: 1, Symbol: "PANW", Shares: shares("6"), Acquired: day("2026-01-15"), Grant: "Payout"},
	}
	assert.Equal(t, expectedLots, lots)

	grants, err := s.Grants()
	require.NoError(t, err)
	expectedVests := []Vest{
		{ID: 1, Date: day("2026-01-15"), Shares: shares("10"), Released: true},
		{ID: 2, Date: day("2026-02-15"), Shares: shares("10")},
	}
	require.Len(t, grants, 1)
	assert.Equal(t, expectedVests, grants[0].Vests)
}

// TestStore_ReleaseVestRefusals covers the vests that cannot be released: one that does not exist,
// one that has been released already, and one that would release no shares or more than vested.
func TestStore_ReleaseVestRefusals(t *testing.T) {
	s := NewTestingStore()
	require.NoError(t, s.SaveGrant(Grant{
		Name:   "Payout",
		Symbol: "PANW",
		Vests:  Repeating(day("2026-01-15"), 1, 2, shares("10")),
	}))
	require.NoError(t, s.ReleaseVest(1, shares("6")))

	assert.EqualError(t, s.ReleaseVest(9, shares("6")), "no vest with ID 9")
	assert.EqualError(t, s.ReleaseVest(1, shares("6")), "the vest of 2026-01-15 is released already")
	assert.EqualError(t, s.ReleaseVest(2, shares("0")), "a vest must release more than 0 shares")
	assert.EqualError(t, s.ReleaseVest(2, shares("10.5")), "the vest of 2026-02-15 has 10 shares, not 10.5")

	lots, err := s.Lots()
	require.NoError(t, err)
	assert.Len(t, lots, 1, "a refused release must not make a lot")
}

// TestStore_DeleteGrant covers taking a grant back: it is gone with all of its vests, and the other
// grants stay as they were. A lot that one of its vests was released into stays too: those shares
// are held whether or not folio still knows where they came from.
func TestStore_DeleteGrant(t *testing.T) {
	s := NewTestingStore()
	require.NoError(t, s.SaveGrant(Grant{
		Name:   "Payout",
		Symbol: "PANW",
		Vests:  Repeating(day("2026-01-15"), 1, 2, shares("10")),
	}))
	require.NoError(t, s.SaveGrant(Grant{
		Name:   "Bonus",
		Symbol: "PANW",
		Vests:  Repeating(day("2026-06-01"), 12, 1, shares("50")),
	}))
	require.NoError(t, s.ReleaseVest(1, shares("6")))

	require.NoError(t, s.DeleteGrant(1))

	grants, err := s.Grants()
	require.NoError(t, err)
	expected := []Grant{
		{
			ID:     2,
			Name:   "Bonus",
			Symbol: "PANW",
			Vests:  []Vest{{ID: 3, Date: day("2026-06-01"), Shares: shares("50")}},
		},
	}
	assert.Equal(t, expected, grants)

	lots, err := s.Lots()
	require.NoError(t, err)
	assert.Len(t, lots, 1, "the lot of the released vest should stay")

	assert.EqualError(t, s.DeleteGrant(7), "no grant with ID 7")
}

// TestNewStore_SharesTheDatabase covers what lets the TUI and a status bar widget use one database
// file at the same time: the store opens it in WAL mode, where a reader does not block a writer, and
// waits a few seconds for a lock rather than failing at once with "database is locked".
func TestNewStore_SharesTheDatabase(t *testing.T) {
	s, err := NewStore(filepath.Join(t.TempDir(), "folio.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	var mode string
	require.NoError(t, s.db.Get(&mode, `PRAGMA journal_mode`))
	assert.Equal(t, "wal", mode)

	var timeout int
	require.NoError(t, s.db.Get(&timeout, `PRAGMA busy_timeout`))
	assert.Equal(t, 5000, timeout)
}

// TestStore_SaveLotWithCost covers the cost of a lot in the database: a lot saved with one comes
// back with it, and a lot saved without comes back without, rather than with a cost of nothing.
func TestStore_SaveLotWithCost(t *testing.T) {
	s := NewTestingStore()
	require.NoError(t, s.SaveLot(Lot{Symbol: "PANW", Shares: shares("6"), Acquired: day("2026-01-15"), Cost: shares("380.12")}))
	require.NoError(t, s.SaveLot(Lot{Symbol: "PANW", Shares: shares("2.5"), Acquired: day("2026-02-15")}))

	lots, err := s.Lots()
	require.NoError(t, err)
	expected := []Lot{
		{ID: 1, Symbol: "PANW", Shares: shares("6"), Acquired: day("2026-01-15"), Cost: shares("380.12")},
		{ID: 2, Symbol: "PANW", Shares: shares("2.5"), Acquired: day("2026-02-15")},
	}
	assert.Equal(t, expected, lots)
}
