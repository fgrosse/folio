package cli

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestReleaseCmd covers releasing a vest from the command line, which is how a history of vests is
// caught up on from their confirmations: "folio release" takes the day of the vest, the shares that
// arrived and what one was worth that day, and turns the vest of that day into a lot.
func TestReleaseCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "release", "2026-11-15", "6", "@380.12")
	seed(t, dbPath) // a grant with vests of 10 shares on 2026-11-15 and 2026-12-15
	require.NoError(t, cmd.Execute())

	all := lots(t, dbPath)
	require.Len(t, all, 3)
	expected := portfolio.Lot{
		ID:       3,
		Symbol:   "PANW",
		Shares:   decimal.RequireFromString("6"),
		Acquired: day("2026-11-15"),
		Cost:     decimal.RequireFromString("380.12"),
		Grant:    "Payout",
	}
	assert.Equal(t, expected, all[2])
}
