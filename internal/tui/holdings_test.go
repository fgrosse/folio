package tui

import (
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestHoldingsModel_LoadsPortfolio covers what the Holdings view starts from: once started, it asks
// the store for the account and shows a row for every lot, valued at the quote the store has of its
// stock.
func TestHoldingsModel_LoadsPortfolio(t *testing.T) {
	p := testPortfolio()
	store := new(MockStore)
	store.returns(p)

	m := NewHoldingsModel(store, DefaultStyle())
	assert.Equal(t, "Holdings", m.Title())

	m.Update(runCmd(t, m.Init()))

	rows := m.table.Rows()
	require.Len(t, rows, 2)
	assert.Equal(t, lotRow(p.Lots[0], p.Quotes["PANW"]), rows[0])
	assert.Equal(t, lotRow(p.Lots[1], p.Quotes["PANW"]), rows[1])
	store.AssertExpectations(t)
}

// TestHoldingsModel_Keys covers the keys the view answers to before it can change anything: q and
// ctrl+c quit, and every other key is the table's, which moves the selection on the ones it knows.
func TestHoldingsModel_Keys(t *testing.T) {
	store := new(MockStore)
	store.returns(testPortfolio())
	m := NewHoldingsModel(store, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))

	_, cmd := m.Update(keyPressed("q"))
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))

	require.Equal(t, 0, m.table.Cursor())
	_, cmd = m.Update(keyPressed("j"))
	assert.Nil(t, cmd)
	assert.Equal(t, 1, m.table.Cursor(), "j should move the selection down a row")
}

// TestPositions covers the line the Holdings view puts above its table: how many shares of each
// stock the lots add up to, which no single row says. The stocks are in alphabetical order, and an
// account without lots says so.
func TestPositions(t *testing.T) {
	tests := map[string]struct {
		lots     []portfolio.Lot
		expected string
	}{
		"no lots": {
			lots:     nil,
			expected: "No shares held",
		},
		"one stock in two lots": {
			lots: []portfolio.Lot{
				{Symbol: "PANW", Shares: dec("6")},
				{Symbol: "PANW", Shares: dec("2.5")},
			},
			expected: "8.5 PANW",
		},
		"several stocks": {
			lots: []portfolio.Lot{
				{Symbol: "PANW", Shares: dec("6")},
				{Symbol: "AAPL", Shares: dec("3")},
				{Symbol: "PANW", Shares: dec("4")},
			},
			expected: "3 AAPL · 10 PANW",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, positions(tt.lots))
		})
	}
}

// TestLotRow covers how one lot reads as a row of the Holdings table: the day it was acquired, its
// symbol, and then the numbers - how many shares, what one is worth and what all of them are -
// right-aligned so that their digits line up down the column. The table has no alignment of its
// own, so the values are padded out to the width of their column.
func TestLotRow(t *testing.T) {
	panw := portfolio.Quote{Symbol: "PANW", Price: dec("396.25")}

	tests := map[string]struct {
		lot      portfolio.Lot
		quote    portfolio.Quote
		expected table.Row
	}{
		"a lot": {
			lot:      portfolio.Lot{Symbol: "PANW", Shares: dec("6"), Acquired: day("2026-01-15")},
			quote:    panw,
			expected: table.Row{"2026-01-15", "PANW", "         6", "   $396.25", "     $2,377.50"},
		},
		"a fraction of a share": {
			lot:      portfolio.Lot{Symbol: "PANW", Shares: dec("2.125"), Acquired: day("2026-02-15")},
			quote:    panw,
			expected: table.Row{"2026-02-15", "PANW", "     2.125", "   $396.25", "       $842.03"},
		},
		"a value in the millions": {
			lot:      portfolio.Lot{Symbol: "PANW", Shares: dec("12000"), Acquired: day("2020-06-01")},
			quote:    panw,
			expected: table.Row{"2020-06-01", "PANW", "     12000", "   $396.25", " $4,755,000.00"},
		},
		// Without a quote there is nothing to value the lot at, which reads as a dash rather than
		// as shares that are worth nothing.
		"a lot without a quote": {
			lot:      portfolio.Lot{Symbol: "SAP.DE", Shares: dec("5"), Acquired: day("2026-01-15")},
			quote:    portfolio.Quote{},
			expected: table.Row{"2026-01-15", "SAP.DE", "         5", "         -", "             -"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, lotRow(tt.lot, tt.quote))
		})
	}
}
