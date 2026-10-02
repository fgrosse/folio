package portfolio

import (
	"time"

	"github.com/shopspring/decimal"
)

// A Lot is a number of shares of one stock that are held and were all acquired on the same day,
// such as what one vest released or one order bought. Lots are what the current account value
// counts.
type Lot struct {
	ID     int
	Symbol string
	Shares decimal.Decimal

	// Acquired is the calendar day the shares arrived, at midnight UTC.
	Acquired time.Time
}
