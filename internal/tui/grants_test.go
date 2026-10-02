package tui

import (
	"testing"

	"charm.land/bubbles/v2/table"
	"github.com/stretchr/testify/assert"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestGrantRow covers how one grant reads as a row of the Grants table: its name and stock, how many
// of its shares are still to come and how many it had in all, and what the ones to come are worth,
// which is the grant's part of the potential value. The numbers are right-aligned like those of the
// other tables.
func TestGrantRow(t *testing.T) {
	panw := portfolio.Quote{Symbol: "PANW", Price: dec("396.25")}

	tests := map[string]struct {
		grant    portfolio.Grant
		quote    portfolio.Quote
		expected table.Row
	}{
		"a grant with a released vest": {
			grant: portfolio.Grant{
				Name:   "Payout",
				Symbol: "PANW",
				Vests: []portfolio.Vest{
					{Shares: dec("10"), Released: true},
					{Shares: dec("10")},
					{Shares: dec("10")},
				},
			},
			quote:    panw,
			expected: table.Row{"Payout", "PANW", "        20", "        30", "     $7,925.00"},
		},
		"a grant that is all released": {
			grant: portfolio.Grant{
				Name:   "Sign-on",
				Symbol: "PANW",
				Vests:  []portfolio.Vest{{Shares: dec("2.5"), Released: true}},
			},
			quote:    panw,
			expected: table.Row{"Sign-on", "PANW", "         0", "       2.5", "         $0.00"},
		},
		"a grant without a quote": {
			grant: portfolio.Grant{
				Name:   "Old plan",
				Symbol: "SAP.DE",
				Vests:  []portfolio.Vest{{Shares: dec("5")}},
			},
			quote:    portfolio.Quote{},
			expected: table.Row{"Old plan", "SAP.DE", "         5", "         5", "             -"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, grantRow(tt.grant, tt.quote))
		})
	}
}
