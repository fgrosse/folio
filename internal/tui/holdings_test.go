package tui

import (
	"testing"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// TestHoldingsModel_Render is the frame the Holdings view puts on screen, which has the shape of
// every view's: the positions and the account values on two header lines, a row for every lot in a
// boxed table, and the keys underneath.
func TestHoldingsModel_Render(t *testing.T) {
	store := new(MockStore)
	store.returns(testPortfolio())
	m := NewHoldingsModel(store, DefaultStyle())

	// The window size matters as much as the portfolio: without it the table has no width, and a
	// golden taken without one would lock in an empty frame.
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))

	// Stripped of styling: the frame is mostly escape sequences otherwise, and this golden is here
	// for the layout.
	golden.RequireEqual(t, ansi.Strip(m.View().Content))
}

// newTestingHoldings returns a Holdings view that has loaded the test portfolio into a window of
// 100 by 20, and the store it loaded it from, for a test to set up what comes next.
func newTestingHoldings(t *testing.T) (*HoldingsModel, *MockStore) {
	t.Helper()

	store := new(MockStore)
	store.returns(testPortfolio())

	m := NewHoldingsModel(store, DefaultStyle())
	m.now = func() time.Time { return time.Date(2026, time.October, 2, 14, 30, 0, 0, time.UTC) }
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))

	return m, store
}

// TestHoldingsModel_AddLot covers recording shares in the view: a opens a dialog that takes the
// keyboard, the spec typed into it becomes a lot on enter, dated today since the spec has no day,
// and the view saves that lot and loads the portfolio again to show it.
func TestHoldingsModel_AddLot(t *testing.T) {
	m, store := newTestingHoldings(t)
	assert.False(t, m.CapturesKeys())

	m.Update(keyPressed("a"))
	assert.True(t, m.CapturesKeys(), "the dialog should take the keyboard")

	for _, key := range keysPressed("12 panw") {
		m.Update(key)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	lot := portfolio.Lot{Symbol: "PANW", Shares: dec("12"), Acquired: day("2026-10-02")}
	msg := runCmd(t, cmd)
	require.Equal(t, SaveLotMsg{lot: lot}, msg)

	store.On("SaveLot", lot).Return(nil)
	_, cmd = m.Update(msg)
	assert.False(t, m.CapturesKeys(), "the dialog should be closed")
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)
}

// TestHoldingsModel_CancelAddLot covers leaving the dialog without a lot: esc closes it, the keys
// are the view's again, and nothing is saved.
func TestHoldingsModel_CancelAddLot(t *testing.T) {
	m, store := newTestingHoldings(t)
	m.Update(keyPressed("a"))
	for _, key := range keysPressed("12 PANW") {
		m.Update(key)
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_, cmd = m.Update(runCmd(t, cmd))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "the dialog should be closed")
	store.AssertNotCalled(t, "SaveLot", mock.Anything)
}

// TestHoldingsModel_RenderLotDialog is the frame while a lot is being typed: the dialog floats
// centered in front of the table, and the help lines show the keys of the dialog, since none of the
// others reach the view in that state. The frame is as tall as it is without the dialog.
func TestHoldingsModel_RenderLotDialog(t *testing.T) {
	m, _ := newTestingHoldings(t)
	without := m.View().Content

	m.Update(keyPressed("a"))
	for _, key := range keysPressed("12 PANW") {
		m.Update(key)
	}

	frame := ansi.Strip(m.View().Content)
	assert.Equal(t, lipgloss.Height(without), lipgloss.Height(frame), "the dialog should not change the height of the frame")
	golden.RequireEqual(t, frame)
}

// TestHoldingsModel_DeleteLot covers taking a lot back: d asks before it deletes the selected lot,
// naming it, and only a y has the store delete it, after which the view loads the portfolio again.
func TestHoldingsModel_DeleteLot(t *testing.T) {
	m, store := newTestingHoldings(t)
	m.Update(keyPressed("j")) // select the second lot

	m.Update(keyPressed("d"))
	assert.True(t, m.CapturesKeys(), "the question should take the keyboard")
	assert.Contains(t, ansi.Strip(m.View().Content), "Delete the 2.5 PANW of 2026-02-15?")

	_, cmd := m.Update(keyPressed("y"))
	msg := runCmd(t, cmd)
	require.Equal(t, DeleteLotMsg{id: 2}, msg)

	store.On("DeleteLot", 2).Return(nil)
	_, cmd = m.Update(msg)
	assert.False(t, m.CapturesKeys(), "the question should be closed")
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)
}

// TestHoldingsModel_CancelDelete covers a no to the question: the dialog closes, the lot stays, and
// the keys are the view's again.
func TestHoldingsModel_CancelDelete(t *testing.T) {
	m, store := newTestingHoldings(t)
	m.Update(keyPressed("d"))

	_, cmd := m.Update(keyPressed("n"))
	_, cmd = m.Update(runCmd(t, cmd))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "the question should be closed")
	store.AssertNotCalled(t, "DeleteLot", mock.Anything)
}

// TestHoldingsModel_DeleteWithNothingSelected covers d in an account without lots, where there is
// nothing to ask about.
func TestHoldingsModel_DeleteWithNothingSelected(t *testing.T) {
	store := new(MockStore)
	store.returns(Portfolio{})
	m := NewHoldingsModel(store, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))

	_, cmd := m.Update(keyPressed("d"))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "there should be no question to answer")
}

// TestHoldingsModel_RenderDeleteDialog is the frame while the view asks whether to delete a lot: the
// question in front of the table, and the keys that answer it in the help lines.
func TestHoldingsModel_RenderDeleteDialog(t *testing.T) {
	m, _ := newTestingHoldings(t)
	m.Update(keyPressed("d"))

	golden.RequireEqual(t, ansi.Strip(m.View().Content))
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
