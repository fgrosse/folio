package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// salesColumnsWidth is what every column of the Sales table other than the grant occupies, padding
// included.
const salesColumnsWidth = dayColumnWidth + symbolColumnWidth + sharesColumnWidth + priceColumnWidth + 2*valueColumnWidth +
	6*cellPadding

// A SalesModel is the Sales view: every sale, with what it brought in and gained. Together they are
// the part of the account that has been turned into money.
type SalesModel struct {
	store     Store
	style     Style
	keys      keyMap
	table     table.Model
	width     int       // width of the table, which the header line is spread across
	portfolio Portfolio // the account as it was last loaded
	err       error     // why the last load failed or had no fresh quotes, nil unless it did
}

// NewSalesModel returns the Sales view over store, rendered in style.
func NewSalesModel(store Store, style Style) *SalesModel {
	m := &SalesModel{
		store: store,
		style: style,
		keys:  defaultKeyMap(),
		// A real width arrives with the first WindowSizeMsg, as it does for the other views.
		width: minTableWidth,
	}
	m.table = newTable(style, m.columns())

	return m
}

// columns returns the table's columns, the column of the grant taking whatever the width leaves.
// The titles of the number columns are padded like the numbers under them, so that they end where
// the numbers do.
func (m *SalesModel) columns() []table.Column {
	return []table.Column{
		{Title: "Sold on", Width: dayColumnWidth},
		{Title: "Symbol", Width: symbolColumnWidth},
		{Title: "From", Width: m.width - salesColumnsWidth - cellPadding},
		{Title: fmt.Sprintf("%*s", sharesColumnWidth, "Shares"), Width: sharesColumnWidth},
		{Title: fmt.Sprintf("%*s", priceColumnWidth, "Price"), Width: priceColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Proceeds"), Width: valueColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Gain"), Width: valueColumnWidth},
	}
}

// Title implements ViewModel by naming the view in the app's tab bar.
func (m *SalesModel) Title() string {
	return "Sales"
}

// Init implements tea.Model by loading the portfolio.
func (m *SalesModel) Init() tea.Cmd {
	return loadPortfolioCmd(m.store)
}

// Update implements tea.Model by fitting the table to the window, filling it with the sales of the
// portfolio once it is loaded, and moving the selection through it.
func (m *SalesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.err = msg.err
		if msg.err == nil {
			// Quotes that could not be refreshed do not make the portfolio any less the latest.
			m.err = msg.quotesErr
			m.portfolio = msg.portfolio
			m.updateRows()
		}
		return m, nil
	}

	return m, nil
}

// handleKeyPress quits on the quit keys and hands every other key to the table, which moves the
// selection.
func (m *SalesModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// updateRows fills the table with a row for every sale of the portfolio.
func (m *SalesModel) updateRows() {
	rows := make([]table.Row, len(m.portfolio.Sales))
	for i, sale := range m.portfolio.Sales {
		rows[i] = saleRow(sale)
	}

	m.table.SetRows(rows)
}

// View implements tea.Model by rendering the header above the table of sales, and the keys below it.
func (m *SalesModel) View() tea.View {
	frame := m.headerView() + "\n" +
		m.style.Table.Render(m.table.View()) + "\n" +
		m.helpView()

	return tea.NewView(trimTrailingSpace(frame) + "\n")
}

// headerView renders the two lines above the table: what the sales have realized on the left, with
// the prices underneath, and the account values on the right.
func (m *SalesModel) headerView() string {
	return portfolioHeader(realizedSummary(m.portfolio.Sales), m.portfolio, m.err, m.width-cellPadding, m.style)
}

// helpView renders the keys worth knowing in two lines, as every view does: getting around on the
// first, and what can be done to the table on the second, which is nothing yet and stays empty so
// that the frame has the height of the other views'.
func (m *SalesModel) helpView() string {
	help, nav := m.table.Help, m.table.KeyMap

	return help.ShortHelpView([]key.Binding{nav.LineUp, nav.LineDown, m.keys.Quit}) + "\n"
}

// realizedSummary sums up sales as the money they brought in and the gain in it, such as "Realized
// $20,510.00 · gain +$12,385.00". It is what the Sales view says above its table. The gain is that
// of the sales whose cost is known, and the summary says how many it leaves out.
func realizedSummary(sales []portfolio.Sale) string {
	if len(sales) == 0 {
		return "Nothing sold yet"
	}

	realized := portfolio.NewRealized(sales)
	summary := "Realized " + portfolio.FormatUSD(realized.Proceeds) + " · gain "

	switch {
	case realized.Uncosted == len(sales):
		return summary + "unknown"
	case realized.Uncosted > 0:
		return summary + formatGain(realized.Gain) + " without " + count(realized.Uncosted, "sale") + " of unknown cost"
	default:
		return summary + formatGain(realized.Gain)
	}
}

// saleRow renders a sale as a row of the Sales table: its day, the stock and the grant of the lot it
// was sold from, the shares and the price of one, and what the sale brought in and gained. The
// numbers are right-aligned and padded out like those of the other tables.
func saleRow(sale portfolio.Sale) table.Row {
	gain := noValue
	if amount, known := sale.Gain(); known {
		gain = formatGain(amount)
	}

	return table.Row{
		sale.Date.Format(time.DateOnly),
		sale.Symbol,
		sale.Grant,
		fmt.Sprintf("%*s", sharesColumnWidth, sale.Shares),
		fmt.Sprintf("%*s", priceColumnWidth, portfolio.FormatUSD(sale.Price)),
		fmt.Sprintf("%*s", valueColumnWidth, portfolio.FormatUSD(sale.Proceeds())),
		fmt.Sprintf("%*s", valueColumnWidth, gain),
	}
}

// formatGain renders a gain in dollars with its sign in front, a plus as well as a minus, so that
// it reads as a change rather than as an amount.
func formatGain(gain decimal.Decimal) string {
	if gain.IsNegative() {
		return portfolio.FormatUSD(gain)
	}

	return "+" + portfolio.FormatUSD(gain)
}
