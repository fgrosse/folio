package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// TestLoadPortfolioCmd_TaxRate covers the rate that vests are taxed at, which the store keeps as
// text among the configuration: an account without one has no rate rather than failing to load,
// and a value that is no rate is an error that says which key it is in.
func TestLoadPortfolioCmd_TaxRate(t *testing.T) {
	p := testPortfolio()
	p.TaxRate = decimal.NullDecimal{}
	store := new(MockStore)
	store.returns(p)

	loaded, ok := runCmd(t, loadPortfolioCmd(store)).(PortfolioLoadedMsg)
	require.True(t, ok)
	require.NoError(t, loaded.err)
	assert.False(t, loaded.portfolio.TaxRate.Valid, "an account without a rate should have none")

	store = new(MockStore)
	store.returnsAccount(testPortfolio())
	store.On("GetConfig", portfolio.TaxRateKey).Return("high", nil)

	loaded, ok = runCmd(t, loadPortfolioCmd(store)).(PortfolioLoadedMsg)
	require.True(t, ok)
	assert.EqualError(t, loaded.err, `tax-rate: "high" is not a tax rate such as 44.3%`)
}

// TestLoadPortfolioCmd_GainsTaxRate covers the rate that the gain of a sale is taxed at, which is
// kept and loaded like the rate of the vests: it is part of the portfolio, an account without one
// has none, and a value that is no rate is an error that says which key it is in.
func TestLoadPortfolioCmd_GainsTaxRate(t *testing.T) {
	p := testPortfolio()
	store := new(MockStore)
	store.returns(p)

	loaded, ok := runCmd(t, loadPortfolioCmd(store)).(PortfolioLoadedMsg)
	require.True(t, ok)
	require.NoError(t, loaded.err)
	assert.Equal(t, "26.4", loaded.portfolio.GainsTaxRate.Decimal.String())

	p.GainsTaxRate = decimal.NullDecimal{}
	store = new(MockStore)
	store.returns(p)

	loaded, ok = runCmd(t, loadPortfolioCmd(store)).(PortfolioLoadedMsg)
	require.True(t, ok)
	require.NoError(t, loaded.err)
	assert.False(t, loaded.portfolio.GainsTaxRate.Valid, "an account without a rate should have none")

	store = new(MockStore)
	store.returnsAccount(testPortfolio())
	store.On("GetConfig", portfolio.GainsTaxRateKey).Return("high", nil)
	store.On("GetConfig", mock.Anything).Return("", portfolio.ErrNotSet)

	loaded, ok = runCmd(t, loadPortfolioCmd(store)).(PortfolioLoadedMsg)
	require.True(t, ok)
	assert.EqualError(t, loaded.err, `gains-tax-rate: "high" is not a tax rate such as 44.3%`)
}

// TestLoadPortfolioCmd_PotentialBasis covers which potential value the header shows, which the
// store keeps among the configuration: the bank's, before tax, unless the account says net, and a
// value that is neither is an error that says which key it is in.
func TestLoadPortfolioCmd_PotentialBasis(t *testing.T) {
	tests := map[string]struct {
		value    string
		err      error
		expected portfolio.Basis
		error    string
	}{
		"not set": {err: portfolio.ErrNotSet, expected: portfolio.Gross},
		"gross":   {value: "gross", expected: portfolio.Gross},
		"net":     {value: "net", expected: portfolio.Net},
		"neither": {value: "after-tax", error: `potential: "after-tax" is neither gross nor net`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			store := new(MockStore)
			store.returnsAccount(testPortfolio())
			store.On("GetConfig", portfolio.TaxRateKey).Return("44.3%", nil)
			store.On("GetConfig", portfolio.PotentialBasisKey).Return(tt.value, tt.err)
			store.On("GetConfig", portfolio.GainsTaxRateKey).Return("", portfolio.ErrNotSet).Maybe()

			loaded, ok := runCmd(t, loadPortfolioCmd(store)).(PortfolioLoadedMsg)
			require.True(t, ok)
			if tt.error != "" {
				assert.EqualError(t, loaded.err, tt.error)
				return
			}

			require.NoError(t, loaded.err)
			assert.Equal(t, tt.expected, loaded.portfolio.PotentialBasis)
		})
	}
}
