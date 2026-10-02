package cli

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// NewTestingCmd returns the folio command set up to run args against a database of its own in a
// temporary directory, and the path of that database for the test to look into afterwards.
func NewTestingCmd(t *testing.T, args ...string) (*Folio, string) {
	dbPath := filepath.Join(t.TempDir(), "folio.db")
	args = append([]string{"--db", dbPath}, args...)

	cmd := New()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	return cmd, dbPath
}

// TestLotCmd covers recording shares from the command line: "folio lot" takes a lot spec, quoted or
// as separate arguments, and saves that lot.
func TestLotCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "lot", "12.5", "panw", "2026-03-15")
	require.NoError(t, cmd.Execute())

	expected := []portfolio.Lot{
		{ID: 1, Symbol: "PANW", Shares: decimal.RequireFromString("12.5"), Acquired: day("2026-03-15")},
	}
	assert.Equal(t, expected, lots(t, dbPath))
}

// TestLotCmd_Today covers a lot without a day, which is the usual case of shares that arrived just
// now: it is of today, the calendar day where the user is, whatever day it is in UTC by then.
func TestLotCmd_Today(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "lot", "40 PANW")
	// Late in the evening in New York, which is the next day already in UTC.
	newYork := time.FixedZone("EDT", -4*60*60)
	cmd.now = func() time.Time { return time.Date(2026, time.October, 2, 22, 30, 0, 0, newYork) }

	require.NoError(t, cmd.Execute())

	expected := []portfolio.Lot{
		{ID: 1, Symbol: "PANW", Shares: decimal.RequireFromString("40"), Acquired: day("2026-10-02")},
	}
	assert.Equal(t, expected, lots(t, dbPath))
}

// lots returns the lots in the database at dbPath.
func lots(t *testing.T, dbPath string) []portfolio.Lot {
	t.Helper()

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	lots, err := store.Lots()
	require.NoError(t, err)

	return lots
}

// day parses a calendar day written as YYYY-MM-DD.
func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}

	return t
}
