package tui

import (
	"testing"

	"charm.land/bubbles/v2/table"
	"github.com/stretchr/testify/assert"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestUnreleasedVests covers which vests the Vesting view lists: those of every grant that have not
// been released, which are the ones the potential value counts, in the order of their days whatever
// grant they belong to. Vests of the same day stay in the order of their grants.
func TestUnreleasedVests(t *testing.T) {
	grants := []portfolio.Grant{
		{
			Name:   "RSU 2025",
			Symbol: "PANW",
			Vests: []portfolio.Vest{
				{ID: 1, Date: day("2026-08-20"), Shares: dec("10"), Released: true},
				{ID: 2, Date: day("2026-11-20"), Shares: dec("10")},
				{ID: 3, Date: day("2027-02-20"), Shares: dec("20")},
			},
		},
		{
			Name:   "Payout",
			Symbol: "AAPL",
			Vests: []portfolio.Vest{
				{ID: 4, Date: day("2026-09-15"), Shares: dec("5"), Released: true},
				{ID: 5, Date: day("2026-10-15"), Shares: dec("5")},
				{ID: 6, Date: day("2026-11-20"), Shares: dec("5")},
			},
		},
	}

	expected := []grantVest{
		{grant: "Payout", symbol: "AAPL", vest: grants[1].Vests[1]},
		{grant: "RSU 2025", symbol: "PANW", vest: grants[0].Vests[1]},
		{grant: "Payout", symbol: "AAPL", vest: grants[1].Vests[2]},
		{grant: "RSU 2025", symbol: "PANW", vest: grants[0].Vests[2]},
	}
	assert.Equal(t, expected, unreleasedVests(grants))
}

// TestVestRow covers how one vest reads as a row of the Vesting table: its day and the grant it
// belongs to, how many shares vest and what they are worth at the latest price, right-aligned like
// the numbers of the Holdings table, and how far off the day is.
func TestVestRow(t *testing.T) {
	today := day("2026-10-02")
	panw := portfolio.Quote{Symbol: "PANW", Price: dec("396.25")}

	tests := map[string]struct {
		vest     grantVest
		quote    portfolio.Quote
		expected table.Row
	}{
		"a vest to come": {
			vest:     grantVest{grant: "Payout", symbol: "PANW", vest: portfolio.Vest{Date: day("2026-11-15"), Shares: dec("10")}},
			quote:    panw,
			expected: table.Row{"2026-11-15", "Payout", "        10", "     $3,962.50", "in 44 days"},
		},
		"a vest pending release": {
			vest:     grantVest{grant: "RSU 2025", symbol: "PANW", vest: portfolio.Vest{Date: day("2026-08-20"), Shares: dec("2.5")}},
			quote:    panw,
			expected: table.Row{"2026-08-20", "RSU 2025", "       2.5", "       $990.63", "pending"},
		},
		"a vest without a quote": {
			vest:     grantVest{grant: "Old plan", symbol: "SAP.DE", vest: portfolio.Vest{Date: day("2027-01-01"), Shares: dec("5")}},
			quote:    portfolio.Quote{},
			expected: table.Row{"2027-01-01", "Old plan", "         5", "             -", "in 2 months"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, vestRow(tt.vest, tt.quote, today))
		})
	}
}

// TestDueIn covers how far off a vest is, in the words of the last column of the Vesting table. The
// nearer the day, the finer the unit: days up to two months out, then whole months, and whole years
// from two years on, since nobody plans by the day for a vest that is years away. A vest whose day
// has come and which has not been released is pending, as the bank calls it.
func TestDueIn(t *testing.T) {
	today := day("2026-10-02")

	tests := map[string]struct {
		date     string
		expected string
	}{
		"today":                    {date: "2026-10-02", expected: "pending"},
		"in the past":              {date: "2026-08-15", expected: "pending"},
		"tomorrow":                 {date: "2026-10-03", expected: "in 1 day"},
		"in some days":             {date: "2026-11-15", expected: "in 44 days"},
		"the last to read in days": {date: "2026-11-30", expected: "in 59 days"},
		"two months":               {date: "2026-12-02", expected: "in 2 months"},
		"months are rounded down":  {date: "2027-02-20", expected: "in 4 months"},
		"almost two years":         {date: "2028-10-01", expected: "in 23 months"},
		"two years":                {date: "2028-10-02", expected: "in 2 years"},
		"years are rounded down":   {date: "2030-08-20", expected: "in 3 years"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, dueIn(day(tt.date), today))
		})
	}
}
