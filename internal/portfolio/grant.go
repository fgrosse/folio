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
		vests[i] = Vest{Date: addMonths(first, i*everyMonths), Shares: shares}
	}

	return vests
}

// addMonths returns the day that many months after day: the same day of the month, or the last day
// of a month that is too short to have it. time.Time.AddDate would spill over into the month after
// instead, and turn the 31st of January into the 3rd of March.
func addMonths(day time.Time, months int) time.Time {
	first := time.Date(day.Year(), day.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	lastDay := first.AddDate(0, 1, -1).Day()

	return first.AddDate(0, 0, min(day.Day(), lastDay)-1)
}
