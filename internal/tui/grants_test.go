package tui

import (
	"errors"
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// newTestingGrants returns a Grants view that has loaded the test portfolio into a window of 100 by
// 20, and the store it loaded it from.
func newTestingGrants(t *testing.T) (*GrantsModel, *MockStore) {
	t.Helper()

	store := new(MockStore)
	store.returns(testPortfolio())

	m := NewGrantsModel(store, DefaultStyle())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))

	return m, store
}

// TestGrantsModel_LoadsPortfolio covers what the Grants view starts from: once started, it loads the
// portfolio and shows a row for every grant, valued at the quote the store has of its stock.
func TestGrantsModel_LoadsPortfolio(t *testing.T) {
	m, store := newTestingGrants(t)
	assert.Equal(t, "Grants", m.Title())

	p := testPortfolio()
	rows := m.table.Rows()
	require.Len(t, rows, 1)
	assert.Equal(t, grantRow(p.Grants[0], p.Quotes["PANW"]), rows[0])
	store.AssertExpectations(t)
}

// TestGrantsModel_Keys covers the keys the view answers to before it can change anything: q and
// ctrl+c quit, and every other key is the table's.
func TestGrantsModel_Keys(t *testing.T) {
	m, _ := newTestingGrants(t)

	_, cmd := m.Update(keyPressed("q"))
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))

	_, cmd = m.Update(keyPressed("j"))
	assert.Nil(t, cmd)
}

// TestGrantsModel_Render is the frame the Grants view puts on screen, in the shape of the other
// views': how many grants there are above the table next to the account values, a row for every
// grant, and the keys underneath.
func TestGrantsModel_Render(t *testing.T) {
	m, _ := newTestingGrants(t)

	golden.RequireEqual(t, ansi.Strip(m.View().Content))
}

// TestGrantsModel_ShowsErrors covers a load that failed: the view says why where the prices
// otherwise are and keeps the grants it had on display, as the other views do.
func TestGrantsModel_ShowsErrors(t *testing.T) {
	m, _ := newTestingGrants(t)

	m.Update(PortfolioLoadedMsg{err: errors.New("database is locked")})

	assert.Contains(t, ansi.Strip(m.View().Content), "database is locked")
	assert.Len(t, m.table.Rows(), 1, "the grants should stay on display")

	m.Update(PortfolioLoadedMsg{portfolio: testPortfolio(), quotesErr: errors.New("no quote of PANW")})

	assert.Contains(t, ansi.Strip(m.View().Content), "no quote of PANW")
}

// TestGrantsModel_AddGrant covers recording a grant in the view: a opens a dialog that takes the
// keyboard, the spec typed into it becomes a grant on enter, and the view saves that grant and loads
// the portfolio again to show it. A spec that does not parse keeps the dialog open, with the reason
// under its field.
func TestGrantsModel_AddGrant(t *testing.T) {
	m, store := newTestingGrants(t)
	assert.False(t, m.CapturesKeys())

	m.Update(keyPressed("a"))
	require.True(t, m.CapturesKeys(), "the dialog should take the keyboard")

	for _, key := range keysPressed("RSU: 400 PANW quarterly 10/20/30 from 2026-02-20") {
		m.Update(key)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Contains(t, ansi.Strip(m.View().Content), "the percentages 10/20/30 add up to 60, not 100")

	const spec = "RSU: 400 PANW quarterly 10/20/30/40 from 2026-02-20"
	m.input.SetValue(spec)
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	grant, err := portfolio.NewGrant(spec)
	require.NoError(t, err)
	msg := runCmd(t, cmd)
	require.Equal(t, SaveGrantMsg{grant: grant}, msg)

	store.On("SaveGrant", grant).Return(nil)
	_, cmd = m.Update(msg)
	assert.False(t, m.CapturesKeys(), "the dialog should be closed")
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)
}

// TestGrantsModel_CancelAddGrant covers leaving the dialog with esc: it closes, the keys are the
// view's again, and nothing is saved.
func TestGrantsModel_CancelAddGrant(t *testing.T) {
	m, store := newTestingGrants(t)
	m.Update(keyPressed("a"))

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_, cmd = m.Update(runCmd(t, cmd))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "the dialog should be closed")
	store.AssertNotCalled(t, "SaveGrant", mock.Anything)
}

// TestGrantsModel_RenderGrantDialog is the frame while a grant is being typed: the dialog in front
// of the table, wide enough for a spec, and its keys in the help lines.
func TestGrantsModel_RenderGrantDialog(t *testing.T) {
	m, _ := newTestingGrants(t)
	m.Update(keyPressed("a"))
	m.input.SetValue("RSU: 400 PANW quarterly 10/20/30/40 from 2026-02-20")

	golden.RequireEqual(t, ansi.Strip(m.View().Content))
}

// TestGrantsModel_DeleteGrant covers taking a grant back: d asks before it deletes the selected
// grant, naming it and what goes with it, and only a y has the store delete it, after which the view
// loads the portfolio again. A no closes the question and deletes nothing.
func TestGrantsModel_DeleteGrant(t *testing.T) {
	m, store := newTestingGrants(t)

	m.Update(keyPressed("d"))
	require.True(t, m.CapturesKeys(), "the question should take the keyboard")
	assert.Contains(t, ansi.Strip(m.View().Content), `Delete "Payout" and its 2 vests still to come?`)

	_, cmd := m.Update(keyPressed("n"))
	_, cmd = m.Update(runCmd(t, cmd))
	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "a no should close the question")

	m.Update(keyPressed("d"))
	_, cmd = m.Update(keyPressed("y"))
	msg := runCmd(t, cmd)
	require.Equal(t, DeleteGrantMsg{id: 1}, msg)

	store.On("DeleteGrant", 1).Return(nil)
	_, cmd = m.Update(msg)
	assert.False(t, m.CapturesKeys(), "the question should be closed")
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)
}

// TestGrantsModel_DeleteWithNothingSelected covers d in an account without grants, where there is
// nothing to ask about.
func TestGrantsModel_DeleteWithNothingSelected(t *testing.T) {
	store := new(MockStore)
	store.returns(Portfolio{})
	m := NewGrantsModel(store, DefaultStyle())
	m.Update(runCmd(t, m.Init()))

	_, cmd := m.Update(keyPressed("d"))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "there should be no question to answer")
}

// TestGrantsModel_RenderDeleteDialog is the frame while the view asks whether to delete a grant: the
// question in front of the table, and the keys that answer it in the help lines.
func TestGrantsModel_RenderDeleteDialog(t *testing.T) {
	m, _ := newTestingGrants(t)
	m.Update(keyPressed("d"))

	golden.RequireEqual(t, ansi.Strip(m.View().Content))
}

// TestGrantRow covers how one grant reads as a row of the Grants table: its name and stock, how many
// of its shares are still to come and how many it had in all, and what the ones to come are worth,
// which is the grant's part of the potential value. The numbers are right-aligned like those of the
// other tables.
func TestGrantRow(t *testing.T) {
	panw := portfolio.Quote{Symbol: "PANW", Price: dec("396.25")}

	cases := map[string]struct {
		grant    portfolio.Grant
		quote    portfolio.Quote
		expected table.Row
	}{
		"a grant with a released vest": {
			grant: portfolio.Grant{
				Name:   "Payout",
				Symbol: "PANW",
				Vests: []portfolio.Vest{
					{Shares: dec("10"), Released: true},
					{Shares: dec("10")},
					{Shares: dec("10")},
				},
			},
			quote:    panw,
			expected: table.Row{"Payout", "PANW", "        20", "        30", "     $7,925.00"},
		},
		"a grant that is all released": {
			grant: portfolio.Grant{
				Name:   "Sign-on",
				Symbol: "PANW",
				Vests:  []portfolio.Vest{{Shares: dec("2.5"), Released: true}},
			},
			quote:    panw,
			expected: table.Row{"Sign-on", "PANW", "         0", "       2.5", "         $0.00"},
		},
		"a grant without a quote": {
			grant: portfolio.Grant{
				Name:   "Old plan",
				Symbol: "SAP.DE",
				Vests:  []portfolio.Vest{{Shares: dec("5")}},
			},
			quote:    portfolio.Quote{},
			expected: table.Row{"Old plan", "SAP.DE", "         5", "         5", "             -"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, grantRow(c.grant, c.quote))
		})
	}
}
