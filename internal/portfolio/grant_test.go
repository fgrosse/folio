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
	cases := map[string]struct {
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

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			grant, err := NewGrant(c.spec)
			require.NoError(t, err)
			assert.Equal(t, c.expected, grant)
		})
	}
}

// TestNewGrant_Errors covers the specs NewGrant refuses, each with an error that says what is wrong
// with it: the dialog the spec was typed into shows it under its field.
func TestNewGrant_Errors(t *testing.T) {
	const syntax = `a grant is written as "<name>: <shares> <symbol> <interval> x<count> from <YYYY-MM-DD>"`

	cases := map[string]struct {
		spec  string
		error string
	}{
		"empty": {
			spec:  "",
			error: syntax,
		},
		"no name": {
			spec:  "10 PANW monthly x3 from 2026-01-15",
			error: syntax,
		},
		"an empty name": {
			spec:  " : 10 PANW monthly x3 from 2026-01-15",
			error: "grant has no name",
		},
		"a field missing": {
			spec:  "Payout: 10 PANW monthly from 2026-01-15",
			error: syntax,
		},
		"no from": {
			spec:  "Payout: 10 PANW monthly x3 on 2026-01-15",
			error: syntax,
		},
		"shares that are not a number": {
			spec:  "Payout: ten PANW monthly x3 from 2026-01-15",
			error: `"ten" is not a number of shares`,
		},
		"no shares": {
			spec:  "Payout: 0 PANW monthly x3 from 2026-01-15",
			error: "grant must have more than 0 shares",
		},
		"an unknown interval": {
			spec:  "Payout: 10 PANW weekly x3 from 2026-01-15",
			error: `"weekly" is not an interval: use monthly, quarterly or yearly`,
		},
		"a count that is not a number": {
			spec:  "Payout: 10 PANW monthly xmany from 2026-01-15",
			error: `"xmany" is not a number of vests such as x24`,
		},
		"a count of zero": {
			spec:  "Payout: 10 PANW monthly x0 from 2026-01-15",
			error: `"x0" is not a number of vests such as x24`,
		},
		"neither a count nor percentages": {
			spec:  "Payout: 10 PANW monthly 24 from 2026-01-15",
			error: "the percentages 24 add up to 24, not 100",
		},
		"percentages that are not numbers": {
			spec:  "RSU: 400 PANW quarterly 10/some/40 from 2026-02-20",
			error: `"10/some/40" is neither a count such as x24 nor percentages such as 10/20/30/40`,
		},
		"percentages that do not add up": {
			spec:  "RSU: 400 PANW quarterly 10/20/30 from 2026-02-20",
			error: "the percentages 10/20/30 add up to 60, not 100",
		},
		"a day that is not YYYY-MM-DD": {
			spec:  "Payout: 10 PANW monthly x3 from January",
			error: `"January" is not a day written as YYYY-MM-DD`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewGrant(c.spec)
			assert.EqualError(t, err, c.error)
		})
	}
}

// TestParseVests covers a schedule that no rule lays out, such as one with a first vest ten times
// the size of the others, or one whose vests the plan rounds its own way: it is written down vest
// by vest, a day and a number of shares to a line, the way the plan's own table lists them. Empty
// lines and comments are skipped, and the vests come back in the order of their days.
func TestParseVests(t *testing.T) {
	vests, err := ParseVests(`
		# the first vest makes up for the months before the grant
		2025-08-01 480
		2025-10-01 41

		2025-09-01 42   # as planned
	`)
	require.NoError(t, err)

	expected := []Vest{
		{Date: day("2025-08-01"), Shares: shares("480")},
		{Date: day("2025-09-01"), Shares: shares("42")},
		{Date: day("2025-10-01"), Shares: shares("41")},
	}
	assert.Equal(t, expected, vests)
}

// TestParseVests_Errors covers the lines ParseVests refuses, each named by its number, since a
// schedule of a few years is a long list to find a typo in.
func TestParseVests_Errors(t *testing.T) {
	cases := map[string]struct {
		text  string
		error string
	}{
		"a line without shares": {
			text:  "2025-08-01 480\n2025-09-01\n",
			error: `line 2: a vest is written as "<YYYY-MM-DD> <shares>"`,
		},
		"a line with too much": {
			text:  "2025-08-01 480 shares\n",
			error: `line 1: a vest is written as "<YYYY-MM-DD> <shares>"`,
		},
		"a day that is not YYYY-MM-DD": {
			text:  "\n\n01.08.2025 480\n",
			error: `line 3: "01.08.2025" is not a day written as YYYY-MM-DD`,
		},
		"shares that are not a number": {
			text:  "2025-08-01 many\n",
			error: `line 1: "many" is not a number of shares`,
		},
		"no shares": {
			text:  "2025-08-01 480\n# all good so far\n2025-09-01 0\n",
			error: "line 3: a vest must have more than 0 shares",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseVests(c.text)
			assert.EqualError(t, err, c.error)
		})
	}
}
