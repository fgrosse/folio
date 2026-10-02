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

// TestRepeating_EndOfMonth covers a schedule that starts on a day not every month has. Its vests
// fall on the last day of a shorter month rather than spilling into the next one, and go back to
// the day of the first vest wherever a month has it.
func TestRepeating_EndOfMonth(t *testing.T) {
	vests := Repeating(day("2027-12-31"), 1, 4, shares("10"))

	expected := []Vest{
		{Date: day("2027-12-31"), Shares: shares("10")},
		{Date: day("2028-01-31"), Shares: shares("10")},
		{Date: day("2028-02-29"), Shares: shares("10")},
		{Date: day("2028-03-31"), Shares: shares("10")},
	}
	assert.Equal(t, expected, vests)
}

// TestGraded covers the schedule of a grant that vests more with every year: so many percent of
// all its shares in each year, spread evenly over the vests of that year.
func TestGraded(t *testing.T) {
	vests := Graded(day("2026-02-20"), 3, shares("400"), []int{10, 20, 30, 40})

	expected := []Vest{
		{Date: day("2026-02-20"), Shares: shares("10")},
		{Date: day("2026-05-20"), Shares: shares("10")},
		{Date: day("2026-08-20"), Shares: shares("10")},
		{Date: day("2026-11-20"), Shares: shares("10")},
		{Date: day("2027-02-20"), Shares: shares("20")},
		{Date: day("2027-05-20"), Shares: shares("20")},
		{Date: day("2027-08-20"), Shares: shares("20")},
		{Date: day("2027-11-20"), Shares: shares("20")},
		{Date: day("2028-02-20"), Shares: shares("30")},
		{Date: day("2028-05-20"), Shares: shares("30")},
		{Date: day("2028-08-20"), Shares: shares("30")},
		{Date: day("2028-11-20"), Shares: shares("30")},
		{Date: day("2029-02-20"), Shares: shares("40")},
		{Date: day("2029-05-20"), Shares: shares("40")},
		{Date: day("2029-08-20"), Shares: shares("40")},
		{Date: day("2029-11-20"), Shares: shares("40")},
	}
	assert.Equal(t, expected, vests)
}
