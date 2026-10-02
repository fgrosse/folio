package tui

import (
	"fmt"
	"time"
)

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
