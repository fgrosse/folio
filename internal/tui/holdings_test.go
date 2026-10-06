package tui

import (
	"errors"
	"testing"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestHoldingsModel_Init covers how the Holdings view starts: with two commands at once. One loads
// the portfolio as the store has it, which is on screen at once, and the other fetches fresh quotes
// and loads it again, which takes as long as the network does.
func TestHoldingsModel_Init(t *testing.T) {
	store := new(MockStore)
	store.returns(testPortfolio())
	store.On("SaveQuote", portfolio.Quote{Symbol: "PANW", Price: dec("401.5"), Currency: "USD"}).Return(nil)
	m := NewHoldingsModel(store, quotes{"PANW": "401.5"}, DefaultStyle())

	batch, ok := runCmd(t, m.Init()).(tea.BatchMsg)
	require.True(t, ok, "Init should batch its commands")
	require.Len(t, batch, 2)

	for _, cmd := range batch {
		msg, ok := runCmd(t, cmd).(PortfolioLoadedMsg)
		require.True(t, ok)
		assert.NoError(t, msg.err)
		assert.NoError(t, msg.quotesErr)
	}
	store.AssertNumberOfCalls(t, "SaveQuote", 1)
}

// TestHoldingsModel_LoadsPortfolio covers what the Holdings view shows once the portfolio is loaded:
// a row for every lot, valued at the quote the store has of its stock.
func TestHoldingsModel_LoadsPortfolio(t *testing.T) {
	p := testPortfolio()
	store := new(MockStore)
	store.returns(p)

	m := NewHoldingsModel(store, quotes{}, DefaultStyle())
	assert.Equal(t, "Holdings", m.Title())

	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, loadPortfolioCmd(store)))

	rows := m.table.Rows()
	require.Len(t, rows, 2)
	assert.Equal(t, table.Row{"2026-01-15", "PANW", "Payout", "         6", "   $380.12", "   $396.25", "   +4.2%", "     $2,377.50"}, rows[0])
	assert.Equal(t, table.Row{"2026-02-15", "PANW", "", "       2.5", "         -", "   $396.25", "       -", "       $990.63"}, rows[1])
	store.AssertExpectations(t)
}

// TestHoldingsModel_Keys covers the keys the view answers to before it can change anything: q and
// ctrl+c quit, and every other key is the table's, which moves the selection on the ones it knows.
func TestHoldingsModel_Keys(t *testing.T) {
	store := new(MockStore)
	store.returns(testPortfolio())
	m := NewHoldingsModel(store, quotes{}, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, loadPortfolioCmd(store)))

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
	m := NewHoldingsModel(store, quotes{}, DefaultStyle())

	// The window size matters as much as the portfolio: without it the table has no width, and a
	// golden taken without one would lock in an empty frame.
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, loadPortfolioCmd(store)))

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

	m := NewHoldingsModel(store, quotes{}, DefaultStyle())
	m.now = func() time.Time { return time.Date(2026, time.October, 2, 14, 30, 0, 0, time.UTC) }
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, loadPortfolioCmd(store)))

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
	m := NewHoldingsModel(store, quotes{}, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, loadPortfolioCmd(store)))

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

// TestHoldingsModel_ShowsErrors covers what the view does when the store or the prices let it down:
// it says what went wrong where the prices otherwise are, and keeps showing the account it had,
// since an account that goes blank reads as shares that are gone. The next load that works takes
// the error away again.
func TestHoldingsModel_ShowsErrors(t *testing.T) {
	m, _ := newTestingHoldings(t)

	m.Update(PortfolioLoadedMsg{err: errors.New("database is locked")})

	frame := ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "database is locked")
	assert.Contains(t, frame, "8.5 PANW", "the account should stay on display")
	assert.Len(t, m.table.Rows(), 2)

	m.Update(PortfolioLoadedMsg{portfolio: testPortfolio()})

	assert.NotContains(t, ansi.Strip(m.View().Content), "database is locked")
}

// TestHoldingsModel_ShowsQuoteErrors covers a portfolio that was loaded although its quotes could
// not be refreshed, such as without a network: the view shows the portfolio, at the prices the store
// still had, and says why they are not fresh where the prices otherwise are.
func TestHoldingsModel_ShowsQuoteErrors(t *testing.T) {
	m, _ := newTestingHoldings(t)
	p := testPortfolio()
	p.Lots = p.Lots[:1]

	m.Update(PortfolioLoadedMsg{portfolio: p, quotesErr: errors.New("no quote of PANW")})

	frame := ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "no quote of PANW")
	assert.Contains(t, frame, "6 PANW")
	assert.Len(t, m.table.Rows(), 1, "the portfolio that came with the error should be on display")
}

// TestHoldingsModel_KeepsQuotesFresh covers the prices of a TUI that is left open: every load of
// the portfolio schedules a refresh of the quotes an interval later, unless one is on its way
// already, since loads come from several places and each would start a timer of its own otherwise.
// When the refresh is due the view fetches the quotes, and the load that follows schedules the next.
func TestHoldingsModel_KeepsQuotesFresh(t *testing.T) {
	p := testPortfolio()
	store := new(MockStore)
	store.returns(p)
	store.On("SaveQuote", mock.Anything).Return(nil)

	m := NewHoldingsModel(store, quotes{"PANW": "401.5"}, DefaultStyle())
	m.refreshInterval = time.Millisecond // the real one would outlast the test

	_, cmd := m.Update(PortfolioLoadedMsg{portfolio: p})
	assert.Equal(t, RefreshQuotesMsg{}, runCmd(t, cmd))

	_, again := m.Update(PortfolioLoadedMsg{portfolio: p})
	assert.Nil(t, again, "a refresh is scheduled already")

	_, cmd = m.Update(RefreshQuotesMsg{})
	loaded := runCmd(t, cmd)
	require.IsType(t, PortfolioLoadedMsg{}, loaded)
	store.AssertNumberOfCalls(t, "SaveQuote", 1)

	_, cmd = m.Update(loaded)
	assert.Equal(t, RefreshQuotesMsg{}, runCmd(t, cmd), "the load after a refresh should schedule the next")
}

// TestHoldingsModel_EditLot covers correcting a lot in the view, such as one that was released
// without its cost: e opens the dialog with the spec of the selected lot in its field, and on enter
// the view saves what the spec now says under the ID of that lot, so that it replaces the lot rather
// than adding one, and loads the portfolio again.
func TestHoldingsModel_EditLot(t *testing.T) {
	m, store := newTestingHoldings(t)
	m.Update(keyPressed("j")) // the second lot, which has no cost

	m.Update(keyPressed("e"))
	require.True(t, m.CapturesKeys(), "the dialog should take the keyboard")
	assert.Equal(t, "2.5 PANW 2026-02-15", m.input.Value())
	assert.Contains(t, ansi.Strip(m.View().Content), "Edit lot")

	for _, key := range keysPressed(" @391.4") {
		m.Update(key)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	lot := portfolio.Lot{ID: 2, Symbol: "PANW", Shares: dec("2.5"), Acquired: day("2026-02-15"), Cost: dec("391.4")}
	msg := runCmd(t, cmd)
	require.Equal(t, SaveLotMsg{lot: lot}, msg)

	store.On("SaveLot", lot).Return(nil)
	_, cmd = m.Update(msg)
	assert.False(t, m.CapturesKeys(), "the dialog should be closed")
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)
}

// TestHoldingsModel_EditKeepsTheDay covers a spec whose day was deleted while editing: the lot
// keeps the day it has, rather than moving to today as a new lot without a day would. In an account
// without lots there is nothing to edit, and e opens no dialog.
func TestHoldingsModel_EditKeepsTheDay(t *testing.T) {
	m, _ := newTestingHoldings(t)
	m.Update(keyPressed("e"))
	m.input.SetValue("7 PANW @380.12")

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	msg, ok := runCmd(t, cmd).(SaveLotMsg)
	require.True(t, ok)
	assert.Equal(t, 1, msg.lot.ID)
	assert.Equal(t, day("2026-01-15"), msg.lot.Acquired)
	assert.Equal(t, "7", msg.lot.Shares.String())

	store := new(MockStore)
	store.returns(Portfolio{})
	empty := NewHoldingsModel(store, quotes{}, DefaultStyle())
	empty.Update(runCmd(t, loadPortfolioCmd(store)))

	_, cmd = empty.Update(keyPressed("e"))

	assert.Nil(t, cmd)
	assert.False(t, empty.CapturesKeys(), "there should be no dialog without a lot")
}

// TestHoldingsModel_ShowsWhatIsLeft covers lots that shares were sold from: a row shows the shares
// that remain of its lot and what those are worth, and a lot that was sold to the last share is not
// a holding any more and has no row. The positions above the table count what is left as well.
func TestHoldingsModel_ShowsWhatIsLeft(t *testing.T) {
	p := testPortfolio()
	p.Lots[0].Sold = dec("2")   // of 6
	p.Lots[1].Sold = dec("2.5") // all of it

	m, _ := newTestingHoldings(t)
	m.Update(PortfolioLoadedMsg{portfolio: p})

	rows := m.table.Rows()
	require.Len(t, rows, 1)
	assert.Equal(t, table.Row{"2026-01-15", "PANW", "Payout", "         4", "   $380.12", "   $396.25", "   +4.2%", "     $1,585.00"}, rows[0])
	assert.Contains(t, ansi.Strip(m.View().Content), "  4 PANW ")
}

// TestHoldingsModel_HidesColumnsInANarrowWindow covers a window that has no room for every column:
// the table then leaves out the gain, which the cost and the price next to it say already, rather
// than squeezing the grant away, and shows it again once the window is wide enough. The row that
// was selected stays selected through it.
func TestHoldingsModel_HidesColumnsInANarrowWindow(t *testing.T) {
	m, _ := newTestingHoldings(t)
	m.Update(keyPressed("j"))
	require.Contains(t, ansi.Strip(m.View().Content), "Gain")
	require.Len(t, m.table.Rows()[0], 8)

	m.Update(tea.WindowSizeMsg{Width: 82, Height: 20})

	frame := ansi.Strip(m.View().Content)
	assert.NotContains(t, frame, "Gain")
	assert.Contains(t, frame, "│ 2026-01-15  PANW      Pay…           6     $380.12     $396.25       $2,377.50 │")
	assert.Equal(t, 1, m.table.Cursor(), "the selection should stay where it was")

	m.Update(tea.WindowSizeMsg{Width: 92, Height: 20})

	frame = ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "Gain")
	assert.Contains(t, frame, "│ 2026-01-15  PANW      Pay…           6     $380.12     $396.25     +4.2%       $2,377.50 │")
	assert.Equal(t, 1, m.table.Cursor())
}

// TestHoldingsModel_HasNoColumnForTheTax covers where the tax on the gain of a lot is not: in the
// table, however wide the window and whatever the rate of the account. It is a detail of a lot,
// which its flyout has, and the room goes to the grant.
func TestHoldingsModel_HasNoColumnForTheTax(t *testing.T) {
	m, _ := newTestingHoldings(t)
	require.True(t, m.portfolio.GainsTaxRate.Valid)

	m.Update(tea.WindowSizeMsg{Width: 108, Height: 20})

	frame := ansi.Strip(m.View().Content)
	assert.NotContains(t, frame, "Tax")
	assert.Contains(t, frame, "│ 2026-01-15  PANW      Payout                         6     $380.12     $396.25     +4.2%       $2,377.50 │")
}

// TestHoldingsModel_SellShares covers recording a sale: s on a lot opens a form that asks how many
// of its shares were sold, at what price and on which day, which is today unless it is changed, and
// has room for notes of several lines. Confirming it has the store save the sale against that lot,
// and the view loads the portfolio again, in which the lot has that many shares fewer.
func TestHoldingsModel_SellShares(t *testing.T) {
	m, store := newTestingHoldings(t)

	m.Update(keyPressed("s"))
	require.True(t, m.CapturesKeys(), "the form should take the keyboard")
	assert.Equal(t, []string{"", "", "2026-10-02", ""}, m.form.Values())

	press := func(keys ...tea.Msg) {
		for _, key := range keys {
			m.Update(key)
		}
	}
	press(keysPressed("4")...)
	press(tabKey)
	press(keysPressed("410.2")...)
	press(tabKey, tabKey)
	press(keysPressed("for the kitchen")...)
	press(enterKey)
	press(keysPressed("in two orders")...)
	_, cmd := m.Update(saveKey)

	sale := portfolio.Sale{
		LotID:  1,
		Date:   day("2026-10-02"),
		Shares: dec("4"),
		Price:  dec("410.2"),
		Note:   "for the kitchen\nin two orders",
	}
	msg := runCmd(t, cmd)
	require.Equal(t, SaveSaleMsg{sale: sale}, msg)

	store.On("SaveSale", sale).Return(nil)
	_, cmd = m.Update(msg)
	assert.False(t, m.CapturesKeys(), "the form should be closed")
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)
}

// TestHoldingsModel_SaleFormRefuses covers what the sale form does not take: shares that are no
// number, none, or more than the lot has left, a price that is none, and a day that is no day or
// before the lot was acquired. The form stays open and says why under its fields.
func TestHoldingsModel_SaleFormRefuses(t *testing.T) {
	cases := map[string]struct {
		shares, price, date string
		error               string
	}{
		"shares not a number":         {shares: "some", price: "410.2", date: "2026-10-02", error: `"some" is not a number of shares`},
		"no shares":                   {shares: "0", price: "410.2", date: "2026-10-02", error: "a sale must have more than 0 shares"},
		"more shares than are left":   {shares: "6.5", price: "410.2", date: "2026-10-02", error: "the lot has 6 shares left, not 6.5"},
		"no price":                    {shares: "4", price: "", date: "2026-10-02", error: `"" is not a price such as 410.20`},
		"a price of nothing":          {shares: "4", price: "0", date: "2026-10-02", error: `"0" is not a price such as 410.20`},
		"a day that is no day":        {shares: "4", price: "410.2", date: "yesterday", error: `"yesterday" is not a day written as YYYY-MM-DD`},
		"before the lot was acquired": {shares: "4", price: "410.2", date: "2026-01-14", error: "the lot was only acquired on 2026-01-15"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m, _ := newTestingHoldings(t)

			msg, err := newSale(m.lots[0], day("2026-10-02"))([]string{c.shares, c.price, c.date, ""})

			assert.Nil(t, msg)
			assert.EqualError(t, err, c.error)
		})
	}

	// A price may have its dollar sign, and a form without a day is of today.
	m, _ := newTestingHoldings(t)
	msg, err := newSale(m.lots[0], day("2026-10-02"))([]string{"4", "$410.20", "", ""})
	require.NoError(t, err)
	sale := msg.(SaveSaleMsg).sale
	assert.Equal(t, "410.2", sale.Price.String())
	assert.Equal(t, day("2026-10-02"), sale.Date)
}

// TestHoldingsModel_CancelSale covers leaving the sale form with esc: it closes, the keys are the
// view's again, and nothing is saved. In an account without lots there is nothing to sell, and s
// opens no form.
func TestHoldingsModel_CancelSale(t *testing.T) {
	m, store := newTestingHoldings(t)
	m.Update(keyPressed("s"))

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_, cmd = m.Update(runCmd(t, cmd))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "the form should be closed")
	store.AssertNotCalled(t, "SaveSale", mock.Anything)

	emptyStore := new(MockStore)
	emptyStore.returns(Portfolio{})
	empty := NewHoldingsModel(emptyStore, quotes{}, DefaultStyle())
	empty.Update(runCmd(t, loadPortfolioCmd(emptyStore)))

	_, cmd = empty.Update(keyPressed("s"))
	assert.Nil(t, cmd)
	assert.False(t, empty.CapturesKeys(), "there should be no form without a lot")
}

// TestHoldingsModel_RenderSaleForm is the frame while a sale is being entered: the form in front of
// the table, its fields under each other after their labels, and the keys of the form in the help
// lines, where the one that saves depends on the field: enter, or ctrl+s in the notes, in which
// enter starts a new line. The frame is as tall as it is without the form.
func TestHoldingsModel_RenderSaleForm(t *testing.T) {
	m, _ := newTestingHoldings(t)
	without := m.View().Content

	m.Update(keyPressed("s"))
	for _, key := range keysPressed("4") {
		m.Update(key)
	}

	frame := ansi.Strip(m.View().Content)
	assert.Equal(t, lipgloss.Height(without), lipgloss.Height(frame), "the form should not change the height of the frame")
	assert.Contains(t, frame, "enter save • tab next field • esc cancel")
	golden.RequireEqual(t, frame)

	m.Update(shiftTabKey) // around to the notes
	assert.Contains(t, ansi.Strip(m.View().Content), "ctrl+s save • tab next field • esc cancel")
}

// TestPositions covers the line the Holdings view puts above its table: how many shares of each
// stock the lots add up to, which no single row says. The stocks are in alphabetical order, and an
// account without lots says so.
func TestPositions(t *testing.T) {
	cases := map[string]struct {
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

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, positions(c.lots))
		})
	}
}

// TestLotRow covers how one lot reads as a row of the Holdings table: the day it was acquired, its
// symbol and the grant it was released from, and then the numbers - how many shares, what one cost
// and is worth now, how far that is from the cost, with its sign, and last what all of them are worth - right-aligned so that
// their digits line up down the column. The table has no alignment of its own, so the values are padded
// out to the width of their column.
func TestLotRow(t *testing.T) {
	panw := portfolio.Quote{Symbol: "PANW", Price: dec("396.25")}

	cases := map[string]struct {
		lot      portfolio.Lot
		quote    portfolio.Quote
		expected table.Row
	}{
		"a lot released from a grant": {
			lot:      portfolio.Lot{Symbol: "PANW", Shares: dec("6"), Acquired: day("2026-01-15"), Cost: dec("380.12"), Grant: "Payout"},
			quote:    panw,
			expected: table.Row{"2026-01-15", "PANW", "Payout", "         6", "   $380.12", "   $396.25", "   +4.2%", "     $2,377.50"},
		},
		// A lot entered by hand came from no grant, and says nothing about where it is from. One
		// without a cost shows a dash for it, rather than shares that cost nothing, and for what
		// is measured from the cost.
		"a fraction of a share, entered by hand without a cost": {
			lot:      portfolio.Lot{Symbol: "PANW", Shares: dec("2.125"), Acquired: day("2026-02-15")},
			quote:    panw,
			expected: table.Row{"2026-02-15", "PANW", "", "     2.125", "         -", "   $396.25", "       -", "       $842.03"},
		},
		"a value in the millions": {
			lot:      portfolio.Lot{Symbol: "PANW", Shares: dec("12000"), Acquired: day("2020-06-01"), Cost: dec("75.5")},
			quote:    panw,
			expected: table.Row{"2020-06-01", "PANW", "", "     12000", "    $75.50", "   $396.25", " +424.8%", " $4,755,000.00"},
		},
		"a lot that lost": {
			lot:      portfolio.Lot{Symbol: "PANW", Shares: dec("3"), Acquired: day("2025-12-01"), Cost: dec("452.86")},
			quote:    panw,
			expected: table.Row{"2025-12-01", "PANW", "", "         3", "   $452.86", "   $396.25", "  -12.5%", "     $1,188.75"},
		},
		// Without a quote there is nothing to value the lot at, which reads as a dash rather than
		// as shares that are worth nothing.
		"a lot without a quote": {
			lot:      portfolio.Lot{Symbol: "SAP.DE", Shares: dec("5"), Acquired: day("2026-01-15"), Cost: dec("120")},
			quote:    portfolio.Quote{},
			expected: table.Row{"2026-01-15", "SAP.DE", "", "         5", "   $120.00", "         -", "       -", "             -"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, lotRow(c.lot, c.quote))
		})
	}
}

// TestHoldingsModel_ShowsDetails covers the flyout of the view: enter opens it on the selected lot,
// and it leaves the keyboard to the view, so that it shows one lot after the other as the selection
// moves through the table, and the keys that switch views still do.
func TestHoldingsModel_ShowsDetails(t *testing.T) {
	m, _ := newTestingHoldings(t)
	assert.NotContains(t, ansi.Strip(m.View().Content), "Cost per share")

	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	frame := ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "PANW of 2026-01-15")
	assert.Contains(t, frame, "Cost per share")
	assert.False(t, m.CapturesKeys(), "the flyout should leave the keys to the view")

	m.Update(keyPressed("j"))

	frame = ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "PANW of 2026-02-15")
	assert.NotContains(t, frame, "PANW of 2026-01-15")
}

// TestHoldingsModel_RenderDetails is the frame with the flyout open: it takes the right of the
// table's box, from the top of it to the bottom, and leaves the left of every row in sight, which
// says whose details these are. The window is a row taller than that of the other frames, which is
// what the flyout takes to show its last row, the tax on the gain.
func TestHoldingsModel_RenderDetails(t *testing.T) {
	m, _ := newTestingHoldings(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 21})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	golden.RequireEqual(t, ansi.Strip(m.View().Content))
}

// TestHoldingsModel_DetailsWithNothingSelected covers enter in an account without lots: there are no
// details to show, so nothing opens, and the help does not offer to close what is not there.
func TestHoldingsModel_DetailsWithNothingSelected(t *testing.T) {
	store := new(MockStore)
	store.returns(Portfolio{})
	m := NewHoldingsModel(store, quotes{}, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, loadPortfolioCmd(store)))

	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	frame := ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "enter details")
	assert.NotContains(t, frame, "esc close")
}

// TestHoldingsModel_ClosesDetails covers getting rid of the flyout: with the key that opened it, and
// with esc, which closes whatever else is in front of the table as well.
func TestHoldingsModel_ClosesDetails(t *testing.T) {
	keys := map[string]tea.KeyPressMsg{
		"enter": {Code: tea.KeyEnter},
		"esc":   {Code: tea.KeyEscape},
	}

	for name, key := range keys {
		t.Run(name, func(t *testing.T) {
			m, _ := newTestingHoldings(t)
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			require.Contains(t, ansi.Strip(m.View().Content), "Cost per share")

			_, cmd := m.Update(key)

			assert.Nil(t, cmd)
			assert.NotContains(t, ansi.Strip(m.View().Content), "Cost per share")
		})
	}
}

// TestHoldingsModel_DetailsHelp covers how the flyout is found: the help names the key that opens
// it, and while it is open the key that closes it in its place, since the other keys stay what they
// were.
func TestHoldingsModel_DetailsHelp(t *testing.T) {
	m, _ := newTestingHoldings(t)
	assert.Contains(t, ansi.Strip(m.View().Content), "a add • e edit • s sell • d delete • enter details\n")

	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.Contains(t, ansi.Strip(m.View().Content), "a add • e edit • s sell • d delete • esc close\n")
}

// TestLotDetails covers what the flyout says about a lot, which is what its row has no room for:
// the shares it was acquired with next to those sold and those left, and what the ones that are left
// have gained or lost since they were acquired, in dollars and in percent of what they cost. The
// percentage has a row of its own, without a label, so that the dollars stand under the value they
// are part of. What is not known reads as a dash, as it does in the row, and a gain is only known
// with both a cost and a price. The value is in the style of a value and the gain in that of a gain
// or a loss, and a dash is in none. In an account with a gains tax rate, the tax that selling the
// lot would cost comes last: nothing for a loss, and a dash where the gain is not known.
func TestLotDetails(t *testing.T) {
	style := DefaultStyle()
	panw := portfolio.Quote{Symbol: "PANW", Price: dec("396.25")}

	cases := map[string]struct {
		lot          portfolio.Lot
		quote        portfolio.Quote
		gainsTaxRate decimal.NullDecimal
		expected     Flyout
	}{
		"a lot released from a grant, partly sold": {
			lot: portfolio.Lot{
				Symbol: "PANW", Shares: dec("6"), Sold: dec("2"), Acquired: day("2026-01-15"),
				Cost: dec("380.12"), Grant: "Payout",
			},
			quote: panw,
			expected: Flyout{
				Title: "PANW of 2026-01-15",
				Sections: []FlyoutSection{
					{Title: "Shares", Rows: []FlyoutRow{
						{Label: "From", Value: "Payout"},
						{Label: "Acquired", Value: "6"},
						{Label: "Sold", Value: "2"},
						{Label: "Left", Value: "4"},
					}},
					{Title: "Value", Rows: []FlyoutRow{
						{Label: "Cost per share", Value: "$380.12"},
						{Label: "Price per share", Value: "$396.25"},
						{Label: "Value of what is left", Value: "$1,585.00", ValueStyle: style.Value},
						{Label: "Gain", Value: "$64.52", ValueStyle: style.Gain},
						{Value: "▲ 4.2%", ValueStyle: style.Gain},
					}},
				},
			},
		},
		"a lot that is worth less than it cost": {
			lot:   portfolio.Lot{Symbol: "PANW", Shares: dec("17"), Acquired: day("2025-10-01"), Cost: dec("691.5")},
			quote: panw,
			expected: Flyout{
				Title: "PANW of 2025-10-01",
				Sections: []FlyoutSection{
					{Title: "Shares", Rows: []FlyoutRow{
						{Label: "From", Value: "-"},
						{Label: "Acquired", Value: "17"},
						{Label: "Sold", Value: "0"},
						{Label: "Left", Value: "17"},
					}},
					{Title: "Value", Rows: []FlyoutRow{
						{Label: "Cost per share", Value: "$691.50"},
						{Label: "Price per share", Value: "$396.25"},
						{Label: "Value of what is left", Value: "$6,736.25", ValueStyle: style.Value},
						{Label: "Gain", Value: "-$5,019.25", ValueStyle: style.Loss},
						{Value: "▼ 42.7%", ValueStyle: style.Loss},
					}},
				},
			},
		},
		"a lot entered by hand, without a cost or a quote": {
			lot:   portfolio.Lot{Symbol: "SAP.DE", Shares: dec("2.5"), Acquired: day("2026-02-15")},
			quote: portfolio.Quote{},
			expected: Flyout{
				Title: "SAP.DE of 2026-02-15",
				Sections: []FlyoutSection{
					{Title: "Shares", Rows: []FlyoutRow{
						{Label: "From", Value: "-"},
						{Label: "Acquired", Value: "2.5"},
						{Label: "Sold", Value: "0"},
						{Label: "Left", Value: "2.5"},
					}},
					{Title: "Value", Rows: []FlyoutRow{
						{Label: "Cost per share", Value: "-"},
						{Label: "Price per share", Value: "-"},
						{Label: "Value of what is left", Value: "-"},
						{Label: "Gain", Value: "-"},
					}},
				},
			},
		},
		"a lot that gained, in an account with a gains tax rate": {
			lot: portfolio.Lot{
				Symbol: "PANW", Shares: dec("6"), Sold: dec("2"), Acquired: day("2026-01-15"),
				Cost: dec("380.12"), Grant: "Payout",
			},
			quote:        panw,
			gainsTaxRate: decimal.NewNullDecimal(dec("26.4")),
			expected: Flyout{
				Title: "PANW of 2026-01-15",
				Sections: []FlyoutSection{
					{Title: "Shares", Rows: []FlyoutRow{
						{Label: "From", Value: "Payout"},
						{Label: "Acquired", Value: "6"},
						{Label: "Sold", Value: "2"},
						{Label: "Left", Value: "4"},
					}},
					{Title: "Value", Rows: []FlyoutRow{
						{Label: "Cost per share", Value: "$380.12"},
						{Label: "Price per share", Value: "$396.25"},
						{Label: "Value of what is left", Value: "$1,585.00", ValueStyle: style.Value},
						{Label: "Gain", Value: "$64.52", ValueStyle: style.Gain},
						{Value: "▲ 4.2%", ValueStyle: style.Gain},
						{Label: "Tax on the gain", Value: "$17.03"},
					}},
				},
			},
		},
		"a lot that lost, in an account with a gains tax rate": {
			lot:          portfolio.Lot{Symbol: "PANW", Shares: dec("17"), Acquired: day("2025-10-01"), Cost: dec("691.5")},
			quote:        panw,
			gainsTaxRate: decimal.NewNullDecimal(dec("26.4")),
			expected: Flyout{
				Title: "PANW of 2025-10-01",
				Sections: []FlyoutSection{
					{Title: "Shares", Rows: []FlyoutRow{
						{Label: "From", Value: "-"},
						{Label: "Acquired", Value: "17"},
						{Label: "Sold", Value: "0"},
						{Label: "Left", Value: "17"},
					}},
					{Title: "Value", Rows: []FlyoutRow{
						{Label: "Cost per share", Value: "$691.50"},
						{Label: "Price per share", Value: "$396.25"},
						{Label: "Value of what is left", Value: "$6,736.25", ValueStyle: style.Value},
						{Label: "Gain", Value: "-$5,019.25", ValueStyle: style.Loss},
						{Value: "▼ 42.7%", ValueStyle: style.Loss},
						{Label: "Tax on the gain", Value: "$0.00"},
					}},
				},
			},
		},
		"a lot without a cost or a quote, in an account with a gains tax rate": {
			lot:          portfolio.Lot{Symbol: "SAP.DE", Shares: dec("2.5"), Acquired: day("2026-02-15")},
			quote:        portfolio.Quote{},
			gainsTaxRate: decimal.NewNullDecimal(dec("26.4")),
			expected: Flyout{
				Title: "SAP.DE of 2026-02-15",
				Sections: []FlyoutSection{
					{Title: "Shares", Rows: []FlyoutRow{
						{Label: "From", Value: "-"},
						{Label: "Acquired", Value: "2.5"},
						{Label: "Sold", Value: "0"},
						{Label: "Left", Value: "2.5"},
					}},
					{Title: "Value", Rows: []FlyoutRow{
						{Label: "Cost per share", Value: "-"},
						{Label: "Price per share", Value: "-"},
						{Label: "Value of what is left", Value: "-"},
						{Label: "Gain", Value: "-"},
						{Label: "Tax on the gain", Value: "-"},
					}},
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, lotDetails(c.lot, c.quote, c.gainsTaxRate, style))
		})
	}
}
