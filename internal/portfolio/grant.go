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

// Graded returns the schedule of a grant of total shares that vests a different share of them in
// each year: percentPerYear has the percentage of every year, in order, which is spread evenly over
// the vests of that year. The first vest is on the day first, and the others follow every so many
// months, which have to divide a year.
//
// Only whole shares vest. Each vest brings what has vested in all up to what the schedule says for
// that day, rounded down, so a fraction a vest leaves behind arrives with a later one and the vests
// add up to the grant.
func Graded(first time.Time, everyMonths int, total decimal.Decimal, percentPerYear []int) []Vest {
	vestsPerYear := 12 / everyMonths

	var (
		vests   []Vest
		vested  decimal.Decimal // the shares of the vests so far
		percent int             // the percentage of the years before the one being laid out
	)
	for year, percentOfYear := range percentPerYear {
		for i := range vestsPerYear {
			// What has vested after this vest, in percent of the grant times vestsPerYear, which
			// keeps the fraction of a year a whole number until the one division below.
			share := percent*vestsPerYear + percentOfYear*(i+1)
			due := total.Mul(decimal.NewFromInt(int64(share))).
				Div(decimal.NewFromInt(int64(100 * vestsPerYear))).
				Floor()

			months := (year*vestsPerYear + i) * everyMonths
			vests = append(vests, Vest{Date: addMonths(first, months), Shares: due.Sub(vested)})
			vested = due
		}

		percent += percentOfYear
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
