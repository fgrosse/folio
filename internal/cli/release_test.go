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

// TestReleaseCmd_WithoutSpace covers shares and cost given as one argument, with nothing between
// the shares and the "@", which reads the same as two.
func TestReleaseCmd_WithoutSpace(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "release", "2026-11-15", "6@380.12")
	seed(t, dbPath)
	require.NoError(t, cmd.Execute())

	all := lots(t, dbPath)
	require.Len(t, all, 3)
	assert.Equal(t, "6", all[2].Shares.String())
	assert.Equal(t, "380.12", all[2].Cost.String())
}

// TestReleaseCmd_WhichVest covers the days that do not name one vest: a day nothing vests on, or
// whose vest is released already, and a day on which two grants vest, which --grant settles.
func TestReleaseCmd_WhichVest(t *testing.T) {
	bonus := portfolio.Grant{
		Name:   "Bonus",
		Symbol: "PANW",
		Vests:  portfolio.Repeating(day("2026-11-15"), 12, 1, decimal.RequireFromString("50")),
	}

	cases := map[string]struct {
		args  []string
		error string
		lot   string // the grant the released lot is from, if the release goes through
	}{
		"a day nothing vests on": {
			args:  []string{"release", "2026-11-16", "6"},
			error: "no vest of 2026-11-16 is waiting to be released",
		},
		"a day two grants vest on": {
			args:  []string{"release", "2026-11-15", "6"},
			error: `2 grants vest on 2026-11-15: name one with --grant, "Payout" or "Bonus"`,
		},
		"the grant named": {
			args: []string{"release", "--grant", "Bonus", "2026-11-15", "25"},
			lot:  "Bonus",
		},
		"a grant that does not vest that day": {
			args:  []string{"release", "--grant", "Bonus", "2026-12-15", "6"},
			error: `no vest of "Bonus" on 2026-12-15 is waiting to be released`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cmd, dbPath := NewTestingCmd(t, c.args...)
			seed(t, dbPath)
			saveGrant(t, dbPath, bonus)

			err := cmd.Execute()
			if c.error != "" {
				assert.EqualError(t, err, c.error)
				assert.Len(t, lots(t, dbPath), 2, "nothing should be released")
				return
			}

			require.NoError(t, err)
			all := lots(t, dbPath)
			require.Len(t, all, 3)
			assert.Equal(t, c.lot, all[2].Grant)
		})
	}
}

// saveGrant saves grant to the database at dbPath.
func saveGrant(t *testing.T, dbPath string, grant portfolio.Grant) {
	t.Helper()

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	require.NoError(t, store.SaveGrant(grant))
}
