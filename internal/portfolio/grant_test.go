package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestGraded_WholeShares covers a grant whose shares do not divide evenly: only whole shares vest,
// so each vest rounds down what has vested by then in all, and a fraction that was left over is
// made up for in a later vest. That way the vests add up to the grant, to the last share.
func TestGraded_WholeShares(t *testing.T) {
	vests := Graded(day("2026-02-20"), 3, shares("100"), []int{10, 20, 30, 40})

	// The first year vests 2.5 shares a quarter, the third 7.5.
	expected := []string{
		"2", "3", "2", "3",
		"5", "5", "5", "5",
		"7", "8", "7", "8",
		"10", "10", "10", "10",
	}

	var actual []string
	total := shares("0")
	for _, vest := range vests {
		actual = append(actual, vest.Shares.String())
		total = total.Add(vest.Shares)
	}
	assert.Equal(t, expected, actual)
	assert.Equal(t, "100", total.String())
}

// TestNewGrant covers the syntax a grant is typed in, which names it and describes its schedule:
// "<name>: <shares> <symbol> <interval> x<count> from <YYYY-MM-DD>" for the same number of shares
// every month, quarter or year, that many times, and with "<percent>/<percent>/..." in place of the
// count for a grant that vests a different share of its shares in each year.
func TestNewGrant(t *testing.T) {
	tests := map[string]struct {
		spec     string
		expected Grant
	}{
		"the same shares every month": {
			spec: "Acquisition payout: 10 PANW monthly x3 from 2026-01-15",
			expected: Grant{
				Name:   "Acquisition payout",
				Symbol: "PANW",
				Vests:  Repeating(day("2026-01-15"), 1, 3, shares("10")),
			},
		},
		"every quarter, in lower case": {
			spec: "Bonus: 2.5 panw quarterly x2 from 2026-03-01",
			expected: Grant{
				Name:   "Bonus",
				Symbol: "PANW",
				Vests:  Repeating(day("2026-03-01"), 3, 2, shares("2.5")),
			},
		},
		"every year": {
			spec: "Retention: 100 PANW yearly x4 from 2027-01-01",
			expected: Grant{
				Name:   "Retention",
				Symbol: "PANW",
				Vests:  Repeating(day("2027-01-01"), 12, 4, shares("100")),
			},
		},
		// The other form gives the shares of the whole grant and the percentage of them that vests
		// in each year, which is spread over the vests of that year.
		"a percentage of all shares each year": {
			spec: "RSU 2025: 400 PANW quarterly 10/20/30/40 from 2026-02-20",
			expected: Grant{
				Name:   "RSU 2025",
				Symbol: "PANW",
				Vests:  Graded(day("2026-02-20"), 3, shares("400"), []int{10, 20, 30, 40}),
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			grant, err := NewGrant(tt.spec)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, grant)
		})
	}
}
