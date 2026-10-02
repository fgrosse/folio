package portfolio

import (
	"errors"
	"fmt"
	"slices"
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

	// Released says that the vest has been turned into a lot, which is what counts from then on.
	Released bool
}

// grantSyntax is how a grant is written, for the errors that say so. It shows the simpler of the
// two forms, which is enough to see what is missing.
const grantSyntax = `a grant is written as "<name>: <shares> <symbol> <interval> x<count> from <YYYY-MM-DD>"`

// intervals are the words a grant spec says how often a grant vests with, and how many months each
// of them is.
var intervals = map[string]int{
	"monthly":   1,
	"quarterly": 3,
	"yearly":    12,
}

// NewGrant parses a grant from its spec, which names it and describes its schedule in one of two
// forms:
//
//	<name>: <shares> <symbol> <interval> x<count> from <YYYY-MM-DD>
//	<name>: <shares> <symbol> <interval> <percent>/<percent>/... from <YYYY-MM-DD>
//
// The interval is monthly, quarterly or yearly, and the day after "from" is that of the first vest.
//
// With a count, the grant vests that many shares count times: "Payout: 10 PANW monthly x24 from
// 2026-01-15" is 10 shares on the 15th of each of 24 months. See Repeating.
//
// With percentages, the shares are those of the whole grant, and each percentage is the part of
// them that vests in one year, spread evenly over the vests of that year: "RSU: 400 PANW quarterly
// 10/20/30/40 from 2026-02-20" vests 40 shares over the four quarters of the first year and 160
// over those of the fourth. See Graded.
func NewGrant(spec string) (Grant, error) {
	name, schedule, found := strings.Cut(spec, ":")
	fields := strings.Fields(schedule)
	if !found || len(fields) != 6 || fields[4] != "from" {
		return Grant{}, errors.New(grantSyntax)
	}

	grant := Grant{
		Name:   strings.TrimSpace(name),
		Symbol: strings.ToUpper(fields[1]),
	}
	if grant.Name == "" {
		return Grant{}, errors.New("grant has no name")
	}

	shares, err := decimal.NewFromString(fields[0])
	if err != nil {
		return Grant{}, fmt.Errorf("%q is not a number of shares", fields[0])
	}
	if !shares.IsPositive() {
		return Grant{}, errors.New("grant must have more than 0 shares")
	}

	months, ok := intervals[strings.ToLower(fields[2])]
	if !ok {
		return Grant{}, fmt.Errorf("%q is not an interval: use monthly, quarterly or yearly", fields[2])
	}

	first, err := ParseDay(fields[5])
	if err != nil {
		return Grant{}, err
	}

	grant.Vests, err = newSchedule(fields[3], first, months, shares)
	if err != nil {
		return Grant{}, err
	}

	return grant, nil
}

// NewListedGrant returns the grant that spec names, "<name>: <symbol>", with vests as its schedule,
// for a grant whose vests no rule lays out and which are listed one by one instead. See ParseVests.
func NewListedGrant(spec string, vests []Vest) (Grant, error) {
	name, symbol, found := strings.Cut(spec, ":")
	if !found || len(strings.Fields(symbol)) != 1 {
		return Grant{}, errors.New(`a grant with listed vests is written as "<name>: <symbol>"`)
	}

	grant := Grant{
		Name:   strings.TrimSpace(name),
		Symbol: strings.ToUpper(strings.TrimSpace(symbol)),
		Vests:  vests,
	}
	switch {
	case grant.Name == "":
		return Grant{}, errors.New("grant has no name")
	case len(vests) == 0:
		return Grant{}, errors.New("grant has no vests")
	}

	return grant, nil
}

// newSchedule lays out the vests of a grant spec from the field that says how many there are: a
// count such as "x24" of vests of that many shares each, or the percentages of the shares that vest
// in each year, such as "10/20/30/40".
func newSchedule(field string, first time.Time, months int, shares decimal.Decimal) ([]Vest, error) {
	if count, ok := strings.CutPrefix(field, "x"); ok {
		n, err := strconv.Atoi(count)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("%q is not a number of vests such as x24", field)
		}

		return Repeating(first, months, n, shares), nil
	}

	var (
		percentPerYear []int
		sum            int
	)
	for percent := range strings.SplitSeq(field, "/") {
		n, err := strconv.Atoi(percent)
		if err != nil {
			return nil, fmt.Errorf("%q is neither a count such as x24 nor percentages such as 10/20/30/40", field)
		}

		percentPerYear = append(percentPerYear, n)
		sum += n
	}

	if sum != 100 {
		return nil, fmt.Errorf("the percentages %s add up to %d, not 100", field, sum)
	}

	return Graded(first, months, shares, percentPerYear), nil
}

// ParseVests parses a schedule that is written down vest by vest, for a grant that no rule lays out:
// one vest to a line, as "<YYYY-MM-DD> <shares>". Empty lines are skipped, and so is whatever
// follows a "#". The vests come back in the order of their days. A line that is not a vest is
// refused by its number.
func ParseVests(text string) ([]Vest, error) {
	var (
		vests  []Vest
		number int
	)
	for line := range strings.Lines(text) {
		number++

		line, _, _ = strings.Cut(line, "#")
		if strings.TrimSpace(line) == "" {
			continue
		}

		vest, err := parseVest(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", number, err)
		}

		vests = append(vests, vest)
	}

	slices.SortStableFunc(vests, func(a, b Vest) int { return a.Date.Compare(b.Date) })

	return vests, nil
}

// parseVest parses one line of a schedule, "<YYYY-MM-DD> <shares>".
func parseVest(line string) (Vest, error) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return Vest{}, errors.New(`a vest is written as "<YYYY-MM-DD> <shares>"`)
	}

	date, err := ParseDay(fields[0])
	if err != nil {
		return Vest{}, err
	}

	shares, err := decimal.NewFromString(fields[1])
	if err != nil {
		return Vest{}, fmt.Errorf("%q is not a number of shares", fields[1])
	}
	if !shares.IsPositive() {
		return Vest{}, errors.New("a vest must have more than 0 shares")
	}

	return Vest{Date: date, Shares: shares}, nil
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
