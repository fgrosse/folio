package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestStatusCmd covers what "folio status" is for: the three values of the account, at the prices
// of right now, the labels and the amounts each lined up in a column.
func TestStatusCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "status")
	cmd.quoter = quotes{"PANW": "396.25"}
	seed(t, dbPath)

	var out strings.Builder
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())

	// 8.5 shares held and 20 still to vest, at 396.25.
	expected := "" +
		"Current     $3,368.13\n" +
		"Potential   $7,925.00\n" +
		"Total      $11,293.13\n"
	assert.Equal(t, expected, out.String())
}

// seed fills the database at dbPath with an account of PANW stock: two lots of 8.5 shares in all,
// and a grant with two vests of 10 shares each that have not been released.
func seed(t *testing.T, dbPath string) {
	t.Helper()

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Migrate())

	require.NoError(t, store.SaveLot(portfolio.Lot{Symbol: "PANW", Shares: decimal.RequireFromString("6"), Acquired: day("2026-01-15")}))
	require.NoError(t, store.SaveLot(portfolio.Lot{Symbol: "PANW", Shares: decimal.RequireFromString("2.5"), Acquired: day("2026-02-15")}))
	require.NoError(t, store.SaveGrant(portfolio.Grant{
		Name:   "Payout",
		Symbol: "PANW",
		Vests:  portfolio.Repeating(day("2026-11-15"), 1, 2, decimal.RequireFromString("10")),
	}))
}

// quotes is a Quoter that knows the price of the symbols in it, and of no others.
type quotes map[string]string

func (q quotes) Quote(_ context.Context, symbol string) (portfolio.Quote, error) {
	price, ok := q[symbol]
	if !ok {
		return portfolio.Quote{}, errors.New("no quote of " + symbol)
	}

	return portfolio.Quote{Symbol: symbol, Price: decimal.RequireFromString(price), Currency: "USD"}, nil
}
