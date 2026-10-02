package portfolio

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
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

// day parses a calendar day written as YYYY-MM-DD.
func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}

	return t
}

// shares parses a number of shares, or any other decimal, written the way it reads.
func shares(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}

	return d
}
