package portfolio

import (
	"errors"
	"fmt"
	"regexp"
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

	// Cost is what one share of the lot cost when it was acquired: the price it was bought at, or
	// what it was worth on the day it vested. A gain is measured from it, and with it the tax that is
	// due. It is zero if it is not known.
	Cost decimal.Decimal

	// Sold is how many of the shares have been sold since, in all sales of the lot. The store fills
	// it in when it lists lots, and ignores it when it saves one.
	Sold decimal.Decimal

	// Grant is the name of the grant the lot was released from, and empty for a lot that was entered
	// by hand. The store fills it in when it lists lots, and ignores it when it saves one.
	Grant string
}

// lotSyntax is how a lot is written, for the errors that say so.
const lotSyntax = `a lot is written as "<shares> <symbol> [YYYY-MM-DD] [@<cost>]"`

// NewLot parses a lot from its spec, "<shares> <symbol> [YYYY-MM-DD] [@<cost>]", such as
// "12.5 PANW 2026-03-15 @380.12". The symbol is written in capitals however it was typed.
//
// The day and the cost are optional, and come in either order. The cost needs no space before its
// "@". A lot without a day comes back with
// a zero Acquired for the caller to fill in, who knows what today is, and a lot without a cost with
// a zero Cost, which says that it is not known. Apart from that day, the lot that comes back is
// valid.
func NewLot(spec string) (Lot, error) {
	fields := specFields(spec)
	if len(fields) < 2 || len(fields) > 4 {
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

	for _, field := range fields[2:] {
		switch isCost := strings.HasPrefix(field, "@"); {
		case isCost && !lot.Cost.IsZero(), !isCost && !lot.Acquired.IsZero():
			return Lot{}, errors.New(lotSyntax) // a second cost, or a second day
		case isCost:
			lot.Cost, err = ParseCost(field)
		default:
			lot.Acquired, err = ParseDay(field)
		}
		if err != nil {
			return Lot{}, err
		}
	}

	return lot, nil
}

// ParseRelease parses what is typed when a vest is released, "<shares> [@<cost>]", such as
// "250 @162.50": the shares that arrived, which have to be more than none, and what one of them was
// worth that day, which is zero if the spec does not say.
func ParseRelease(spec string) (shares, cost decimal.Decimal, err error) {
	fields := specFields(spec)
	if len(fields) < 1 || len(fields) > 2 {
		return shares, cost, errors.New(`a release is written as "<shares> [@<cost>]"`)
	}

	shares, err = decimal.NewFromString(fields[0])
	switch {
	case err != nil:
		return shares, cost, fmt.Errorf("%q is not a number of shares", fields[0])
	case !shares.IsPositive():
		return shares, cost, errors.New("a vest must release more than 0 shares")
	}

	cost = decimal.Zero
	if len(fields) == 2 {
		cost, err = ParseCost(fields[1])
	}

	return shares, cost, err
}

// costSign finds the "@" that a cost is written after, with whatever space there is around it.
var costSign = regexp.MustCompile(`\s*@\s*`)

// specFields splits a spec into its fields, which are set apart by space. A cost is a field of its
// own that starts with its "@", however the "@" was typed: right after the field before it, as in
// "250@162.50", or with space after it.
func specFields(spec string) []string {
	return strings.Fields(costSign.ReplaceAllString(spec, " @"))
}

// ParseCost parses what a share cost, written after an "@" such as "@380.12", which has to be more
// than nothing.
func ParseCost(s string) (decimal.Decimal, error) {
	cost, err := decimal.NewFromString(strings.TrimPrefix(s, "@"))
	if err != nil || !strings.HasPrefix(s, "@") || !cost.IsPositive() {
		return decimal.Decimal{}, fmt.Errorf("%q is not a cost such as @380.12", s)
	}

	return cost, nil
}

// Remaining is how many shares of the lot are still held: the ones it was acquired with, less the
// ones that were sold.
func (l Lot) Remaining() decimal.Decimal {
	return l.Shares.Sub(l.Sold)
}

// String renders the lot as its spec, in the syntax NewLot reads: "<shares> <symbol> <YYYY-MM-DD>",
// and "@<cost>" after it if the lot has a cost.
func (l Lot) String() string {
	spec := l.Shares.String() + " " + l.Symbol + " " + l.Acquired.Format(time.DateOnly)
	if !l.Cost.IsZero() {
		spec += " @" + l.Cost.String()
	}

	return spec
}

// DayOf returns the calendar day that t falls on where t is, at midnight UTC, which is how every
// day in the portfolio is held. Late in the evening it is still today, even though it is tomorrow in
// UTC by then.
func DayOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
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
