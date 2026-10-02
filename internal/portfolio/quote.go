package portfolio

import (
	"time"

	"github.com/shopspring/decimal"
)

// A Quote is what one share of a stock was worth at some point in time.
type Quote struct {
	Symbol string
	Price  decimal.Decimal

	// PreviousClose is the price at the end of the trading day before, which is what the change of
	// the day is measured against.
	PreviousClose decimal.Decimal

	// Currency is the ISO code of the currency both prices are in, such as USD.
	Currency string

	// At is when the price was read.
	At time.Time
}
