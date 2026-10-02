package tui

import (
	"fmt"
	"slices"
	"time"

	"github.com/fgrosse/folio/internal/portfolio"
)

// A grantVest is a vest together with what the Vesting view needs to know of the grant it belongs
// to: the name to list it under, and the symbol of the stock to value it at.
type grantVest struct {
	grant  string
	symbol string
	vest   portfolio.Vest
}

// unreleasedVests returns the vests of grants that have not been released, which are the ones the
// potential value counts, in the order of their days. Vests of the same day stay in the order of
// their grants.
func unreleasedVests(grants []portfolio.Grant) []grantVest {
	var vests []grantVest
	for _, grant := range grants {
		for _, vest := range grant.Vests {
			if !vest.Released {
				vests = append(vests, grantVest{grant: grant.Name, symbol: grant.Symbol, vest: vest})
			}
		}
	}

	slices.SortStableFunc(vests, func(a, b grantVest) int {
		return a.vest.Date.Compare(b.vest.Date)
	})

	return vests
}

// dueIn says how far off the day of a vest is from today, such as "in 44 days". The nearer the day,
// the finer the unit: days for less than sixty of them, then whole months, and whole years from two
// years on. A day that has come reads "pending": the vest is due and has not been released.
func dueIn(date, today time.Time) string {
	if !date.After(today) {
		return "pending"
	}

	days := int(date.Sub(today).Hours() / 24)
	if days < 60 {
		return plural(days, "day")
	}

	months := (date.Year()-today.Year())*12 + int(date.Month()-today.Month())
	if date.Day() < today.Day() {
		months-- // the last of them is not over yet
	}

	if months < 24 {
		return plural(months, "month")
	}

	return plural(months/12, "year")
}

// plural renders "in n units", with the unit in the singular for one of them.
func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("in 1 %s", unit)
	}

	return fmt.Sprintf("in %d %ss", n, unit)
}
