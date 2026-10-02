package portfolio

import (
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// A Grant is an award of shares of one stock that vest over time, such as a grant of RSUs. What it
// is worth is potential until its vests are released, one by one, into lots.
type Grant struct {
	ID     int
	Name   string
	Symbol string

	// Vests are the days on which the shares of the grant vest, in order.
	Vests []Vest
}

// A Vest is one day on which shares of a grant vest. Until it is released into a lot, what it is
// worth counts as potential.
type Vest struct {
	ID int

	// Date is the calendar day the shares vest, at midnight UTC.
	Date   time.Time
	Shares decimal.Decimal
}

// intervals are the words a grant spec says how often a grant vests with, and how many months each
// of them is.
var intervals = map[string]int{
	"monthly":   1,
	"quarterly": 3,
	"yearly":    12,
}

// NewGrant parses a grant from its spec, which names it and describes its schedule:
//
//	<name>: <shares> <symbol> <interval> x<count> from <YYYY-MM-DD>
//
// The interval is monthly, quarterly or yearly. The grant vests that many shares count times, the
// first time on the day after "from": "Payout: 10 PANW monthly x24 from 2026-01-15" is 10 shares
// on the 15th of each of 24 months.
func NewGrant(spec string) (Grant, error) {
	name, schedule, _ := strings.Cut(spec, ":")
	fields := strings.Fields(schedule)

	shares, err := decimal.NewFromString(fields[0])
	if err != nil {
		return Grant{}, err
	}

	count, err := strconv.Atoi(strings.TrimPrefix(fields[3], "x"))
	if err != nil {
		return Grant{}, err
	}

	first, err := ParseDay(fields[5])
	if err != nil {
		return Grant{}, err
	}

	return Grant{
		Name:   strings.TrimSpace(name),
		Symbol: strings.ToUpper(fields[1]),
		Vests:  Repeating(first, intervals[strings.ToLower(fields[2])], count, shares),
	}, nil
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
