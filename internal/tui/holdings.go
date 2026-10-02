package tui

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// Column widths of the Holdings table. Every column except the symbol is bounded by its own
// content, so the symbol is the one that flexes to fill whatever room the window leaves.
const (
	// dayColumnWidth fits a day written as YYYY-MM-DD.
	dayColumnWidth = 10

	// sharesColumnWidth fits a number of shares up to 9,999,999, or fewer with a fraction.
	sharesColumnWidth = 10

	// priceColumnWidth fits the price of one share up to $99,999.99.
	priceColumnWidth = 10

	// valueColumnWidth fits a value up to $9,999,999.99, which is as much as the header's total
	// will ever have to add up.
	valueColumnWidth = 14

	// holdingsColumnsWidth is what every column other than the symbol occupies, padding included.
	holdingsColumnsWidth = dayColumnWidth + sharesColumnWidth + priceColumnWidth + valueColumnWidth + 4*cellPadding

	// noValue stands where a price or value would be if there was a quote to work it out from.
	noValue = "-"
)

// A HoldingsModel is the Holdings view: every lot of the account, with what it is worth at the
// latest price of its stock.
type HoldingsModel struct {
	store     Store
	style     Style
	keys      keyMap
	table     table.Model
	width     int       // width of the table, which the header line is spread across
	portfolio Portfolio // the account as it was last loaded
}

// NewHoldingsModel returns the Holdings view over store, rendered in style.
func NewHoldingsModel(store Store, style Style) *HoldingsModel {
	m := &HoldingsModel{
		store: store,
		style: style,
		keys:  defaultKeyMap(),
		// A real width arrives with the first WindowSizeMsg, which Bubble Tea sends at startup.
		// Until then the narrowest supported layout is the safest thing to hold.
		width: minTableWidth,
	}
	m.table = newTable(style, m.columns())

	return m
}

// columns returns the table's columns, the symbol column taking whatever the width leaves. The
// titles of the number columns are padded like the numbers under them, so that they end where the
// numbers do.
func (m *HoldingsModel) columns() []table.Column {
	return []table.Column{
		{Title: "Acquired", Width: dayColumnWidth},
		{Title: "Symbol", Width: m.width - holdingsColumnsWidth - cellPadding},
		{Title: fmt.Sprintf("%*s", sharesColumnWidth, "Shares"), Width: sharesColumnWidth},
		{Title: fmt.Sprintf("%*s", priceColumnWidth, "Price"), Width: priceColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Value"), Width: valueColumnWidth},
	}
}

// Title implements ViewModel by naming the view in the app's tab bar.
func (m *HoldingsModel) Title() string {
	return "Holdings"
}

// Init implements tea.Model by loading the portfolio.
func (m *HoldingsModel) Init() tea.Cmd {
	return loadPortfolioCmd(m.store)
}

// Update implements tea.Model by fitting the table to the window, filling it with the lots of the
// portfolio once it is loaded, and moving the selection through it.
func (m *HoldingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.width = tableWidth(msg.Width)
		m.table.SetColumns(m.columns())
		m.table.SetWidth(m.width)
		m.table.SetHeight(msg.Height - chromeHeight)
		return m, nil
	case PortfolioLoadedMsg:
		m.portfolio = msg.portfolio
		m.updateRows()
		return m, nil
	}

	return m, nil
}

// handleKeyPress quits on the quit keys and hands every other key to the table, which moves the
// selection.
func (m *HoldingsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// updateRows fills the table with a row for every lot of the portfolio.
func (m *HoldingsModel) updateRows() {
	rows := make([]table.Row, len(m.portfolio.Lots))
	for i, lot := range m.portfolio.Lots {
		rows[i] = lotRow(lot, m.portfolio.Quotes[lot.Symbol])
	}

	m.table.SetRows(rows)
}

// View implements tea.Model by rendering the header above the table of lots, and the keys below it.
func (m *HoldingsModel) View() tea.View {
	frame := m.headerView() + "\n" +
		m.style.Table.Render(m.table.View()) + "\n" +
		m.helpView()

	return tea.NewView(trimTrailingSpace(frame) + "\n")
}

// headerView renders the two lines above the table: the positions the lots add up to on the left,
// with the prices they are valued at underneath, and the account values on the right.
func (m *HoldingsModel) headerView() string {
	p := m.portfolio
	account := portfolio.NewAccount(p.Lots, p.Grants, p.Quotes)
	status := m.style.Hint.Render(quoteStatus(portfolio.Symbols(p.Lots, p.Grants), p.Quotes))

	return accountHeader(positions(p.Lots), status, account, m.width-cellPadding, m.style)
}

// helpView renders the keys worth knowing in two lines, as every view does: getting around on the
// first, and what can be done to the table on the second, which is nothing yet and stays empty so
// that the frame has the height it will have.
func (m *HoldingsModel) helpView() string {
	help, nav := m.table.Help, m.table.KeyMap

	return help.ShortHelpView([]key.Binding{nav.LineUp, nav.LineDown, m.keys.Quit}) + "\n"
}

// positions sums up lots as how many shares of each stock they hold, the stocks in alphabetical
// order, such as "3 AAPL · 10 PANW". It is what the Holdings view says above its table, where the
// rows only have the shares of one lot each.
func positions(lots []portfolio.Lot) string {
	if len(lots) == 0 {
		return "No shares held"
	}

	shares := make(map[string]decimal.Decimal)
	for _, lot := range lots {
		shares[lot.Symbol] = shares[lot.Symbol].Add(lot.Shares)
	}

	var parts []string
	for _, symbol := range slices.Sorted(maps.Keys(shares)) {
		parts = append(parts, shares[symbol].String()+" "+symbol)
	}

	return strings.Join(parts, " · ")
}

// lotRow renders a lot as a row of the Holdings table, valued at quote, which is the zero Quote if
// there is none of the lot's stock. The numbers are right-aligned for their digits to line up down
// the column. The table has no alignment of its own, so the values are padded out here.
func lotRow(lot portfolio.Lot, quote portfolio.Quote) table.Row {
	price, value := noValue, noValue
	if quote.Symbol != "" {
		price = portfolio.FormatUSD(quote.Price)
		value = portfolio.FormatUSD(lot.Shares.Mul(quote.Price))
	}

	return table.Row{
		lot.Acquired.Format(time.DateOnly),
		lot.Symbol,
		fmt.Sprintf("%*s", sharesColumnWidth, lot.Shares),
		fmt.Sprintf("%*s", priceColumnWidth, price),
		fmt.Sprintf("%*s", valueColumnWidth, value),
	}
}
