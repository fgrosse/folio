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

// TestStatusCmd_CachesQuotes covers what status leaves behind: the quotes it fetched are saved to
// the database, where the TUI finds them the next time it opens, before it has fetched any itself.
func TestStatusCmd_CachesQuotes(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "status")
	cmd.quoter = quotes{"PANW": "396.25"}
	seed(t, dbPath)

	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	cached, err := store.Quotes()
	require.NoError(t, err)
	require.Contains(t, cached, "PANW")
	assert.Equal(t, "396.25", cached["PANW"].Price.String())
}

// TestStatusCmd_FallsBackOnCachedQuotes covers status without a network, or with a source of prices
// that is having a bad day: it values the account at the last quotes the database has, and says on
// stderr what went wrong, where it does not get in the way of whoever reads the values.
func TestStatusCmd_FallsBackOnCachedQuotes(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "status")
	cmd.quoter = quotes{} // knows no symbol at all
	seed(t, dbPath)
	saveQuote(t, dbPath, portfolio.Quote{Symbol: "PANW", Price: decimal.RequireFromString("400"), Currency: "USD"})

	var out, errOut strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	require.NoError(t, cmd.Execute())

	expected := "" +
		"Current     $3,400.00\n" +
		"Potential   $8,000.00\n" +
		"Total      $11,400.00\n"
	assert.Equal(t, expected, out.String())
	assert.Equal(t, "Warning: no quote of PANW\n", errOut.String())
}

// TestStatusCmd_JSON covers status for a program to read, such as a status bar widget: one JSON
// object with the three values as plain decimals to the cent, their currency, and the symbols that
// have no price, which is an empty list rather than null when there are none.
func TestStatusCmd_JSON(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "status", "--json")
	cmd.quoter = quotes{"PANW": "396.25"}
	seed(t, dbPath)

	var out strings.Builder
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())

	expected := `{"current":"3368.13","potential":"7925.00","total":"11293.13",` +
		`"realized":"0.00","realized_gain":"0.00","currency":"USD","unpriced":[]}` + "\n"
	assert.Equal(t, expected, out.String())
}

// TestStatusCmd_Realized covers status once shares were sold: the current value counts what is left
// of the lots, and what the sales brought in is stated on a line of its own, apart from the three
// values, since it is money that has left the account rather than a part of what the account is
// worth. The JSON output has it too, with the gain in it.
func TestStatusCmd_Realized(t *testing.T) {
	sell := func(t *testing.T, dbPath string) {
		t.Helper()

		store, err := portfolio.NewStore(dbPath)
		require.NoError(t, err)
		t.Cleanup(func() { _ = store.Close() })

		// The first lot of the seeded account has no cost, so only the release has a gain to state.
		require.NoError(t, store.ReleaseVest(1, decimal.RequireFromString("10"), decimal.RequireFromString("380")))
		require.NoError(t, store.SaveSale(portfolio.Sale{
			LotID: 1, Date: day("2026-09-15"), Shares: decimal.RequireFromString("2"), Price: decimal.RequireFromString("410.2"),
		}))
		require.NoError(t, store.SaveSale(portfolio.Sale{
			LotID: 3, Date: day("2026-11-20"), Shares: decimal.RequireFromString("4"), Price: decimal.RequireFromString("400"),
		}))
	}

	cmd, dbPath := NewTestingCmd(t, "status")
	cmd.quoter = quotes{"PANW": "396.25"}
	seed(t, dbPath)
	sell(t, dbPath)

	var out strings.Builder
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())

	// 12.5 shares are left of 8.5 + 10 - 6, and one vest of 10: 2 × 410.20 + 4 × 400 were realized.
	expected := "" +
		"Current    $4,953.13\n" +
		"Potential  $3,962.50\n" +
		"Total      $8,915.63\n" +
		"\n" +
		"Realized   $2,420.40\n"
	assert.Equal(t, expected, out.String())

	cmd, dbPath = NewTestingCmd(t, "status", "--json")
	cmd.quoter = quotes{"PANW": "396.25"}
	seed(t, dbPath)
	sell(t, dbPath)

	out.Reset()
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())

	expectedJSON := `{"current":"4953.13","potential":"3962.50","total":"8915.63",` +
		`"realized":"2420.40","realized_gain":"80.00","currency":"USD","unpriced":[]}` + "\n"
	assert.Equal(t, expectedJSON, out.String())
}

// saveQuote saves quote to the database at dbPath, as a quote cached by an earlier run.
func saveQuote(t *testing.T, dbPath string, quote portfolio.Quote) {
	t.Helper()

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	require.NoError(t, store.SaveQuote(quote))
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
