package portfolio

import (
	"errors"
	"fmt"
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

// Validate says what a lot is missing to be worth something: a symbol to look its price up by, more
// than no shares to multiply it with, and the day they arrived.
func (l Lot) Validate() error {
	switch {
	case l.Symbol == "":
		return errors.New("lot has no symbol")
	case !l.Shares.IsPositive():
		return fmt.Errorf("lot of %s must have more than 0 shares", l.Symbol)
	case l.Acquired.IsZero():
		return fmt.Errorf("lot of %s has no day it was acquired on", l.Symbol)
	}

	return nil
}
