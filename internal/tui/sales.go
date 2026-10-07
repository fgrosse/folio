package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

const (
	// taxColumnWidth fits the tax on a sale up to $99,999.99 with the mark of a cleared sale in
	// front of it.
	taxColumnWidth = 12

	// salesColumnsWidth is what every column of the Sales table other than the grant occupies,
	// padding included.
	salesColumnsWidth = dayColumnWidth + symbolColumnWidth + sharesColumnWidth + priceColumnWidth + 2*valueColumnWidth +
		taxColumnWidth + 7*cellPadding

	// clearedMark stands in front of the tax on a sale that is cleared.
	clearedMark = "✓"
)

// A SalesModel is the Sales view: every sale, with what it brought in and gained. Together they are
// the part of the account that has been turned into money.
type SalesModel struct {
	store     Store
	style     Style
	keys      keyMap
	table     table.Model
	width     int            // width of the table, which the header line is spread across
	portfolio Portfolio      // the account as it was last loaded
	err       error          // why the last load failed or had no fresh quotes, nil unless it did
	confirm   *ConfirmDialog // the dialog asking whether to delete a sale, nil unless it is open
}

// DeleteSaleMsg reports that the user confirmed deleting the sale with the given ID.
type DeleteSaleMsg struct {
	id int
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
		{Title: "From", Width: max(m.width-salesColumnsWidth-cellPadding, 0)},
		{Title: fmt.Sprintf("%*s", sharesColumnWidth, "Shares"), Width: sharesColumnWidth},
		{Title: fmt.Sprintf("%*s", priceColumnWidth, "Price"), Width: priceColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Proceeds"), Width: valueColumnWidth},
		{Title: fmt.Sprintf("%*s", taxColumnWidth, "Tax"), Width: taxColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Gain"), Width: valueColumnWidth},
	}
}

// Title implements ViewModel by naming the view in the app's tab bar.
func (m *SalesModel) Title() string {
	return "Sales"
}

// CapturesKeys implements KeyCapturer: while the question is open the keyboard belongs to it, down
// to the keys that would otherwise switch views.
func (m *SalesModel) CapturesKeys() bool {
	return m.confirm != nil
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
	case DeleteSaleMsg:
		m.confirm = nil
		return m, m.deleteSaleCmd(msg.id)
	case ConfirmCanceledMsg:
		m.confirm = nil
		return m, nil
	}

	return m, nil
}

// handleKeyPress quits on the quit keys, asks whether to delete the selected sale on the delete key,
// and hands every other key to the table, which moves the selection. While the question is open,
// every key is its own.
func (m *SalesModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirm != nil {
		return m, m.confirm.HandleKeyPress(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Delete):
		return m.askToDelete()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// selected returns the sale the selection is on, if there is one.
func (m *SalesModel) selected() (portfolio.Sale, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.portfolio.Sales) {
		return portfolio.Sale{}, false
	}

	return m.portfolio.Sales[i], true
}

// askToDelete opens a dialog asking whether to delete the selected sale, which sends a DeleteSaleMsg
// if the answer is yes. Before anything was sold, there is nothing to ask.
func (m *SalesModel) askToDelete() (tea.Model, tea.Cmd) {
	sale, ok := m.selected()
	if !ok {
		return m, nil
	}

	question := fmt.Sprintf("Delete the sale of %s %s on %s?", sale.Shares, sale.Symbol, sale.Date.Format(time.DateOnly))
	m.confirm = NewConfirmDialog("Delete sale", question, DeleteSaleMsg{id: sale.ID}, m.style)

	return m, nil
}

// deleteSaleCmd returns a command that deletes the sale with the given ID and loads the portfolio
// again, in which the shares of the sale are their lot's again.
func (m *SalesModel) deleteSaleCmd(id int) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		if err := store.DeleteSale(id); err != nil {
			return PortfolioLoadedMsg{err: err}
		}

		return loadPortfolioCmd(store)()
	}
}

// updateRows fills the table with a row for every sale of the portfolio.
func (m *SalesModel) updateRows() {
	rows := make([]table.Row, len(m.portfolio.Sales))
	for i, sale := range m.portfolio.Sales {
		rows[i] = saleRow(sale, m.portfolio.GainsTaxRate)
	}

	setRows(&m.table, rows)
}

// View implements tea.Model by rendering the header above the table of sales, and the keys below it.
func (m *SalesModel) View() tea.View {
	frame := m.headerView() + "\n" +
		m.style.Table.Render(m.table.View()) + "\n" +
		m.helpView()

	layers := []*lipgloss.Layer{lipgloss.NewLayer(frame)}
	if m.confirm != nil {
		// The question floats centered in front of the table, as those of the other views do.
		dialog := m.confirm.Layer()
		dialog.X((lipgloss.Width(frame) - dialog.Width()) / 2)
		dialog.Y((lipgloss.Height(frame) - dialog.Height()) / 2)
		layers = append(layers, dialog)
	}

	c := lipgloss.NewCompositor(layers...)
	return tea.NewView(trimTrailingSpace(c.Render()) + "\n")
}

// headerView renders the two lines above the table: what the sales have realized on the left, and
// the account values on the right. Under the summary is the note of the selected sale, which the
// table has no column for, all of its lines on that one, or the prices if the sale has no note.
func (m *SalesModel) headerView() string {
	var note string
	if sale, ok := m.selected(); ok {
		note = strings.Join(strings.Fields(strings.ReplaceAll(sale.Note, "\n", " · ")), " ")
	}

	return notedHeader(realizedSummary(m.portfolio.Sales), note, m.portfolio, m.err, m.width-cellPadding, m.style)
}

// helpView renders the keys worth knowing in two lines, as every view does: getting around on the
// first, and what can be done to the selected sale on the second. While the question is open it
// shows the keys that answer it instead, on one line, and leaves the second empty so that the frame
// keeps its height.
func (m *SalesModel) helpView() string {
	help, nav := m.table.Help, m.table.KeyMap

	if m.confirm != nil {
		return help.ShortHelpView([]key.Binding{
			key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "delete")),
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "cancel")),
		}) + "\n"
	}

	return help.ShortHelpView([]key.Binding{nav.LineUp, nav.LineDown, m.keys.Quit}) + "\n" +
		help.ShortHelpView([]key.Binding{m.keys.Delete})
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
// was sold from, the shares and the price of one, what the sale brought in, the tax on its gain at
// rate, and the gain. The numbers are right-aligned and padded out like those of the other tables.
func saleRow(sale portfolio.Sale, rate decimal.NullDecimal) table.Row {
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
		fmt.Sprintf("%*s", taxColumnWidth, saleTax(sale, rate)),
		fmt.Sprintf("%*s", valueColumnWidth, gain),
	}
}

// saleTax renders the tax on the gain of sale at rate, which is in percent and not valid if the
// account has none. A sale that is cleared has its tax checked off, whether or not folio can say
// how much it was: that it is settled is the user's to say.
func saleTax(sale portfolio.Sale, rate decimal.NullDecimal) string {
	tax := noValue
	if amount, known := sale.Tax(rate.Decimal); known && rate.Valid {
		tax = portfolio.FormatUSD(amount)
	}

	if sale.Cleared {
		return clearedMark + " " + tax
	}

	return tax
}

// formatGain renders a gain in dollars with its sign in front, a plus as well as a minus, so that
// it reads as a change rather than as an amount.
func formatGain(gain decimal.Decimal) string {
	if gain.IsNegative() {
		return portfolio.FormatUSD(gain)
	}

	return "+" + portfolio.FormatUSD(gain)
}
