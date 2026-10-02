package tui

import (
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// newTestingGrants returns a Grants view that has loaded the test portfolio into a window of 100 by
// 20, and the store it loaded it from.
func newTestingGrants(t *testing.T) (*GrantsModel, *MockStore) {
	t.Helper()

	store := new(MockStore)
	store.returns(testPortfolio())

	m := NewGrantsModel(store, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))

	return m, store
}

// TestGrantsModel_LoadsPortfolio covers what the Grants view starts from: once started, it loads the
// portfolio and shows a row for every grant, valued at the quote the store has of its stock.
func TestGrantsModel_LoadsPortfolio(t *testing.T) {
	m, store := newTestingGrants(t)
	assert.Equal(t, "Grants", m.Title())

	p := testPortfolio()
	rows := m.table.Rows()
	require.Len(t, rows, 1)
	assert.Equal(t, grantRow(p.Grants[0], p.Quotes["PANW"]), rows[0])
	store.AssertExpectations(t)
}

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
