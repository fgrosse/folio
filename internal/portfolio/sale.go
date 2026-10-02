package portfolio

import (
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
