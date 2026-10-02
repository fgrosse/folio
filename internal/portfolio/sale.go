package portfolio

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// A Sale is a number of shares of one lot that were sold on one day, at one price. The lot keeps the
// shares it was acquired with, and what is left of it is those less its sales. What the sales bring
// in is the money that was actually realized, as opposed to what the shares still held are worth.
type Sale struct {
	ID int

	// LotID is the lot the shares were sold from.
	LotID int

	// Date is the calendar day of the sale, at midnight UTC.
	Date   time.Time
	Shares decimal.Decimal

	// Price is what one share sold for.
	Price decimal.Decimal

	// Note is whatever there is to remember about the sale, in as many lines as it takes.
	Note string

	// Symbol, Cost and Grant are those of the lot the shares were sold from: its stock, what one
	// share of it cost, which is zero if that is not known, and the name of the grant it was
	// released from, if any. The store fills them in when it lists sales, and ignores them when it
	// saves one.
	Symbol string
	Cost   decimal.Decimal
	Grant  string
}

// Validate says what a sale is missing to have happened: shares that were sold, a price they sold
// at, and the day of it.
func (s Sale) Validate() error {
	switch {
	case !s.Shares.IsPositive():
		return errors.New("a sale must have more than 0 shares")
	case !s.Price.IsPositive():
		return errors.New("a sale must have a price of more than 0")
	case s.Date.IsZero():
		return errors.New("sale has no day")
	}

	return nil
}

// Proceeds is what the sale brought in: its shares at the price they sold for.
func (s Sale) Proceeds() decimal.Decimal {
	return s.Shares.Mul(s.Price)
}

// Gain is how much of the proceeds is more than the shares cost, or less, as a negative gain. It is
// only known if the cost of the lot is.
func (s Sale) Gain() (gain decimal.Decimal, known bool) {
	if s.Cost.IsZero() {
		return decimal.Decimal{}, false
	}

	return s.Shares.Mul(s.Price.Sub(s.Cost)), true
}

// Realized is what sales have turned shares into: money, as opposed to the value of what is still
// held.
type Realized struct {
	// Proceeds is what the sales brought in.
	Proceeds decimal.Decimal

	// Gain is how much of the proceeds is more than the shares cost, of the sales whose cost is
	// known.
	Gain decimal.Decimal

	// Uncosted is how many sales have no known cost. Their gain is missing from Gain.
	Uncosted int
}

// NewRealized adds up what sales brought in and the gain in it, each to the cent.
func NewRealized(sales []Sale) Realized {
	var r Realized
	for _, sale := range sales {
		r.Proceeds = r.Proceeds.Add(sale.Proceeds())

		gain, known := sale.Gain()
		if !known {
			r.Uncosted++
			continue
		}

		r.Gain = r.Gain.Add(gain)
	}

	r.Proceeds = r.Proceeds.Round(2)
	r.Gain = r.Gain.Round(2)

	return r
}
