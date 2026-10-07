package tui

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// soldPortfolio is the test portfolio after two sales from its first lot, the second of them with a
// note of two lines.
func soldPortfolio() Portfolio {
	p := testPortfolio()
	p.Lots[0].Sold = dec("3")
	p.Sales = []portfolio.Sale{
		{
			ID: 1, LotID: 1, Date: day("2026-03-02"), Shares: dec("2"), Price: dec("401.5"),
			Symbol: "PANW", Cost: dec("380.12"), Grant: "Payout",
		},
		{
			ID: 2, LotID: 1, Date: day("2026-09-15"), Shares: dec("1"), Price: dec("410.2"),
			Note:   "for the kitchen\nsold in the morning",
			Symbol: "PANW", Cost: dec("380.12"), Grant: "Payout",
		},
	}

	return p
}

// newTestingSales returns a Sales view that has loaded the sold portfolio into a window of 100 by
// 20, and the store it loaded it from.
func newTestingSales(t *testing.T) (*SalesModel, *MockStore) {
	t.Helper()

	store := new(MockStore)
	store.returns(soldPortfolio())

	m := NewSalesModel(store, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))

	return m, store
}

// TestSalesModel_LoadsPortfolio covers what the Sales view starts from: once started, it loads the
// portfolio and shows a row for every sale, in the order of their days.
func TestSalesModel_LoadsPortfolio(t *testing.T) {
	m, store := newTestingSales(t)
	assert.Equal(t, "Sales", m.Title())

	p := soldPortfolio()
	rows := m.table.Rows()
	require.Len(t, rows, 2)
	assert.Equal(t, table.Row(shownSale(m, saleRow(p.Sales[0], p.GainsTaxRate))), rows[0])
	assert.Equal(t, table.Row(shownSale(m, saleRow(p.Sales[1], p.GainsTaxRate))), rows[1])
	store.AssertExpectations(t)
}

// TestSalesModel_Keys covers the keys the view answers to before it can change anything: q and
// ctrl+c quit, and every other key is the table's.
func TestSalesModel_Keys(t *testing.T) {
	m, _ := newTestingSales(t)

	_, cmd := m.Update(keyPressed("q"))
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))

	require.Equal(t, 0, m.table.Cursor())
	m.Update(keyPressed("j"))
	assert.Equal(t, 1, m.table.Cursor(), "j should move the selection down a row")
}

// TestSalesModel_Columns covers which columns the Sales table has at which width. What a sale
// brought in, the tax on it and its gain are in every window. The price of a share, which the
// proceeds and the shares imply, is only in one wide enough to leave the grant its room as well,
// and the narrowest window does without the grant too. A row has a cell for every column.
func TestSalesModel_Columns(t *testing.T) {
	cases := map[string]struct {
		width    int
		expected []string
	}{
		"the narrowest window": {
			width:    82,
			expected: []string{"Sold on", "Symbol", "Shares", "Proceeds", "Tax", "Gain"},
		},
		"room for the grant": {
			width:    100,
			expected: []string{"Sold on", "Symbol", "From", "Shares", "Proceeds", "Tax", "Gain"},
		},
		"room for the price": {
			width:    110,
			expected: []string{"Sold on", "Symbol", "From", "Shares", "Price", "Proceeds", "Tax", "Gain"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m, _ := newTestingSales(t)
			m.Update(tea.WindowSizeMsg{Width: c.width, Height: 20})

			var titles []string
			width := 0
			for _, column := range m.table.Columns() {
				titles = append(titles, strings.TrimSpace(column.Title))
				width += column.Width + cellPadding
			}

			assert.Equal(t, c.expected, titles)
			assert.Equal(t, tableWidth(c.width), width, "the columns should fill the table")
			assert.Len(t, m.table.Rows()[0], len(c.expected))
		})
	}
}

// TestSalesModel_Render is the frame the Sales view puts on screen, in the shape of the other
// views': what the sales have realized above the table next to the account values, a row for every
// sale, and the keys underneath.
func TestSalesModel_Render(t *testing.T) {
	m, _ := newTestingSales(t)

	golden.RequireEqual(t, ansi.Strip(m.View().Content))
}

// TestSalesModel_ShowsErrors covers a load that failed: the view says why where the prices otherwise
// are and keeps the sales it had on display, as the other views do.
func TestSalesModel_ShowsErrors(t *testing.T) {
	m, _ := newTestingSales(t)

	m.Update(PortfolioLoadedMsg{err: errors.New("database is locked")})

	assert.Contains(t, ansi.Strip(m.View().Content), "database is locked")
	assert.Len(t, m.table.Rows(), 2, "the sales should stay on display")
}

// TestSalesModel_DeleteSale covers taking a sale back: d asks before it deletes the selected sale,
// naming it, and only a y has the store delete it, which gives its shares back to their lot, after
// which the view loads the portfolio again. A no closes the question and deletes nothing.
func TestSalesModel_DeleteSale(t *testing.T) {
	m, store := newTestingSales(t)
	m.Update(keyPressed("j")) // the second sale

	m.Update(keyPressed("d"))
	require.True(t, m.CapturesKeys(), "the question should take the keyboard")
	frame := ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "Delete the sale of 1 PANW on 2026-09-15?")
	assert.Contains(t, frame, "y delete • n cancel")

	_, cmd := m.Update(keyPressed("n"))
	_, cmd = m.Update(runCmd(t, cmd))
	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "a no should close the question")

	m.Update(keyPressed("d"))
	_, cmd = m.Update(keyPressed("y"))
	msg := runCmd(t, cmd)
	require.Equal(t, DeleteSaleMsg{id: 2}, msg)

	store.On("DeleteSale", 2).Return(nil)
	_, cmd = m.Update(msg)
	assert.False(t, m.CapturesKeys(), "the question should be closed")
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)
}

// TestSalesModel_ClearSale covers settling the tax on one sale: c has the store mark the selected
// sale as cleared, without a question since another c takes it back, and the view loads the
// portfolio again. On a sale that is cleared the same key marks it as not cleared, and the help
// says which of the two it is going to do. Before anything was sold it does nothing.
func TestSalesModel_ClearSale(t *testing.T) {
	p := soldPortfolio()
	p.Sales[1].Cleared = true
	store := new(MockStore)
	store.returns(p)
	m := NewSalesModel(store, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))
	assert.Contains(t, ansi.Strip(m.View().Content), "d delete • c clear tax")

	store.On("ClearSales", []int{1}, true).Return(nil).Once()
	_, cmd := m.Update(keyPressed("c"))
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))

	m.Update(keyPressed("j")) // the second sale, which is cleared
	assert.Contains(t, ansi.Strip(m.View().Content), "d delete • c unclear tax")

	store.On("ClearSales", []int{2}, false).Return(nil).Once()
	_, cmd = m.Update(keyPressed("c"))
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)

	empty := NewSalesModel(new(MockStore), DefaultStyle())
	_, cmd = empty.Update(keyPressed("c"))
	assert.Nil(t, cmd)
}

// TestSalesModel_DeleteWithNothingSelected covers d before anything was sold, where there is nothing
// to ask about.
func TestSalesModel_DeleteWithNothingSelected(t *testing.T) {
	store := new(MockStore)
	store.returns(Portfolio{})
	m := NewSalesModel(store, DefaultStyle())
	m.Update(runCmd(t, m.Init()))

	_, cmd := m.Update(keyPressed("d"))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "there should be no question to answer")
	assert.Contains(t, ansi.Strip(m.View().Content), "Nothing sold yet")
}

// TestSalesModel_ShowsTheNote covers the notes of a sale, which the table has no column for: the
// note of the selected sale stands under the summary, where the prices otherwise are, all of its
// lines on that one. A sale without a note leaves the prices there.
func TestSalesModel_ShowsTheNote(t *testing.T) {
	m, _ := newTestingSales(t)
	assert.Contains(t, ansi.Strip(m.View().Content), "PANW $396.25")

	m.Update(keyPressed("j"))

	frame := ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "for the kitchen · sold in the morning")
	assert.NotContains(t, frame, "PANW $396.25 ▼")
}

// TestViews_SelectTheFirstRow covers a view whose table was empty and is then given rows, which is
// every view of an account that is just being set up, and the Sales view until the first sale: the
// selection is on the first row then, rather than on none, which would leave the keys that act on
// the selected row doing nothing until the selection was moved.
func TestViews_SelectTheFirstRow(t *testing.T) {
	store := new(MockStore)
	store.returns(Portfolio{})
	empty := PortfolioLoadedMsg{}
	loaded := PortfolioLoadedMsg{portfolio: soldPortfolio()}

	holdings := NewHoldingsModel(store, quotes{}, DefaultStyle())
	holdings.Update(empty)
	holdings.Update(loaded)
	assert.Equal(t, 0, holdings.table.Cursor(), "Holdings")

	vesting := NewVestingModel(store, DefaultStyle())
	vesting.Update(empty)
	vesting.Update(loaded)
	assert.Equal(t, 0, vesting.table.Cursor(), "Vesting")

	grants := NewGrantsModel(store, DefaultStyle())
	grants.Update(empty)
	grants.Update(loaded)
	assert.Equal(t, 0, grants.table.Cursor(), "Grants")

	sales := NewSalesModel(store, DefaultStyle())
	sales.Update(empty)
	sales.Update(loaded)
	assert.Equal(t, 0, sales.table.Cursor(), "Sales")
}

// TestSaleRow covers how one sale reads as a row of the Sales table: its day, the stock and the
// grant its lot was from, and the numbers right-aligned - how many shares, what one sold for, what
// that brought in, the tax on the gain in it, and that gain over what the shares cost. A gain says
// which way it goes with a sign, and a sale of a lot without a cost has none to state, and no tax.
// Neither has any sale of an account without a rate to tax a gain at. A loss is not taxed, and the
// tax on a sale that is cleared is checked off.
func TestSaleRow(t *testing.T) {
	rate := decimal.NewNullDecimal(dec("26.4"))

	cases := map[string]struct {
		sale     portfolio.Sale
		rate     decimal.NullDecimal
		expected table.Row
	}{
		"sold at a gain": {
			sale: portfolio.Sale{
				Date: day("2026-09-15"), Symbol: "PANW", Grant: "Payout",
				Shares: dec("50"), Price: dec("410.2"), Cost: dec("162.5"),
			},
			rate: rate,
			expected: table.Row{
				"2026-09-15", "PANW", "Payout", "        50", "   $410.20", "    $20,510.00", "   $3,269.64", "   +$12,385.00",
			},
		},
		"sold at a gain and cleared": {
			sale: portfolio.Sale{
				Date: day("2026-09-15"), Symbol: "PANW", Grant: "Payout",
				Shares: dec("50"), Price: dec("410.2"), Cost: dec("162.5"), Cleared: true,
			},
			rate: rate,
			expected: table.Row{
				"2026-09-15", "PANW", "Payout", "        50", "   $410.20", "    $20,510.00", " ✓ $3,269.64", "   +$12,385.00",
			},
		},
		"sold at a loss": {
			sale: portfolio.Sale{
				Date: day("2025-08-01"), Symbol: "PANW", Grant: "Payout",
				Shares: dec("230"), Price: dec("131.1961"), Cost: dec("162.5"),
			},
			rate: rate,
			expected: table.Row{
				"2025-08-01", "PANW", "Payout", "       230", "   $131.20", "    $30,175.10", "       $0.00", "    -$7,199.90",
			},
		},
		"a lot without a cost, entered by hand": {
			sale: portfolio.Sale{
				Date: day("2026-09-20"), Symbol: "AAPL",
				Shares: dec("2.5"), Price: dec("330"),
			},
			rate: rate,
			expected: table.Row{
				"2026-09-20", "AAPL", "", "       2.5", "   $330.00", "       $825.00", "           -", "             -",
			},
		},
		"no rate to tax a gain at": {
			sale: portfolio.Sale{
				Date: day("2026-09-15"), Symbol: "PANW", Grant: "Payout",
				Shares: dec("50"), Price: dec("410.2"), Cost: dec("162.5"), Cleared: true,
			},
			expected: table.Row{
				"2026-09-15", "PANW", "Payout", "        50", "   $410.20", "    $20,510.00", "         ✓ -", "   +$12,385.00",
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, saleRow(c.sale, c.rate))
		})
	}
}

// TestRealizedSummary covers the line the Sales view puts above its table, which is what tracking
// sales is for: how much money the sales have brought in, how much of that is gain, and how much tax
// is still owed on it, which is the tax on the sales that are not cleared. It says so if the gain
// leaves out sales whose cost is not known, and if nothing was sold yet. An account without a rate
// to tax a gain at is not told what it owes, and neither is one whose every gain is unknown.
func TestRealizedSummary(t *testing.T) {
	gain := portfolio.Sale{Shares: dec("50"), Price: dec("410.2"), Cost: dec("162.5")}
	cleared := portfolio.Sale{Shares: dec("2"), Price: dec("401.5"), Cost: dec("380.12"), Cleared: true}
	loss := portfolio.Sale{Shares: dec("10"), Price: dec("150"), Cost: dec("162.5")}
	uncosted := portfolio.Sale{Shares: dec("2.5"), Price: dec("330")}
	rate := decimal.NewNullDecimal(dec("26.4"))

	cases := map[string]struct {
		sales    []portfolio.Sale
		rate     decimal.NullDecimal
		expected string
	}{
		"nothing sold": {
			sales:    nil,
			rate:     rate,
			expected: "Nothing sold yet",
		},
		"a gain": {
			sales:    []portfolio.Sale{gain},
			rate:     rate,
			expected: "Realized $20,510.00 · gain +$12,385.00 · tax owed $3,269.64",
		},
		"a gain that is cleared": {
			sales:    []portfolio.Sale{gain, cleared},
			rate:     rate,
			expected: "Realized $21,313.00 · gain +$12,427.76 · tax owed $3,269.64",
		},
		"all cleared": {
			sales:    []portfolio.Sale{cleared},
			rate:     rate,
			expected: "Realized $803.00 · gain +$42.76 · tax owed $0.00",
		},
		"a loss overall": {
			sales:    []portfolio.Sale{loss},
			rate:     rate,
			expected: "Realized $1,500.00 · gain -$125.00 · tax owed $0.00",
		},
		"a sale without a cost": {
			sales:    []portfolio.Sale{gain, uncosted},
			rate:     rate,
			expected: "Realized $21,335.00 · gain +$12,385.00 without 1 sale of unknown cost · tax owed $3,269.64",
		},
		"only sales without a cost": {
			sales:    []portfolio.Sale{uncosted, uncosted},
			rate:     rate,
			expected: "Realized $1,650.00 · gain unknown",
		},
		"no rate to tax a gain at": {
			sales:    []portfolio.Sale{gain},
			expected: "Realized $20,510.00 · gain +$12,385.00",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, realizedSummary(c.sales, c.rate))
		})
	}
}
