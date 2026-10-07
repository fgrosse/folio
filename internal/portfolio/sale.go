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

	// Cleared is whether the tax on the gain of the sale is settled with the tax office, which it
	// is once the tax return of its year is done. Until then that tax is owed.
	Cleared bool

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

// Tax is the tax that is due on the gain of the sale at rate, in percent, to the cent. Like the gain
// it is only known if the cost of the lot is. Every sale is taxed on its own gain, as every lot that
// is held is: a sale at a loss is not taxed and takes nothing off the tax on another.
func (s Sale) Tax(rate decimal.Decimal) (tax decimal.Decimal, known bool) {
	gain, known := s.Gain()
	if !known {
		return decimal.Decimal{}, false
	}

	return GainsTax(gain, rate), true
}

// TaxOwed adds up the tax at rate, in percent, on those of sales that are not cleared: what the tax
// office is still to get for them. A sale whose cost is not known adds nothing, since its tax is
// not known either.
func TaxOwed(sales []Sale, rate decimal.Decimal) decimal.Decimal {
	var owed decimal.Decimal
	for _, sale := range sales {
		if sale.Cleared {
			continue
		}

		// A tax that is not known is zero, which adds nothing.
		tax, _ := sale.Tax(rate)
		owed = owed.Add(tax)
	}

	return owed
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
