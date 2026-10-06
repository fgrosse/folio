package demo

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// today is the day the demo accounts of the tests are made on.
var today = time.Date(2026, time.October, 2, 0, 0, 0, 0, time.UTC)

// newDemo returns a store that only lives in memory, filled with the demo account of seed.
func newDemo(t *testing.T, seed uint64) *portfolio.SQLiteStore {
	t.Helper()

	store, err := portfolio.NewStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Migrate())

	require.NoError(t, Fill(store, rand.New(rand.NewPCG(seed, seed)), today))

	return store
}

// TestFill covers what a demo account has to have for every view of folio to show something: grants
// with vests still to come and at least one that is due and pending release, lots that were released
// from them at a cost, a lot that was bought, a sale, a quote of every stock in it, so that the
// account has all of its values without a network, and a rate that the vests are taxed at.
func TestFill(t *testing.T) {
	for seed := range uint64(20) {
		store := newDemo(t, seed)

		grants, err := store.Grants()
		require.NoError(t, err)
		lots, err := store.Lots()
		require.NoError(t, err)
		sales, err := store.Sales()
		require.NoError(t, err)
		quotes, err := store.Quotes()
		require.NoError(t, err)
		taxRate, err := store.GetConfig(portfolio.TaxRateKey)
		require.NoError(t, err, "seed %d: the account should have a tax rate", seed)
		gainsTaxRate, err := store.GetConfig(portfolio.GainsTaxRateKey)
		require.NoError(t, err, "seed %d: the account should have a gains tax rate", seed)

		require.GreaterOrEqual(t, len(grants), 2, "seed %d", seed)

		var pending, coming int
		for _, grant := range grants {
			for _, vest := range grant.Vests {
				switch {
				case vest.Released:
				case vest.Date.After(today):
					coming++
				default:
					pending++
				}
			}
		}
		assert.Positive(t, coming, "seed %d: vests to come", seed)
		assert.Positive(t, pending, "seed %d: vests pending release", seed)

		var released, bought int
		for _, lot := range lots {
			assert.True(t, lot.Cost.IsPositive(), "seed %d: every lot should have a cost", seed)
			assert.False(t, lot.Acquired.After(today), "seed %d: no lot should be of the future", seed)
			if lot.Grant != "" {
				released++
			} else {
				bought++
			}
		}
		assert.Positive(t, released, "seed %d: lots released from grants", seed)
		assert.Positive(t, bought, "seed %d: lots that were bought", seed)

		assert.NotEmpty(t, sales, "seed %d", seed)

		account := portfolio.NewAccount(lots, grants, quotes)
		assert.Empty(t, account.Unpriced, "seed %d: every stock should have a quote", seed)
		assert.True(t, account.Current.IsPositive(), "seed %d", seed)
		assert.True(t, account.Potential.IsPositive(), "seed %d", seed)

		_, err = portfolio.ParseTaxRate(taxRate)
		assert.NoError(t, err, "seed %d", seed)
		_, err = portfolio.ParseTaxRate(gainsTaxRate)
		assert.NoError(t, err, "seed %d", seed)
	}
}

// summary renders what an account holds as text, for two accounts to be compared by.
func summary(t *testing.T, store *portfolio.SQLiteStore) string {
	t.Helper()

	grants, err := store.Grants()
	require.NoError(t, err)
	lots, err := store.Lots()
	require.NoError(t, err)
	sales, err := store.Sales()
	require.NoError(t, err)

	var s string
	for _, grant := range grants {
		s += grant.Name + " " + grant.Symbol + "\n"
		for _, vest := range grant.Vests {
			s += " " + vest.Date.Format(time.DateOnly) + " " + vest.Shares.String() + "\n"
		}
	}
	for _, lot := range lots {
		s += lot.String() + "\n"
	}
	for _, sale := range sales {
		s += sale.Date.Format(time.DateOnly) + " " + sale.Shares.String() + " @" + sale.Price.String() + "\n"
	}

	return s
}

// TestFill_Seed covers what the random numbers are for: the same ones make the same account, so
// that an account can be made again, and others make another, so that not every demo looks alike.
func TestFill_Seed(t *testing.T) {
	first := summary(t, newDemo(t, 7))

	assert.Equal(t, first, summary(t, newDemo(t, 7)))
	assert.NotEqual(t, first, summary(t, newDemo(t, 8)))
}
