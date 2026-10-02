package portfolio

import (
	"errors"
	"fmt"
	"strings"
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

// lotSyntax is how a lot is written, for the errors that say so.
const lotSyntax = `a lot is written as "<shares> <symbol> [YYYY-MM-DD]"`

// NewLot parses a lot from its spec, "<shares> <symbol> [YYYY-MM-DD]", such as "12.5 PANW 2026-03-15".
// The symbol is written in capitals however it was typed. The day is optional: a lot without one
// comes back with a zero Acquired for the caller to fill in, who knows what today is. Apart from
// that day, the lot that comes back is valid.
func NewLot(spec string) (Lot, error) {
	fields := strings.Fields(spec)
	if len(fields) < 2 || len(fields) > 3 {
		return Lot{}, errors.New(lotSyntax)
	}

	shares, err := decimal.NewFromString(fields[0])
	if err != nil {
		return Lot{}, fmt.Errorf("%q is not a number of shares", fields[0])
	}

	lot := Lot{Symbol: strings.ToUpper(fields[1]), Shares: shares}
	if !lot.Shares.IsPositive() {
		return Lot{}, lot.Validate()
	}

	if len(fields) == 3 {
		lot.Acquired, err = ParseDay(fields[2])
		if err != nil {
			return Lot{}, err
		}
	}

	return lot, nil
}

// ParseDay parses a calendar day written as YYYY-MM-DD.
func ParseDay(s string) (time.Time, error) {
	day, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q is not a day written as YYYY-MM-DD", s)
	}

	return day, nil
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
