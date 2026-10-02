package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/fgrosse/folio/internal/portfolio"
)

// dueColumnWidth fits "in 23 months", the longest the last column of the Vesting table gets.
const dueColumnWidth = 12

// vestingColumnsWidth is what every column of the Vesting table other than the grant occupies,
// padding included.
const vestingColumnsWidth = dayColumnWidth + sharesColumnWidth + valueColumnWidth + dueColumnWidth + 4*cellPadding

// A VestingModel is the Vesting view: every vest that has not been released, across all grants and
// in the order of their days, with what it is worth at the latest price. Those vests are what the
// potential value counts.
type VestingModel struct {
	store     Store
	style     Style
	keys      keyMap
	table     table.Model
	width     int              // width of the table, which the header line is spread across
	now       func() time.Time // the clock that says what today is
	portfolio Portfolio        // the account as it was last loaded
	vests     []grantVest      // the vests on display, as the table shows them
}

// NewVestingModel returns the Vesting view over store, rendered in style.
func NewVestingModel(store Store, style Style) *VestingModel {
	m := &VestingModel{
		store: store,
		style: style,
		keys:  defaultKeyMap(),
		now:   time.Now,
		// A real width arrives with the first WindowSizeMsg, as it does for the Holdings view.
		width: minTableWidth,
	}
	m.table = newTable(style, m.columns())

	return m
}

// columns returns the table's columns, the grant column taking whatever the width leaves. The
// titles of the number columns are padded like the numbers under them, so that they end where the
// numbers do.
func (m *VestingModel) columns() []table.Column {
	return []table.Column{
		{Title: "Vests on", Width: dayColumnWidth},
		{Title: "Grant", Width: m.width - vestingColumnsWidth - cellPadding},
		{Title: fmt.Sprintf("%*s", sharesColumnWidth, "Shares"), Width: sharesColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Value"), Width: valueColumnWidth},
		{Title: "Due", Width: dueColumnWidth},
	}
}

// Title implements ViewModel by naming the view in the app's tab bar.
func (m *VestingModel) Title() string {
	return "Vesting"
}

// Init implements tea.Model by loading the portfolio.
func (m *VestingModel) Init() tea.Cmd {
	return loadPortfolioCmd(m.store)
}

// Update implements tea.Model by fitting the table to the window, filling it with the vests of the
// portfolio once it is loaded, and moving the selection through it.
func (m *VestingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func (m *VestingModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// updateRows fills the table with a row for every vest of the portfolio that has not been released.
func (m *VestingModel) updateRows() {
	today := portfolio.DayOf(m.now())
	m.vests = unreleasedVests(m.portfolio.Grants)

	rows := make([]table.Row, len(m.vests))
	for i, vest := range m.vests {
		rows[i] = vestRow(vest, m.portfolio.Quotes[vest.symbol], today)
	}

	m.table.SetRows(rows)
}

// View implements tea.Model by rendering the table of vests.
func (m *VestingModel) View() tea.View {
	return tea.NewView(m.style.Table.Render(m.table.View()))
}

// A grantVest is a vest together with what the Vesting view needs to know of the grant it belongs
// to: the name to list it under, and the symbol of the stock to value it at.
type grantVest struct {
	grant  string
	symbol string
	vest   portfolio.Vest
}

// unreleasedVests returns the vests of grants that have not been released, which are the ones the
// potential value counts, in the order of their days. Vests of the same day stay in the order of
// their grants.
func unreleasedVests(grants []portfolio.Grant) []grantVest {
	var vests []grantVest
	for _, grant := range grants {
		for _, vest := range grant.Vests {
			if !vest.Released {
				vests = append(vests, grantVest{grant: grant.Name, symbol: grant.Symbol, vest: vest})
			}
		}
	}

	slices.SortStableFunc(vests, func(a, b grantVest) int {
		return a.vest.Date.Compare(b.vest.Date)
	})

	return vests
}

// vestingSummary sums up vests, which are in the order of their days, as how many of them are due
// and pending release and how many are still to come, with the day of the next. It is what the
// Vesting view says above its table.
func vestingSummary(vests []grantVest, today time.Time) string {
	// The vests are sorted, so the pending ones come first.
	pending := 0
	for pending < len(vests) && !vests[pending].vest.Date.After(today) {
		pending++
	}

	var parts []string
	if pending > 0 {
		parts = append(parts, count(pending, "vest")+" pending release")
	}
	if coming := vests[pending:]; len(coming) > 0 {
		next := coming[0].vest.Date.Format(time.DateOnly)
		parts = append(parts, count(len(coming), "vest")+" to come, the next on "+next)
	}

	if len(parts) == 0 {
		return "Nothing left to vest"
	}

	return strings.Join(parts, " · ")
}

// vestRow renders a vest as a row of the Vesting table, valued at quote, which is the zero Quote if
// there is none of the grant's stock. The numbers are right-aligned and padded out like those of
// the Holdings table.
func vestRow(v grantVest, quote portfolio.Quote, today time.Time) table.Row {
	value := noValue
	if quote.Symbol != "" {
		value = portfolio.FormatUSD(v.vest.Shares.Mul(quote.Price))
	}

	return table.Row{
		v.vest.Date.Format(time.DateOnly),
		v.grant,
		fmt.Sprintf("%*s", sharesColumnWidth, v.vest.Shares),
		fmt.Sprintf("%*s", valueColumnWidth, value),
		dueIn(v.vest.Date, today),
	}
}

// dueIn says how far off the day of a vest is from today, such as "in 44 days". The nearer the day,
// the finer the unit: days for less than sixty of them, then whole months, and whole years from two
// years on. A day that has come reads "pending": the vest is due and has not been released.
func dueIn(date, today time.Time) string {
	if !date.After(today) {
		return "pending"
	}

	days := int(date.Sub(today).Hours() / 24)
	if days < 60 {
		return plural(days, "day")
	}

	months := (date.Year()-today.Year())*12 + int(date.Month()-today.Month())
	if date.Day() < today.Day() {
		months-- // the last of them is not over yet
	}

	if months < 24 {
		return plural(months, "month")
	}

	return plural(months/12, "year")
}

// plural renders "in n units", with the unit in the singular for one of them.
func plural(n int, unit string) string {
	return "in " + count(n, unit)
}

// count renders "n units", with the unit in the singular for one of them.
func count(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}

	return fmt.Sprintf("%d %ss", n, unit)
}
