package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// symbolColumnWidth fits a symbol with an exchange behind it, such as "SAP.DE".
const symbolColumnWidth = 8

// grantsColumnsWidth is what every column of the Grants table other than the name occupies, padding
// included.
const grantsColumnsWidth = symbolColumnWidth + 2*sharesColumnWidth + valueColumnWidth + 4*cellPadding

// A GrantsModel is the Grants view: every grant of the account, with how many of its shares are
// still to come and what they are worth. It is where grants are added and deleted.
type GrantsModel struct {
	store     Store
	style     Style
	keys      keyMap
	table     table.Model
	width     int            // width of the table, which the header line is spread across
	portfolio Portfolio      // the account as it was last loaded
	err       error          // why the last load failed or had no fresh quotes, nil unless it did
	input     *InputDialog   // the dialog that adds a grant, nil unless it is open
	confirm   *ConfirmDialog // the dialog asking whether to delete a grant, nil unless it is open
}

// DeleteGrantMsg reports that the user confirmed deleting the grant with the given ID.
type DeleteGrantMsg struct {
	id int
}

// SaveGrantMsg reports that the dialog was confirmed with the spec of a grant. The grant has not
// been saved yet; that is up to whoever receives it.
type SaveGrantMsg struct {
	grant portfolio.Grant
}

// NewGrantsModel returns the Grants view over store, rendered in style.
func NewGrantsModel(store Store, style Style) *GrantsModel {
	m := &GrantsModel{
		store: store,
		style: style,
		keys:  defaultKeyMap(),
		// A real width arrives with the first WindowSizeMsg, as it does for the other views.
		width: minTableWidth,
	}
	m.table = newTable(style, m.columns())

	return m
}

// columns returns the table's columns, the name column taking whatever the width leaves. The titles
// of the number columns are padded like the numbers under them, so that they end where the numbers
// do.
func (m *GrantsModel) columns() []table.Column {
	return []table.Column{
		{Title: "Grant", Width: m.width - grantsColumnsWidth - cellPadding},
		{Title: "Symbol", Width: symbolColumnWidth},
		{Title: fmt.Sprintf("%*s", sharesColumnWidth, "To come"), Width: sharesColumnWidth},
		{Title: fmt.Sprintf("%*s", sharesColumnWidth, "Granted"), Width: sharesColumnWidth},
		{Title: fmt.Sprintf("%*s", valueColumnWidth, "Value"), Width: valueColumnWidth},
	}
}

// Title implements ViewModel by naming the view in the app's tab bar.
func (m *GrantsModel) Title() string {
	return "Grants"
}

// CapturesKeys implements KeyCapturer: while a dialog is open the keyboard belongs to it, down to
// the keys that would otherwise switch views. The digits are part of the spec being typed.
func (m *GrantsModel) CapturesKeys() bool {
	return m.input != nil || m.confirm != nil
}

// Init implements tea.Model by loading the portfolio.
func (m *GrantsModel) Init() tea.Cmd {
	return loadPortfolioCmd(m.store)
}

// Update implements tea.Model by fitting the table to the window, filling it with the grants of the
// portfolio once it is loaded, and moving the selection through it.
func (m *GrantsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case SaveGrantMsg:
		m.input = nil
		return m, m.saveGrantCmd(msg.grant)
	case InputCanceledMsg:
		m.input = nil
		return m, nil
	case DeleteGrantMsg:
		m.confirm = nil
		return m, m.deleteGrantCmd(msg.id)
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

// handleKeyPress quits on the quit keys, opens the dialog for a new grant on the add key, asks
// whether to delete the selected grant on the delete key, and hands every other key to the table,
// which moves the selection. While a dialog is open, every key is the dialog's.
func (m *GrantsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		m.input = NewInputDialog("New grant", "Name: 10 PANW monthly x24 from 2026-01-15", newGrant, dialogWidth, m.style)
		return m, m.input.Init()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// askToDelete opens a dialog asking whether to delete the selected grant, which sends a
// DeleteGrantMsg if the answer is yes. The question names what goes with the grant: its vests that
// are still to come, which leave the potential value. The lots of the vests that were released stay.
// Without a grant to select, there is nothing to ask.
func (m *GrantsModel) askToDelete() (tea.Model, tea.Cmd) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.portfolio.Grants) {
		return m, nil
	}

	grant := m.portfolio.Grants[i]
	unreleased := 0
	for _, vest := range grant.Vests {
		if !vest.Released {
			unreleased++
		}
	}

	question := fmt.Sprintf("Delete %q and its %s still to come?", grant.Name, count(unreleased, "vest"))
	m.confirm = NewConfirmDialog("Delete grant", question, DeleteGrantMsg{id: grant.ID}, m.style)

	return m, nil
}

// deleteGrantCmd returns a command that deletes the grant with the given ID and loads the portfolio
// again, which then no longer has it.
func (m *GrantsModel) deleteGrantCmd(id int) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		if err := store.DeleteGrant(id); err != nil {
			return PortfolioLoadedMsg{err: err}
		}

		return loadPortfolioCmd(store)()
	}
}

// newGrant turns the spec typed into the dialog into the message that asks for the grant to be
// saved.
func newGrant(spec string) (tea.Msg, error) {
	grant, err := portfolio.NewGrant(spec)
	if err != nil {
		return nil, err
	}

	return SaveGrantMsg{grant: grant}, nil
}

// saveGrantCmd returns a command that saves grant and loads the portfolio again, which then has it.
func (m *GrantsModel) saveGrantCmd(grant portfolio.Grant) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		if err := store.SaveGrant(grant); err != nil {
			return PortfolioLoadedMsg{err: err}
		}

		return loadPortfolioCmd(store)()
	}
}

// updateRows fills the table with a row for every grant of the portfolio.
func (m *GrantsModel) updateRows() {
	rows := make([]table.Row, len(m.portfolio.Grants))
	for i, grant := range m.portfolio.Grants {
		rows[i] = grantRow(grant, m.portfolio.Quotes[grant.Symbol])
	}

	m.table.SetRows(rows)
}

// View implements tea.Model by rendering the header above the table of grants, and the keys below
// it.
func (m *GrantsModel) View() tea.View {
	frame := m.headerView() + "\n" +
		m.style.Table.Render(m.table.View()) + "\n" +
		m.helpView()

	layers := []*lipgloss.Layer{lipgloss.NewLayer(frame)}
	if dialog := m.dialogLayer(); dialog != nil {
		// The dialog floats centered in front of the table, as those of the other views do.
		dialog.X((lipgloss.Width(frame) - dialog.Width()) / 2)
		dialog.Y((lipgloss.Height(frame) - dialog.Height()) / 2)
		layers = append(layers, dialog)
	}

	c := lipgloss.NewCompositor(layers...)
	return tea.NewView(trimTrailingSpace(c.Render()) + "\n")
}

// dialogLayer renders whichever dialog is open, or returns nil if none is.
func (m *GrantsModel) dialogLayer() *lipgloss.Layer {
	switch {
	case m.input != nil:
		return m.input.Layer()
	case m.confirm != nil:
		return m.confirm.Layer()
	default:
		return nil
	}
}

// headerView renders the two lines above the table: how many grants there are on the left, with the
// prices underneath, and the account values on the right.
func (m *GrantsModel) headerView() string {
	summary := "No grants"
	if n := len(m.portfolio.Grants); n > 0 {
		summary = count(n, "grant")
	}

	return portfolioHeader(summary, m.portfolio, m.err, m.width-cellPadding, m.style)
}

// helpView renders the keys worth knowing in two lines, as every view does: getting around on the
// first, and what can be done to the table on the second. While a dialog is open it shows the
// dialog's keys instead, on one line, and leaves the second empty so that the frame keeps its
// height.
func (m *GrantsModel) helpView() string {
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

// grantRow renders a grant as a row of the Grants table: how many of its shares are still to come,
// how many it had in all, and what the ones to come are worth at quote, which is the zero Quote if
// there is none of the grant's stock. The numbers are right-aligned and padded out like those of the
// other tables.
func grantRow(grant portfolio.Grant, quote portfolio.Quote) table.Row {
	var unreleased, granted decimal.Decimal
	for _, vest := range grant.Vests {
		granted = granted.Add(vest.Shares)
		if !vest.Released {
			unreleased = unreleased.Add(vest.Shares)
		}
	}

	value := noValue
	if quote.Symbol != "" {
		value = portfolio.FormatUSD(unreleased.Mul(quote.Price))
	}

	return table.Row{
		grant.Name,
		grant.Symbol,
		fmt.Sprintf("%*s", sharesColumnWidth, unreleased),
		fmt.Sprintf("%*s", sharesColumnWidth, granted),
		fmt.Sprintf("%*s", valueColumnWidth, value),
	}
}
