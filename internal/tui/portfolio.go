package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// Store is the part of the portfolio storage that the views need.
type Store interface {
	Lots() ([]portfolio.Lot, error)
	Grants() ([]portfolio.Grant, error)
	Quotes() (map[string]portfolio.Quote, error)
	GetConfig(key string) (string, error)
	SetConfig(key, value string) error
	UnsetConfig(key string) error
	SaveLot(lot portfolio.Lot) error
	DeleteLot(id int) error
	SaveQuote(quote portfolio.Quote) error
	ReleaseVest(id int, shares, cost decimal.Decimal) error
	SaveGrant(grant portfolio.Grant) error
	DeleteGrant(id int) error
	Sales() ([]portfolio.Sale, error)
	SaveSale(sale portfolio.Sale) error
	DeleteSale(id int) error
}

// A Portfolio is everything the views show, as the store had it at one moment: the lots and grants
// of the account, the quotes they are valued at, the sales that took shares out of the lots, the
// rates that vests and the gain of a sale are taxed at, and whether the potential and the current
// value are shown before or after tax. Every view holds the latest one it was sent.
type Portfolio struct {
	Lots   []portfolio.Lot
	Grants []portfolio.Grant
	Quotes map[string]portfolio.Quote
	Sales  []portfolio.Sale

	// TaxRate is the rate in percent that vests are taxed at, not valid if none was set.
	TaxRate decimal.NullDecimal

	// GainsTaxRate is the rate in percent that the gain of a sale is taxed at, not valid if none was
	// set.
	GainsTaxRate decimal.NullDecimal

	// PotentialBasis is whether the header shows the potential value before tax or after it.
	PotentialBasis portfolio.Basis

	// CurrentBasis is whether the header shows the current value before the tax on its gains or
	// after it.
	CurrentBasis portfolio.Basis
}

// PortfolioLoadedMsg reports the result of loading the portfolio from the Store. Every view
// receives it, whichever of them asked for it, since they all show the same account.
type PortfolioLoadedMsg struct {
	portfolio Portfolio
	err       error // why the portfolio could not be loaded, in which case there is none

	// quotesErr is why the quotes could not be refreshed before the portfolio was loaded, or not all
	// of them. The portfolio is as good as any other then, with the quotes the store had.
	quotesErr error
}

// refreshTimeout is how long fetching the quotes of the account may take before the views go on
// with the ones they have.
const refreshTimeout = 20 * time.Second

// loadPortfolioCmd returns a command that loads the portfolio from store.
func loadPortfolioCmd(store Store) tea.Cmd {
	return func() tea.Msg {
		p, err := loadPortfolio(store)
		return PortfolioLoadedMsg{portfolio: p, err: err}
	}
}

// refreshQuotesCmd returns a command that fetches the latest quote of every stock in the account
// from quoter, saves them to store, and then loads the portfolio, which is valued at them. The store
// is the cache of the quotes, so the next start opens with what this one fetched.
func refreshQuotesCmd(store Store, quoter portfolio.Quoter) tea.Cmd {
	return func() tea.Msg {
		p, err := loadPortfolio(store)
		if err != nil {
			return PortfolioLoadedMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
		defer cancel()

		fetched, quotesErr := portfolio.FetchQuotes(ctx, quoter, portfolio.Symbols(p.Lots, p.Grants))
		for symbol, quote := range fetched {
			if err := store.SaveQuote(quote); err != nil {
				return PortfolioLoadedMsg{err: fmt.Errorf("save quote of %s: %w", symbol, err)}
			}
		}

		p, err = loadPortfolio(store)
		return PortfolioLoadedMsg{portfolio: p, err: err, quotesErr: quotesErr}
	}
}

func loadPortfolio(store Store) (Portfolio, error) {
	lots, err := store.Lots()
	if err != nil {
		return Portfolio{}, err
	}

	grants, err := store.Grants()
	if err != nil {
		return Portfolio{}, err
	}

	quotes, err := store.Quotes()
	if err != nil {
		return Portfolio{}, err
	}

	sales, err := store.Sales()
	if err != nil {
		return Portfolio{}, err
	}

	taxRate, err := loadTaxRate(store, portfolio.TaxRateKey)
	if err != nil {
		return Portfolio{}, err
	}

	gainsTaxRate, err := loadTaxRate(store, portfolio.GainsTaxRateKey)
	if err != nil {
		return Portfolio{}, err
	}

	potentialBasis, err := loadBasis(store, portfolio.PotentialBasisKey)
	if err != nil {
		return Portfolio{}, err
	}

	currentBasis, err := loadBasis(store, portfolio.CurrentBasisKey)
	if err != nil {
		return Portfolio{}, err
	}

	return Portfolio{
		Lots:           lots,
		Grants:         grants,
		Quotes:         quotes,
		Sales:          sales,
		TaxRate:        taxRate,
		GainsTaxRate:   gainsTaxRate,
		PotentialBasis: potentialBasis,
		CurrentBasis:   currentBasis,
	}, nil
}

// loadTaxRate returns the tax rate that is set under key, which is not valid if the account has
// none. The store keeps it as text among the configuration, as folio config set it.
func loadTaxRate(store Store, key string) (decimal.NullDecimal, error) {
	value, err := store.GetConfig(key)
	switch {
	case errors.Is(err, portfolio.ErrNotSet):
		return decimal.NullDecimal{}, nil
	case err != nil:
		return decimal.NullDecimal{}, err
	}

	rate, err := portfolio.ParseTaxRate(value)
	if err != nil {
		return decimal.NullDecimal{}, fmt.Errorf("%s: %w", key, err)
	}

	return decimal.NewNullDecimal(rate), nil
}

// loadBasis returns which of a value the header shows, as it is set under key. An account that has
// not said shows the bank's, before tax, which is the number its web site has.
func loadBasis(store Store, key string) (portfolio.Basis, error) {
	value, err := store.GetConfig(key)
	switch {
	case errors.Is(err, portfolio.ErrNotSet):
		return portfolio.Gross, nil
	case err != nil:
		return "", err
	}

	basis, err := portfolio.ParseBasis(value)
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}

	return basis, nil
}
