// Package demo makes up an account for folio to show, for trying it out without entering one.
package demo

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// Store is the part of the portfolio storage that a demo account is written to.
type Store interface {
	SaveQuote(quote portfolio.Quote) error
	SaveGrant(grant portfolio.Grant) error
	Grants() ([]portfolio.Grant, error)
	ReleaseVest(id int, shares, cost decimal.Decimal) error
	SaveLot(lot portfolio.Lot) error
	Lots() ([]portfolio.Lot, error)
	SaveSale(sale portfolio.Sale) error
	SetTaxRate(rate decimal.Decimal) error
}

// A stock is one that a demo account may hold, with a price to value it at until a real quote of it
// is fetched. The symbols are real ones, so that the quotes folio fetches replace the made-up price.
type stock struct {
	symbol string
	price  float64
}

var (
	// employers are the stocks the grants of a demo account may be of.
	employers = []stock{{"MSFT", 430}, {"NVDA", 135}, {"GOOGL", 175}, {"CRM", 290}, {"ADBE", 480}}

	// others are the stocks a demo account may have bought on its own.
	others = []stock{{"AAPL", 230}, {"AMZN", 200}, {"COST", 900}}
)

// Fill writes a made-up account to store, as it could look on the day today: someone's stock from
// work, in a grant that came with the job and vests every quarter and a second that pays out every
// month, most of what has vested released and some of it sold, and a little stock they bought
// themselves. Every stock in it gets a quote, so the account has its values without a network, and
// the account gets a tax rate, so that the vests have a value after tax.
//
// What the account holds comes out of rng, so two accounts of the same random numbers are the same
// account, and others differ in the stock, the size and the age of everything. store should be
// empty.
func Fill(store Store, rng *rand.Rand, today time.Time) error {
	g := generator{store: store, rng: rng, today: today}

	employer := employers[rng.IntN(len(employers))]
	other := others[rng.IntN(len(others))]

	for _, s := range []stock{employer, other} {
		if err := g.saveQuote(s); err != nil {
			return fmt.Errorf("save quote: %w", err)
		}
	}

	if err := g.saveGrants(employer); err != nil {
		return fmt.Errorf("save grants: %w", err)
	}

	if err := g.releaseVests(employer); err != nil {
		return fmt.Errorf("release vests: %w", err)
	}

	if err := g.buy(other); err != nil {
		return fmt.Errorf("save lot: %w", err)
	}

	if err := g.sell(); err != nil {
		return fmt.Errorf("save sale: %w", err)
	}

	if err := store.SetTaxRate(taxRate); err != nil {
		return fmt.Errorf("set tax rate: %w", err)
	}

	return nil
}

// taxRate is the rate in percent that the vests of a demo account are taxed at, about what the top
// of a salary that comes with stock is taxed at in much of Europe.
var taxRate = decimal.NewFromInt(42)

// A generator makes up the parts of a demo account and writes them to its store.
type generator struct {
	store Store
	rng   *rand.Rand
	today time.Time
}

// saveQuote saves a quote of s at its price, with a previous close up to two percent off it.
func (g *generator) saveQuote(s stock) error {
	return g.store.SaveQuote(portfolio.Quote{
		Symbol:        s.symbol,
		Price:         dollars(s.price),
		PreviousClose: dollars(s.price * g.between(0.98, 1.02)),
		Currency:      "USD",
		At:            g.today,
	})
}

// saveGrants saves the two grants of the account, both of the employer's stock. The first came with
// the job: it vests every quarter over four years, evenly or more with every year, and began one to
// two years ago. The second pays out the same number of shares every month and began within the
// last year. Both vest on the first of a month, so each has a vest in the month of today.
func (g *generator) saveGrants(employer stock) error {
	month := time.Date(g.today.Year(), g.today.Month(), 1, 0, 0, 0, 0, time.UTC)

	percentPerYear := []int{25, 25, 25, 25}
	if g.rng.IntN(2) == 0 {
		percentPerYear = []int{10, 20, 30, 40}
	}

	hire := portfolio.Grant{
		Name:   "New hire grant",
		Symbol: employer.symbol,
		Vests: portfolio.Graded(
			month.AddDate(0, -3*(4+g.rng.IntN(4)), 0),
			3,
			decimal.NewFromInt(int64(16*(25+g.rng.IntN(60)))),
			percentPerYear,
		),
	}
	if err := g.store.SaveGrant(hire); err != nil {
		return err
	}

	payout := portfolio.Grant{
		Name:   "Acquisition payout",
		Symbol: employer.symbol,
		Vests: portfolio.Repeating(
			month.AddDate(0, -(3+g.rng.IntN(7)), 0),
			1,
			24+12*g.rng.IntN(2),
			decimal.NewFromInt(int64(8+g.rng.IntN(23))),
		),
	}

	return g.store.SaveGrant(payout)
}

// releaseVests releases the vests that are due, each into a lot of somewhat more than half its
// shares, as is left once the rest was sold for tax, at what the stock may have been worth that day.
// The latest vest of the first grant stays as it is, due and pending release, so that the account
// has one to release.
func (g *generator) releaseVests(employer stock) error {
	grants, err := g.store.Grants()
	if err != nil {
		return err
	}

	for i, grant := range grants {
		var due []portfolio.Vest
		for _, vest := range grant.Vests {
			if !vest.Date.After(g.today) {
				due = append(due, vest)
			}
		}

		if i == 0 && len(due) > 0 {
			due = due[:len(due)-1]
		}

		for _, vest := range due {
			arrived := vest.Shares.Mul(decimal.NewFromFloat(g.between(0.52, 0.6))).Floor()
			if !arrived.IsPositive() {
				arrived = decimal.NewFromInt(1)
			}

			if err := g.store.ReleaseVest(vest.ID, arrived, g.priceOn(employer, vest.Date)); err != nil {
				return err
			}
		}
	}

	return nil
}

// buy saves a lot of a stock the account bought on its own, some months to a few years ago.
func (g *generator) buy(other stock) error {
	acquired := g.today.AddDate(0, -(6 + g.rng.IntN(25)), -g.rng.IntN(27))

	return g.store.SaveLot(portfolio.Lot{
		Symbol:   other.symbol,
		Shares:   decimal.NewFromInt(int64(5 + g.rng.IntN(36))),
		Acquired: acquired,
		Cost:     g.priceOn(other, acquired),
	})
}

// sell saves a sale of a part of the first lot that came from a grant, halfway between then and
// today and at a gain, with a note of two lines.
func (g *generator) sell() error {
	lots, err := g.store.Lots()
	if err != nil {
		return err
	}

	for _, lot := range lots {
		if lot.Grant == "" {
			continue
		}

		shares := lot.Shares.Mul(decimal.NewFromFloat(g.between(0.3, 0.7))).Floor()
		if !shares.IsPositive() {
			shares = lot.Shares
		}

		return g.store.SaveSale(portfolio.Sale{
			LotID:  lot.ID,
			Date:   lot.Acquired.Add(g.today.Sub(lot.Acquired) / 2).Truncate(24 * time.Hour),
			Shares: shares,
			Price:  lot.Cost.Mul(decimal.NewFromFloat(g.between(1.05, 1.45))).Round(2),
			Note:   "Down payment for the car\nSold in one order, the day after the earnings call",
		})
	}

	return nil
}

// priceOn makes up what a share of s was worth on a day in the past: less the longer ago it was,
// by about a percent a month, give or take a few, and never less than two fifths of its price now.
func (g *generator) priceOn(s stock, date time.Time) decimal.Decimal {
	months := g.today.Sub(date).Hours() / 24 / 30
	factor := max(1-0.012*months+g.between(-0.05, 0.05), 0.4)

	return dollars(s.price * factor)
}

// between returns a random number from low up to high.
func (g *generator) between(low, high float64) float64 {
	return low + g.rng.Float64()*(high-low)
}

// dollars is an amount as a decimal to the cent.
func dollars(amount float64) decimal.Decimal {
	return decimal.NewFromFloat(amount).Round(2)
}
