package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// quotes is a Quoter that knows the price of the symbols in it, and of no others.
type quotes map[string]string

func (q quotes) Quote(_ context.Context, symbol string) (portfolio.Quote, error) {
	price, ok := q[symbol]
	if !ok {
		return portfolio.Quote{}, errors.New("no quote of " + symbol)
	}

	return portfolio.Quote{Symbol: symbol, Price: dec(price), Currency: "USD"}, nil
}

// TestRefreshQuotesCmd covers how the views come by fresh prices: the command asks the quoter for a
// quote of every stock in the account, saves each to the store, which is the cache the next start
// opens with, and then loads the portfolio, so that what the views are sent has the new prices.
func TestRefreshQuotesCmd(t *testing.T) {
	p := testPortfolio()
	fresh := portfolio.Quote{Symbol: "PANW", Price: dec("401.5"), Currency: "USD"}

	store := new(MockStore)
	store.returns(p)
	store.On("SaveQuote", fresh).Return(nil)

	msg := runCmd(t, refreshQuotesCmd(store, quotes{"PANW": "401.5"}))

	loaded, ok := msg.(PortfolioLoadedMsg)
	require.True(t, ok, "the command should report a loaded portfolio, not %T", msg)
	assert.NoError(t, loaded.err)
	assert.NoError(t, loaded.quotesErr)
	assert.Equal(t, p, loaded.portfolio)
	store.AssertExpectations(t)
}

// TestLoadPortfolioCmd covers what the views are sent when they ask for the account: everything the
// store has of it, down to the rate that vests are taxed at.
func TestLoadPortfolioCmd(t *testing.T) {
	p := testPortfolio()
	store := new(MockStore)
	store.returns(p)

	msg := runCmd(t, loadPortfolioCmd(store))

	loaded, ok := msg.(PortfolioLoadedMsg)
	require.True(t, ok, "the command should report a loaded portfolio, not %T", msg)
	require.NoError(t, loaded.err)
	assert.Equal(t, p, loaded.portfolio)
	assert.Equal(t, "44.3", loaded.portfolio.TaxRate.Decimal.String())
	store.AssertExpectations(t)
}
