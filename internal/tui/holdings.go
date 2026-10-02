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
	"charm.land/lipgloss/v2"
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
	quoter    portfolio.Quoter // where fresh prices come from
	style     Style
	keys      keyMap
	table     table.Model
	width     int              // width of the table, which the header line is spread across
	now       func() time.Time // the clock that says what today is
	portfolio Portfolio        // the account as it was last loaded
	err       error            // why the last load failed or had no fresh quotes, nil unless it did

	refreshInterval time.Duration  // how long the quotes are left alone before they are fetched again
	refreshing      bool           // a RefreshQuotesMsg is on its way, so no load needs to schedule another
	input           *InputDialog   // the dialog that adds a lot, nil unless it is open
	confirm         *ConfirmDialog // the dialog asking whether to delete a lot, nil unless it is open
}

// RefreshQuotesMsg asks the view to fetch the quotes again, which it does every refreshInterval.
type RefreshQuotesMsg struct{}

// DeleteLotMsg reports that the user confirmed deleting the lot with the given ID.
type DeleteLotMsg struct {
	id int
}

// SaveLotMsg reports that the dialog was confirmed with the spec of a lot. The lot has not been saved
// yet; that is up to whoever receives it.
type SaveLotMsg struct {
	lot portfolio.Lot
}

// NewHoldingsModel returns the Holdings view over store, rendered in style. It is the view that
// keeps the prices fresh, with the quotes it fetches from quoter.
func NewHoldingsModel(store Store, quoter portfolio.Quoter, style Style) *HoldingsModel {
	m := &HoldingsModel{
		store:  store,
		quoter: quoter,
		style:  style,
		keys:   defaultKeyMap(),
		now:    time.Now,
		// Often enough to follow a trading day, and seldom enough for a source of quotes that is
		// not an official API to put up with.
		refreshInterval: 5 * time.Minute,
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

// CapturesKeys implements KeyCapturer: while a dialog is open the keyboard belongs to it, down to
// the keys that would otherwise switch views. The digits are the shares of the lot being typed.
func (m *HoldingsModel) CapturesKeys() bool {
	return m.input != nil || m.confirm != nil
}

// Init implements tea.Model by loading the portfolio as the store has it, which is on screen at
// once, and fetching fresh quotes for it at the same time, which takes as long as the network does
// and loads the portfolio a second time.
func (m *HoldingsModel) Init() tea.Cmd {
	return tea.Batch(
		loadPortfolioCmd(m.store),
		refreshQuotesCmd(m.store, m.quoter),
	)
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
		return m.handlePortfolioLoaded(msg)
	case RefreshQuotesMsg:
		m.refreshing = false
		return m, refreshQuotesCmd(m.store, m.quoter)
	case SaveLotMsg:
		m.input = nil
		return m, m.saveLotCmd(msg.lot)
	case InputCanceledMsg:
		m.input = nil
		return m, nil
	case DeleteLotMsg:
		m.confirm = nil
		return m, m.deleteLotCmd(msg.id)
	case ConfirmCanceledMsg:
		m.confirm = nil
		return m, nil
	}

	if m.input != nil {
		// Whatever the view does not handle may be the dialog's, such as its cursor's blink ticks.
		return m, m.input.Update(msg)
	}

	return m, nil
}

// handlePortfolioLoaded takes over the portfolio that was loaded and fills the table with its lots.
// A load that failed leaves the one on display where it is, and its error to the header. Either way
// it sees to it that the quotes are fetched again in a while.
func (m *HoldingsModel) handlePortfolioLoaded(msg PortfolioLoadedMsg) (tea.Model, tea.Cmd) {
	m.err = msg.err
	if msg.err == nil {
		// Quotes that could not be refreshed do not make the portfolio any less the latest there is.
		m.err = msg.quotesErr
		m.portfolio = msg.portfolio
		m.updateRows()
	}

	return m, m.scheduleRefreshCmd()
}

// scheduleRefreshCmd returns a command that sends a RefreshQuotesMsg once the refresh interval has
// passed, or nil if one is on its way already: loads come from saving, deleting and refreshing
// alike, and a timer for each would have the quotes fetched ever more often.
func (m *HoldingsModel) scheduleRefreshCmd() tea.Cmd {
	if m.refreshing {
		return nil
	}

	m.refreshing = true
	return tea.Tick(m.refreshInterval, func(time.Time) tea.Msg { return RefreshQuotesMsg{} })
}

// handleKeyPress quits on the quit keys, opens the dialog for a new lot on the add key, asks whether
// to delete the selected lot on the delete key, and hands every other key to the table, which moves
// the selection. While a dialog is open, every key is the dialog's.
func (m *HoldingsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.input != nil {
		return m, m.input.HandleKeyPress(msg)
	}
	if m.confirm != nil {
		return m, m.confirm.HandleKeyPress(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Delete):
		return m.askToDelete()
	case key.Matches(msg, m.keys.Add):
		m.input = NewInputDialog("New lot", "12.5 PANW 2026-03-15", m.newLot, dialogWidth, m.style)
		return m, m.input.Init()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// askToDelete opens a dialog asking whether to delete the selected lot, which sends a DeleteLotMsg
// if the answer is yes. Without a lot to select, there is nothing to ask.
func (m *HoldingsModel) askToDelete() (tea.Model, tea.Cmd) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.portfolio.Lots) {
		return m, nil
	}

	lot := m.portfolio.Lots[i]
	question := fmt.Sprintf("Delete the %s %s of %s?", lot.Shares, lot.Symbol, lot.Acquired.Format(time.DateOnly))
	m.confirm = NewConfirmDialog("Delete lot", question, DeleteLotMsg{id: lot.ID}, m.style)

	return m, nil
}

// deleteLotCmd returns a command that deletes the lot with the given ID and loads the portfolio
// again, which then no longer has it.
func (m *HoldingsModel) deleteLotCmd(id int) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		if err := store.DeleteLot(id); err != nil {
			return PortfolioLoadedMsg{err: err}
		}

		return loadPortfolioCmd(store)()
	}
}

// newLot turns the spec typed into the dialog into the message that asks for the lot to be saved. A
// lot whose spec has no day is of today.
func (m *HoldingsModel) newLot(spec string) (tea.Msg, error) {
	lot, err := portfolio.NewLot(spec)
	if err != nil {
		return nil, err
	}

	if lot.Acquired.IsZero() {
		lot.Acquired = portfolio.DayOf(m.now())
	}

	return SaveLotMsg{lot: lot}, nil
}

// saveLotCmd returns a command that saves lot and loads the portfolio again, which then has it.
func (m *HoldingsModel) saveLotCmd(lot portfolio.Lot) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		if err := store.SaveLot(lot); err != nil {
			return PortfolioLoadedMsg{err: err}
		}

		return loadPortfolioCmd(store)()
	}
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

	layers := []*lipgloss.Layer{lipgloss.NewLayer(frame)}
	if dialog := m.dialogLayer(); dialog != nil {
		// The dialog floats centered in front of everything else.
		dialog.X((lipgloss.Width(frame) - dialog.Width()) / 2)
		dialog.Y((lipgloss.Height(frame) - dialog.Height()) / 2)
		layers = append(layers, dialog)
	}

	c := lipgloss.NewCompositor(layers...)
	return tea.NewView(trimTrailingSpace(c.Render()) + "\n")
}

// dialogLayer renders whichever dialog is open, or returns nil if none is.
func (m *HoldingsModel) dialogLayer() *lipgloss.Layer {
	switch {
	case m.input != nil:
		return m.input.Layer()
	case m.confirm != nil:
		return m.confirm.Layer()
	default:
		return nil
	}
}

// headerView renders the two lines above the table: the positions the lots add up to on the left,
// with the prices they are valued at underneath, and the account values on the right. If the last
// load failed, why it did stands where the prices otherwise are.
func (m *HoldingsModel) headerView() string {
	return portfolioHeader(positions(m.portfolio.Lots), m.portfolio, m.err, m.width-cellPadding, m.style)
}

// helpView renders the keys worth knowing in two lines, as every view does: getting around on the
// first, and what can be done to the table on the second. While a dialog is open it shows the
// dialog's keys instead, since none of the others reach the view in that state anyway. Those take a
// single line, and the second is left empty so that the frame keeps its height.
func (m *HoldingsModel) helpView() string {
	help, nav := m.table.Help, m.table.KeyMap

	switch {
	case m.input != nil:
		return help.ShortHelpView([]key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}) + "\n"
	case m.confirm != nil:
		return help.ShortHelpView([]key.Binding{
			key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "delete")),
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "cancel")),
		}) + "\n"
	}

	return help.ShortHelpView([]key.Binding{nav.LineUp, nav.LineDown, m.keys.Quit}) + "\n" +
		help.ShortHelpView([]key.Binding{m.keys.Add, m.keys.Delete})
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
