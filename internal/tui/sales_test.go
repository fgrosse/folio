package tui

import (
	"errors"
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
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
	assert.Equal(t, saleRow(p.Sales[0]), rows[0])
	assert.Equal(t, saleRow(p.Sales[1]), rows[1])
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

// TestSaleRow covers how one sale reads as a row of the Sales table: its day, the stock and the
// grant its lot was from, and the numbers right-aligned - how many shares, what one sold for, what
// that brought in, and how much of it is gain over what the shares cost. A gain says which way it
// goes with a sign, and a sale of a lot without a cost has none to state.
func TestSaleRow(t *testing.T) {
	tests := map[string]struct {
		sale     portfolio.Sale
		expected table.Row
	}{
		"sold at a gain": {
			sale: portfolio.Sale{
				Date: day("2026-09-15"), Symbol: "PANW", Grant: "Payout",
				Shares: dec("50"), Price: dec("410.2"), Cost: dec("162.5"),
			},
			expected: table.Row{"2026-09-15", "PANW", "Payout", "        50", "   $410.20", "    $20,510.00", "   +$12,385.00"},
		},
		"sold at a loss": {
			sale: portfolio.Sale{
				Date: day("2025-08-01"), Symbol: "PANW", Grant: "Payout",
				Shares: dec("230"), Price: dec("131.1961"), Cost: dec("162.5"),
			},
			expected: table.Row{"2025-08-01", "PANW", "Payout", "       230", "   $131.20", "    $30,175.10", "    -$7,199.90"},
		},
		"a lot without a cost, entered by hand": {
			sale: portfolio.Sale{
				Date: day("2026-09-20"), Symbol: "AAPL",
				Shares: dec("2.5"), Price: dec("330"),
			},
			expected: table.Row{"2026-09-20", "AAPL", "", "       2.5", "   $330.00", "       $825.00", "             -"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, saleRow(tt.sale))
		})
	}
}

// TestRealizedSummary covers the line the Sales view puts above its table, which is what tracking
// sales is for: how much money the sales have brought in, and how much of that is gain. It says so
// if the gain leaves out sales whose cost is not known, and if nothing was sold yet.
func TestRealizedSummary(t *testing.T) {
	gain := portfolio.Sale{Shares: dec("50"), Price: dec("410.2"), Cost: dec("162.5")}
	loss := portfolio.Sale{Shares: dec("10"), Price: dec("150"), Cost: dec("162.5")}
	uncosted := portfolio.Sale{Shares: dec("2.5"), Price: dec("330")}

	tests := map[string]struct {
		sales    []portfolio.Sale
		expected string
	}{
		"nothing sold": {
			sales:    nil,
			expected: "Nothing sold yet",
		},
		"a gain": {
			sales:    []portfolio.Sale{gain},
			expected: "Realized $20,510.00 · gain +$12,385.00",
		},
		"a loss overall": {
			sales:    []portfolio.Sale{loss},
			expected: "Realized $1,500.00 · gain -$125.00",
		},
		"a sale without a cost": {
			sales:    []portfolio.Sale{gain, uncosted},
			expected: "Realized $21,335.00 · gain +$12,385.00 without 1 sale of unknown cost",
		},
		"only sales without a cost": {
			sales:    []portfolio.Sale{uncosted, uncosted},
			expected: "Realized $1,650.00 · gain unknown",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, realizedSummary(tt.sales))
		})
	}
}
