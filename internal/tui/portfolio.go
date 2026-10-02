package tui

import (
	"context"
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
	SaveLot(lot portfolio.Lot) error
	DeleteLot(id int) error
	SaveQuote(quote portfolio.Quote) error
	ReleaseVest(id int, shares decimal.Decimal) error
	SaveGrant(grant portfolio.Grant) error
	DeleteGrant(id int) error
}

// A Portfolio is everything the views show, as the store had it at one moment: the lots and grants
// of the account and the quotes they are valued at. Every view holds the latest one it was sent.
type Portfolio struct {
	Lots   []portfolio.Lot
	Grants []portfolio.Grant
	Quotes map[string]portfolio.Quote
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

	return Portfolio{Lots: lots, Grants: grants, Quotes: quotes}, nil
}
