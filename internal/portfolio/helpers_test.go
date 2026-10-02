package portfolio

import (
	"time"

	"github.com/shopspring/decimal"
)

// day parses a calendar day written as YYYY-MM-DD.
func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}

	return t
}

// shares parses a number of shares, or any other decimal, written the way it reads.
func shares(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}

	return d
}
