package portfolio

import (
	"time"

	"github.com/shopspring/decimal"
)

// A Vest is one day on which shares of a grant vest. Until it is released into a lot, what it is
// worth counts as potential.
type Vest struct {
	ID int

	// Date is the calendar day the shares vest, at midnight UTC.
	Date   time.Time
	Shares decimal.Decimal
}

// Repeating returns the schedule of a grant that vests the same number of shares count times, the
// first time on the day first and then every so many months after it.
func Repeating(first time.Time, everyMonths, count int, shares decimal.Decimal) []Vest {
	vests := make([]Vest, count)
	for i := range vests {
		vests[i] = Vest{Date: first.AddDate(0, i*everyMonths, 0), Shares: shares}
	}

	return vests
}
