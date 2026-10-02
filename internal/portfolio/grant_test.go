package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRepeating covers the simplest vesting schedule, the one of a payout that releases the same
// number of shares at a steady interval: a vest of that many shares every so many months, starting
// with the day of the first.
func TestRepeating(t *testing.T) {
	vests := Repeating(day("2026-01-15"), 1, 4, shares("10"))

	expected := []Vest{
		{Date: day("2026-01-15"), Shares: shares("10")},
		{Date: day("2026-02-15"), Shares: shares("10")},
		{Date: day("2026-03-15"), Shares: shares("10")},
		{Date: day("2026-04-15"), Shares: shares("10")},
	}
	assert.Equal(t, expected, vests)
}
