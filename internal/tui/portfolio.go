package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/fgrosse/folio/internal/portfolio"
)

// Store is the part of the portfolio storage that the views need.
type Store interface {
	Lots() ([]portfolio.Lot, error)
	Grants() ([]portfolio.Grant, error)
	Quotes() (map[string]portfolio.Quote, error)
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
	err       error
}

// loadPortfolioCmd returns a command that loads the portfolio from store.
func loadPortfolioCmd(store Store) tea.Cmd {
	return func() tea.Msg {
		p, err := loadPortfolio(store)
		return PortfolioLoadedMsg{portfolio: p, err: err}
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
