package tui

import (
	"errors"
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

// Column widths of the Holdings table. Every column except the grant a lot is from is bounded by
// its own content, so that is the one that flexes to fill whatever room the window leaves. Not every
// column is in every window: the ones that say what the others imply are left out of a window too
// narrow for them, which is what HoldingsModel.layout decides.
const (
	// dayColumnWidth fits a day written as YYYY-MM-DD.
	dayColumnWidth = 10

	// sharesColumnWidth fits a number of shares up to 9,999,999, or fewer with a fraction.
	sharesColumnWidth = 10

	// priceColumnWidth fits the price of one share up to $99,999.99, which is what both its cost
	// and its price now are.
	priceColumnWidth = 10

	// gainColumnWidth fits how far a price is from a cost, in percent, up to +9999.9%.
	gainColumnWidth = 8

	// valueColumnWidth fits a value up to $9,999,999.99, which is as much as the header's total
	// will ever have to add up.
	valueColumnWidth = 14

	// holdingsColumnsWidth is what the columns that are always there occupy, other than the grant
	// and with their padding.
	holdingsColumnsWidth = dayColumnWidth + symbolColumnWidth + sharesColumnWidth + 2*priceColumnWidth + valueColumnWidth +
		6*cellPadding

	// grantColumnMinWidth is the least the column of the grant is left with by a column that is not
	// always there: what it has in the narrowest window, with none of them.
	grantColumnMinWidth = minTableWidth - holdingsColumnsWidth - cellPadding

	// gainColumn and taxColumn are where the gain and the tax stand among all the columns, for
	// leaving them out.
	gainColumn = 6
	taxColumn  = 7

	// saleFormWidth is how many columns a field of the form that records a sale occupies, and
	// saleNoteLines how many lines its notes have room for before they scroll.
	saleFormWidth = 44
	saleNoteLines = 3

	// lotPlaceholder shows how a lot is written, in the empty field of the dialog that takes one.
	lotPlaceholder = "12.5 PANW 2026-03-15 @380.12"

	// noValue stands where a cost, price or value would be if it was known.
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
	showGain  bool             // the window has room for the column of the gain
	showTax   bool             // and for that of the tax on it, which the account has a rate for
	now       func() time.Time // the clock that says what today is
	portfolio Portfolio        // the account as it was last loaded
	lots      []portfolio.Lot  // the lots on display, as the table shows them: those with shares left
	err       error            // why the last load failed or had no fresh quotes, nil unless it did

	refreshInterval time.Duration  // how long the quotes are left alone before they are fetched again
	refreshing      bool           // a RefreshQuotesMsg is on its way, so no load needs to schedule another
	input           *InputDialog   // the dialog that adds or edits a lot, nil unless it is open
	confirm         *ConfirmDialog // the dialog asking whether to delete a lot, nil unless it is open
	form            *FormDialog    // the form that records a sale, nil unless it is open
	details         bool           // the flyout with the details of the selected lot is open
}

// RefreshQuotesMsg asks the view to fetch the quotes again, which it does every refreshInterval.
type RefreshQuotesMsg struct{}

// SaveSaleMsg reports that the form was confirmed with a sale of shares of a lot. The sale has not
// been saved yet; that is up to whoever receives it.
type SaveSaleMsg struct {
	sale portfolio.Sale
}

// DeleteLotMsg reports that the user confirmed deleting the lot with the given ID.
type DeleteLotMsg struct {
	id int
}

// SaveLotMsg reports that the dialog was confirmed with the spec of a lot. The lot has not been saved
// yet; that is up to whoever receives it. A new lot has no ID, and an edited one keeps its own.
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

// layout decides which columns the table has room for at its width and sets it up with those, and
// with the rows of the portfolio to match. The gain comes first, and the tax on it only where there
// is room for both, and where the account has a rate to tax a gain at: a column of dashes would
// take the room of the grant for nothing. The selection stays on the row it was on.
func (m *HoldingsModel) layout() {
	room := m.grantColumnWidth() - grantColumnMinWidth
	m.showGain = room >= gainColumnWidth+cellPadding
	m.showTax = m.portfolio.GainsTaxRate.Valid && room >= gainColumnWidth+valueColumnWidth+2*cellPadding

	// The table renders as soon as it is given columns or rows, and panics over a row with more
	// cells than it has columns. So the rows go before the columns change and come back after.
	cursor := m.table.Cursor()
	m.table.SetRows(nil)
	m.table.SetColumns(m.columns())
	m.updateRows()
	m.table.SetCursor(cursor)
}

// grantColumnWidth is what the columns that are always there leave to the column of the grant.
func (m *HoldingsModel) grantColumnWidth() int {
	return m.width - holdingsColumnsWidth - cellPadding
}

// shown returns those of cells, which are one for every column of the table, whose columns the
// window has room for. It is what keeps the columns and the cells of a row the same in number.
func shown[T any](m *HoldingsModel, cells []T) []T {
	cells = slices.Clone(cells)

	// The one further back first, so that the other is still where its index says.
	if !m.showTax {
		cells = slices.Delete(cells, taxColumn, taxColumn+1)
	}
	if !m.showGain {
		cells = slices.Delete(cells, gainColumn, gainColumn+1)
	}

	return cells
}

// columns returns the table's columns, the column of the grant taking whatever the width leaves. The
// titles of the number columns are padded like the numbers under them, so that they end where the
// numbers do.
func (m *HoldingsModel) columns() []table.Column {
	grantWidth := m.grantColumnWidth()
	if m.showGain {
		grantWidth -= gainColumnWidth + cellPadding
	}
	if m.showTax {
		grantWidth -= valueColumnWidth + cellPadding
	}

	return shown(m, []table.Column{
		{Title: "Acquired", Width: dayColumnWidth},
		{Title: "Symbol", Width: symbolColumnWidth},
		{Title: "From", Width: grantWidth},
		{Title: fmt.Sprintf("%*s", sharesColumnWidth, "Shares"), Width: sharesColumnWidth},
		{Title: fmt.Sprintf("%*s", priceColumnWidth, "Cost"), Width: priceColumnWidth},
		{Title: fmt.Sprintf("%*s", priceColumnWidth, "Price"), Width: priceColumnWidth},
		{Title: fmt.Sprintf("%*s", gainColumnWidth, "Gain"), Width: gainColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Tax"), Width: valueColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Value"), Width: valueColumnWidth},
	})
}

// Title implements ViewModel by naming the view in the app's tab bar.
func (m *HoldingsModel) Title() string {
	return "Holdings"
}

// CapturesKeys implements KeyCapturer: while a dialog is open the keyboard belongs to it, down to
// the keys that would otherwise switch views. The digits are the shares of the lot being typed.
func (m *HoldingsModel) CapturesKeys() bool {
	return m.input != nil || m.confirm != nil || m.form != nil
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
		m.layout()
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
	case SaveSaleMsg:
		m.form = nil
		return m, m.saveSaleCmd(msg.sale)
	case FormCanceledMsg:
		m.form = nil
		return m, nil
	}

	// Whatever the view does not handle may be an open dialog's, such as its cursor's blink ticks.
	if m.input != nil {
		return m, m.input.Update(msg)
	}
	if m.form != nil {
		return m, m.form.Update(msg)
	}

	return m, nil
}

// handlePortfolioLoaded takes over the portfolio that was loaded and fills the table with its lots,
// in the columns that this portfolio has, since the tax on a lot is only shown at a rate.
// A load that failed leaves the one on display where it is, and its error to the header. Either way
// it sees to it that the quotes are fetched again in a while.
func (m *HoldingsModel) handlePortfolioLoaded(msg PortfolioLoadedMsg) (tea.Model, tea.Cmd) {
	m.err = msg.err
	if msg.err == nil {
		// Quotes that could not be refreshed do not make the portfolio any less the latest there is.
		m.err = msg.quotesErr
		m.portfolio = msg.portfolio
		m.layout()
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

// handleKeyPress quits on the quit keys, opens the dialog for a new lot on the add key and for the
// selected one on the edit key, the form for a sale of its shares on the sell key, asks whether to
// delete the selected lot on the delete key, and hands every other key to the table, which moves
// the selection. While a dialog is open, every key is the dialog's.
func (m *HoldingsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.input != nil {
		return m, m.input.HandleKeyPress(msg)
	}
	if m.confirm != nil {
		return m, m.confirm.HandleKeyPress(msg)
	}
	if m.form != nil {
		return m, m.form.HandleKeyPress(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Delete):
		return m.askToDelete()
	case key.Matches(msg, m.keys.Sell):
		return m.sellSelected()
	case key.Matches(msg, m.keys.Add):
		m.input = NewInputDialog("New lot", lotPlaceholder, m.newLot, dialogWidth, m.style)
		return m, m.input.Init()
	case key.Matches(msg, m.keys.Edit):
		return m.editSelected()
	case key.Matches(msg, m.keys.Details):
		// Without a lot to select there are no details to open, as there is nothing to edit.
		i := m.table.Cursor()
		m.details = !m.details && i >= 0 && i < len(m.lots)
		return m, nil
	case key.Matches(msg, m.keys.Close):
		m.details = false
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// editSelected opens the dialog on the selected lot, with the spec of that lot in its field. What
// the spec says once it is confirmed replaces the lot: it keeps its ID, and its day if the spec no
// longer has one. Without a lot to select, there is nothing to edit.
func (m *HoldingsModel) editSelected() (tea.Model, tea.Cmd) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.lots) {
		return m, nil
	}

	lot := m.lots[i]
	submit := func(spec string) (tea.Msg, error) {
		edited, err := portfolio.NewLot(spec)
		if err != nil {
			return nil, err
		}

		edited.ID = lot.ID
		if edited.Acquired.IsZero() {
			edited.Acquired = lot.Acquired
		}

		return SaveLotMsg{lot: edited}, nil
	}

	m.input = NewInputDialog("Edit lot", lotPlaceholder, submit, dialogWidth, m.style)
	m.input.SetValue(lot.String())

	return m, m.input.Init()
}

// sellSelected opens the form that records a sale of shares of the selected lot: how many, at what
// price, on which day, which is today unless it is changed, and notes. Without a lot to select,
// there is nothing to sell.
func (m *HoldingsModel) sellSelected() (tea.Model, tea.Cmd) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.lots) {
		return m, nil
	}

	lot := m.lots[i]
	today := portfolio.DayOf(m.now())

	// The placeholders are what is most likely typed: all that is left, at the price of today.
	price := "410.20"
	if quote, ok := m.portfolio.Quotes[lot.Symbol]; ok {
		price = quote.Price.StringFixed(2)
	}

	title := fmt.Sprintf("Sell %s of %s", lot.Symbol, lot.Acquired.Format(time.DateOnly))
	fields := []FormField{
		{Label: "Shares", Placeholder: lot.Remaining().String()},
		{Label: "Price", Placeholder: price},
		{Label: "Date", Value: today.Format(time.DateOnly)},
		{Label: "Notes", Lines: saleNoteLines},
	}
	m.form = NewFormDialog(title, fields, newSale(lot, today), saleFormWidth, m.style)

	return m, m.form.Init()
}

// newSale returns what the sale form of lot does with its values, which are the shares, the price,
// the day and the notes, in that order. It turns them into the message that asks for the sale to be
// saved, on today's day if the form has none. The form refuses what the store would, so that it can
// say so while it is still open.
func newSale(lot portfolio.Lot, today time.Time) func(values []string) (tea.Msg, error) {
	return func(values []string) (tea.Msg, error) {
		shares, err := decimal.NewFromString(values[0])
		switch {
		case err != nil:
			return nil, fmt.Errorf("%q is not a number of shares", values[0])
		case !shares.IsPositive():
			return nil, errors.New("a sale must have more than 0 shares")
		case shares.GreaterThan(lot.Remaining()):
			return nil, fmt.Errorf("the lot has %s shares left, not %s", lot.Remaining(), shares)
		}

		// A price is a price with or without the dollar sign in front of it.
		price, err := decimal.NewFromString(strings.TrimPrefix(values[1], "$"))
		if err != nil || !price.IsPositive() {
			return nil, fmt.Errorf("%q is not a price such as 410.20", values[1])
		}

		date := today
		if values[2] != "" {
			date, err = portfolio.ParseDay(values[2])
			if err != nil {
				return nil, err
			}
		}
		if date.Before(lot.Acquired) {
			return nil, fmt.Errorf("the lot was only acquired on %s", lot.Acquired.Format(time.DateOnly))
		}

		sale := portfolio.Sale{LotID: lot.ID, Date: date, Shares: shares, Price: price, Note: values[3]}
		return SaveSaleMsg{sale: sale}, nil
	}
}

// saveSaleCmd returns a command that saves sale and loads the portfolio again, in which the lot of
// the sale has that many shares fewer.
func (m *HoldingsModel) saveSaleCmd(sale portfolio.Sale) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		if err := store.SaveSale(sale); err != nil {
			return PortfolioLoadedMsg{err: err}
		}

		return loadPortfolioCmd(store)()
	}
}

// askToDelete opens a dialog asking whether to delete the selected lot, which sends a DeleteLotMsg
// if the answer is yes. Without a lot to select, there is nothing to ask.
func (m *HoldingsModel) askToDelete() (tea.Model, tea.Cmd) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.lots) {
		return m, nil
	}

	lot := m.lots[i]
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

// updateRows fills the table with a row for every lot of the portfolio that has shares left. A lot
// that was sold to the last share is no holding any more, and is found among the sales.
func (m *HoldingsModel) updateRows() {
	m.lots = nil
	for _, lot := range m.portfolio.Lots {
		if lot.Remaining().IsPositive() {
			m.lots = append(m.lots, lot)
		}
	}

	rows := make([]table.Row, len(m.lots))
	for i, lot := range m.lots {
		rows[i] = shown(m, lotRow(lot, m.portfolio.Quotes[lot.Symbol], m.portfolio.GainsTaxRate))
	}

	setRows(&m.table, rows)
}

// View implements tea.Model by rendering the header above the table of lots, and the keys below it.
func (m *HoldingsModel) View() tea.View {
	header := m.headerView()
	box := m.style.Table.Render(m.table.View())
	frame := header + "\n" + box + "\n" + m.helpView()

	layers := []*lipgloss.Layer{lipgloss.NewLayer(frame)}
	if flyout := m.flyoutLayer(lipgloss.Height(box)); flyout != nil {
		// The flyout takes the right of the table's box, from its top to its bottom, and leaves the
		// left of every row in sight, which is what says whose details these are.
		flyout.X(lipgloss.Width(box) - flyout.Width())
		flyout.Y(lipgloss.Height(header))
		layers = append(layers, flyout)
	}
	if dialog := m.dialogLayer(); dialog != nil {
		// The dialog floats centered in front of everything else.
		dialog.X((lipgloss.Width(frame) - dialog.Width()) / 2)
		dialog.Y((lipgloss.Height(frame) - dialog.Height()) / 2)
		layers = append(layers, dialog)
	}

	c := lipgloss.NewCompositor(layers...)
	return tea.NewView(trimTrailingSpace(c.Render()) + "\n")
}

// flyoutLayer renders the details of the selected lot as a flyout of the given height, or returns
// nil if the flyout is closed or there is no lot to select. It is rendered from the selection every
// time, so it shows another lot as soon as another row is selected.
func (m *HoldingsModel) flyoutLayer(height int) *lipgloss.Layer {
	i := m.table.Cursor()
	if !m.details || i < 0 || i >= len(m.lots) {
		return nil
	}

	lot := m.lots[i]
	return lotDetails(lot, m.portfolio.Quotes[lot.Symbol], m.portfolio.GainsTaxRate, m.style).Layer(flyoutWidth, height, m.style)
}

// dialogLayer renders whichever dialog is open, or returns nil if none is.
func (m *HoldingsModel) dialogLayer() *lipgloss.Layer {
	switch {
	case m.input != nil:
		return m.input.Layer()
	case m.confirm != nil:
		return m.confirm.Layer()
	case m.form != nil:
		return m.form.Layer()
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
	case m.form != nil:
		// In the notes enter starts a new line, so there it is ctrl+s that saves.
		save := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save"))
		if m.form.OnMultiline() {
			save = key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save"))
		}

		return help.ShortHelpView([]key.Binding{
			save,
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}) + "\n"
	}

	// The flyout is no dialog and leaves every key what it was, so all it changes is which key
	// is the one to know about it: the one that opens it, or the one that closes it.
	details := m.keys.Details
	if m.details {
		details = m.keys.Close
	}

	return help.ShortHelpView([]key.Binding{nav.LineUp, nav.LineDown, m.keys.Quit}) + "\n" +
		help.ShortHelpView([]key.Binding{m.keys.Add, m.keys.Edit, m.keys.Sell, m.keys.Delete, details})
}

// positions sums up lots as how many shares of each stock are left of them, the stocks in alphabetical
// order, such as "3 AAPL · 10 PANW". It is what the Holdings view says above its table, where the
// rows only have the shares of one lot each.
func positions(lots []portfolio.Lot) string {
	shares := make(map[string]decimal.Decimal)
	for _, lot := range lots {
		if left := lot.Remaining(); left.IsPositive() {
			shares[lot.Symbol] = shares[lot.Symbol].Add(left)
		}
	}

	if len(shares) == 0 {
		return "No shares held"
	}

	var parts []string
	for _, symbol := range slices.Sorted(maps.Keys(shares)) {
		parts = append(parts, shares[symbol].String()+" "+symbol)
	}

	return strings.Join(parts, " · ")
}

// lotDetails is what the flyout says about a lot, valued at quote, which is the zero Quote if there
// is none of the lot's stock. It has what the row has no room for: the shares the lot was acquired
// with and how many of them were sold, where the row only has those that are left, and what those
// have gained since. What is not known is a dash, as it is in the row, and the gain is known only
// if both the cost and the price are. The value is in the gold of style, as the part of the total
// that it is, and the gain in its green or, if it is a loss, its red. Under the gain is the tax that
// selling the shares at that price would cost at gainsTaxRate, the number the row has in its Tax
// column, which a narrow terminal leaves out. An account without that rate, where gainsTaxRate is
// not valid, has no such row here either.
func lotDetails(lot portfolio.Lot, quote portfolio.Quote, gainsTaxRate decimal.NullDecimal, style Style) Flyout {
	from := noValue
	if lot.Grant != "" {
		from = lot.Grant
	}

	cost, price, valueLeft := noValue, noValue, noValue
	if !lot.Cost.IsZero() {
		cost = portfolio.FormatUSD(lot.Cost)
	}
	if quote.Symbol != "" {
		price = portfolio.FormatUSD(quote.Price)
		valueLeft = portfolio.FormatUSD(lot.Remaining().Mul(quote.Price))
	}

	// Only a value that is known stands out. A dash in gold would be a number that is not there.
	var valueStyle lipgloss.Style
	if quote.Symbol != "" {
		valueStyle = style.Value
	}

	value := []FlyoutRow{
		{Label: "Cost per share", Value: cost},
		{Label: "Price per share", Value: price},
		{Label: "Value of what is left", Value: valueLeft, ValueStyle: valueStyle},
	}
	if !lot.Cost.IsZero() && quote.Symbol != "" {
		gainStyle := style.Gain
		if quote.Price.LessThan(lot.Cost) {
			gainStyle = style.Loss
		}

		// The percentage goes on a row of its own, so that the dollars of the gain end where the
		// dollars of the value above them do.
		dollars, percent := lotGain(lot, quote.Price)
		value = append(value,
			FlyoutRow{Label: "Gain", Value: dollars, ValueStyle: gainStyle},
			FlyoutRow{Value: percent, ValueStyle: gainStyle},
		)
	} else {
		value = append(value, FlyoutRow{Label: "Gain", Value: noValue})
	}

	if gainsTaxRate.Valid {
		value = append(value, FlyoutRow{Label: "Tax on the gain", Value: lotTax(lot, quote, gainsTaxRate.Decimal)})
	}

	return Flyout{
		Title: lot.Symbol + " of " + lot.Acquired.Format(time.DateOnly),
		Sections: []FlyoutSection{
			{Title: "Shares", Rows: []FlyoutRow{
				{Label: "From", Value: from},
				{Label: "Acquired", Value: lot.Shares.String()},
				{Label: "Sold", Value: lot.Sold.String()},
				{Label: "Left", Value: lot.Remaining().String()},
			}},
			{Title: "Value", Rows: value},
		},
	}
}

// lotGain renders what the shares that are left of lot have gained since they were acquired, at the
// given price of one of them: in dollars, such as "$64.52", and in percent of what they cost, such
// as "▲ 4.2%", or "-$5,019.25" and "▼ 42.7%" for a loss. The arrow carries the direction of the
// percentage, as it does for the change of a price in the header. The lot has to have a cost.
func lotGain(lot portfolio.Lot, price decimal.Decimal) (dollars, percent string) {
	gain, _ := lot.Gain(price)
	growth, _ := lot.GrowthPercent(price)

	arrow := "▲"
	if growth.IsNegative() {
		arrow = "▼"
	}

	return portfolio.FormatUSD(gain), arrow + " " + growth.Abs().StringFixed(1) + "%"
}

// lotTax renders the tax that selling the shares that are left of lot at quote would cost, at rate
// in percent, such as "$17.03". It is a dash if the gain is not known, for want of a cost or of a
// quote, which is then the zero Quote. A lot that lost is not taxed, which reads as "$0.00".
func lotTax(lot portfolio.Lot, quote portfolio.Quote, rate decimal.Decimal) string {
	gained, ok := lot.Gain(quote.Price)
	if !ok || quote.Symbol == "" {
		return noValue
	}

	return portfolio.FormatUSD(portfolio.GainsTax(gained, rate))
}

// lotRow renders a lot as a row of the Holdings table with all its columns: the grant it was
// released from, if any, the shares that are left of it, what one of them cost, if that is known,
// and what they are worth at quote, which is the zero Quote if there is none of the lot's stock.
// With both a cost and a quote, it says how far the price is from the cost, and the tax that
// selling the shares at that price would cost at gainsTaxRate, which is not valid if the account
// has none. The value comes last, where the header's values end, with the tax left of it. The
// numbers are right-aligned for their digits to line up down the column. The table has no
// alignment of its own, so the values are padded out here.
func lotRow(lot portfolio.Lot, quote portfolio.Quote, gainsTaxRate decimal.NullDecimal) table.Row {
	cost, price, gain, value, tax := noValue, noValue, noValue, noValue, noValue
	if !lot.Cost.IsZero() {
		cost = portfolio.FormatUSD(lot.Cost)
	}
	if quote.Symbol != "" {
		price = portfolio.FormatUSD(quote.Price)
		value = portfolio.FormatUSD(lot.Remaining().Mul(quote.Price))
		if growth, ok := lot.GrowthPercent(quote.Price); ok {
			gain = formatGrowth(growth)
		}
		if gainsTaxRate.Valid {
			tax = lotTax(lot, quote, gainsTaxRate.Decimal)
		}
	}

	return table.Row{
		lot.Acquired.Format(time.DateOnly),
		lot.Symbol,
		lot.Grant,
		fmt.Sprintf("%*s", sharesColumnWidth, lot.Remaining()),
		fmt.Sprintf("%*s", priceColumnWidth, cost),
		fmt.Sprintf("%*s", priceColumnWidth, price),
		fmt.Sprintf("%*s", gainColumnWidth, gain),
		fmt.Sprintf("%*s", valueColumnWidth, tax),
		fmt.Sprintf("%*s", valueColumnWidth, value),
	}
}

// formatGrowth renders a growth in percent to one decimal and with its sign, such as "+123.4%" or
// "-12.5%". The sign is there for a gain as well, so that the column reads as changes rather than
// as shares of something.
func formatGrowth(growth decimal.Decimal) string {
	// Rounded before the sign is read, so that next to nothing lost is not "-0.0%".
	growth = growth.Round(1)

	sign := "+"
	if growth.IsNegative() {
		sign = "-"
	}

	return sign + growth.Abs().StringFixed(1) + "%"
}
