package tui

import (
	"errors"
	"testing"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestUnreleasedVests covers which vests the Vesting view lists: those of every grant that have not
// been released, which are the ones the potential value counts, in the order of their days whatever
// grant they belong to. Vests of the same day stay in the order of their grants.
func TestUnreleasedVests(t *testing.T) {
	grants := []portfolio.Grant{
		{
			Name:   "RSU 2025",
			Symbol: "PANW",
			Vests: []portfolio.Vest{
				{ID: 1, Date: day("2026-08-20"), Shares: dec("10"), Released: true},
				{ID: 2, Date: day("2026-11-20"), Shares: dec("10")},
				{ID: 3, Date: day("2027-02-20"), Shares: dec("20")},
			},
		},
		{
			Name:   "Payout",
			Symbol: "AAPL",
			Vests: []portfolio.Vest{
				{ID: 4, Date: day("2026-09-15"), Shares: dec("5"), Released: true},
				{ID: 5, Date: day("2026-10-15"), Shares: dec("5")},
				{ID: 6, Date: day("2026-11-20"), Shares: dec("5")},
			},
		},
	}

	expected := []grantVest{
		{grant: "Payout", symbol: "AAPL", vest: grants[1].Vests[1]},
		{grant: "RSU 2025", symbol: "PANW", vest: grants[0].Vests[1]},
		{grant: "Payout", symbol: "AAPL", vest: grants[1].Vests[2]},
		{grant: "RSU 2025", symbol: "PANW", vest: grants[0].Vests[2]},
	}
	assert.Equal(t, expected, unreleasedVests(grants))
}

// newTestingVesting returns a Vesting view that has loaded the test portfolio into a window of 100
// by 20 on the 2nd of October 2026, and the store it loaded it from.
func newTestingVesting(t *testing.T) (*VestingModel, *MockStore) {
	t.Helper()

	store := new(MockStore)
	store.returns(testPortfolio())

	m := NewVestingModel(store, DefaultStyle())
	m.now = func() time.Time { return time.Date(2026, time.October, 2, 14, 30, 0, 0, time.UTC) }
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(runCmd(t, m.Init()))

	return m, store
}

// TestVestingModel_LoadsPortfolio covers what the Vesting view starts from: once started, it loads
// the portfolio and shows a row for every vest that has not been released, valued at the quote the
// store has of its grant's stock.
func TestVestingModel_LoadsPortfolio(t *testing.T) {
	m, store := newTestingVesting(t)
	assert.Equal(t, "Vesting", m.Title())

	p := testPortfolio()
	today := day("2026-10-02")
	vests := unreleasedVests(p.Grants)
	require.Len(t, vests, 2)

	rows := m.table.Rows()
	require.Len(t, rows, 2)
	assert.Equal(t, vestRow(vests[0], p.Quotes["PANW"], today), rows[0])
	assert.Equal(t, vestRow(vests[1], p.Quotes["PANW"], today), rows[1])
	store.AssertExpectations(t)
}

// TestVestingModel_Keys covers the keys the view answers to before it can change anything: q and
// ctrl+c quit, and every other key is the table's, which moves the selection on the ones it knows.
func TestVestingModel_Keys(t *testing.T) {
	m, _ := newTestingVesting(t)

	_, cmd := m.Update(keyPressed("q"))
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.Equal(t, tea.QuitMsg{}, runCmd(t, cmd))

	require.Equal(t, 0, m.table.Cursor())
	_, cmd = m.Update(keyPressed("j"))
	assert.Nil(t, cmd)
	assert.Equal(t, 1, m.table.Cursor(), "j should move the selection down a row")
}

// TestVestingModel_Render is the frame the Vesting view puts on screen, in the shape of the Holdings
// view's: what is due and what is to come above the table next to the account values, a row for
// every vest that has not been released, and the keys underneath.
func TestVestingModel_Render(t *testing.T) {
	m, _ := newTestingVesting(t)

	golden.RequireEqual(t, ansi.Strip(m.View().Content))
}

// TestVestingModel_ShowsErrors covers a load that failed: the view says why where the prices
// otherwise are and keeps the vests it had on display, as the Holdings view does with its lots.
func TestVestingModel_ShowsErrors(t *testing.T) {
	m, _ := newTestingVesting(t)

	m.Update(PortfolioLoadedMsg{err: errors.New("database is locked")})

	assert.Contains(t, ansi.Strip(m.View().Content), "database is locked")
	assert.Len(t, m.table.Rows(), 2, "the vests should stay on display")

	m.Update(PortfolioLoadedMsg{portfolio: testPortfolio(), quotesErr: errors.New("no quote of PANW")})

	assert.Contains(t, ansi.Strip(m.View().Content), "no quote of PANW")
}

// TestVestingModel_ReleaseVest covers what there is to do in the view: r on a vest that is due opens
// a dialog that asks how many shares arrived, with all of the vest's in its field, since that is the
// usual answer, and what a share was worth that day, after an "@". Enter has the store release the
// vest with that many shares at that cost, which makes a lot of them, and the view loads the
// portfolio again.
func TestVestingModel_ReleaseVest(t *testing.T) {
	m, store := newTestingVesting(t)
	// A few days after the first of the two vests, which is pending release by then.
	m.now = func() time.Time { return time.Date(2026, time.November, 20, 9, 0, 0, 0, time.UTC) }

	m.Update(keyPressed("r"))
	require.True(t, m.CapturesKeys(), "the dialog should take the keyboard")
	assert.Equal(t, "10", m.input.Value())

	// Four of the ten shares were withheld for tax.
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	for _, key := range keysPressed("6 @380.12") {
		m.Update(key)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	msg := runCmd(t, cmd)
	require.Equal(t, ReleaseVestMsg{id: 2, shares: dec("6"), cost: dec("380.12")}, msg)

	store.On("ReleaseVest", 2, dec("6"), dec("380.12")).Return(nil)
	_, cmd = m.Update(msg)
	assert.False(t, m.CapturesKeys(), "the dialog should be closed")
	assert.IsType(t, PortfolioLoadedMsg{}, runCmd(t, cmd))
	store.AssertExpectations(t)
}

// TestVestingModel_ReleaseWithoutCost covers a release that only says how many shares arrived, which
// is as much as is known when the confirmation of the vest is not at hand: the vest is released at
// no cost, which the store takes for one that is not known.
func TestVestingModel_ReleaseWithoutCost(t *testing.T) {
	m := newTestingRelease(t)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	msg, ok := runCmd(t, cmd).(ReleaseVestMsg)
	require.True(t, ok)
	assert.Equal(t, "10", msg.shares.String())
	assert.True(t, msg.cost.IsZero(), "a release without a cost should have none")
}

// TestVestingModel_ReleaseRefused covers r where there is nothing to release: on a vest whose day
// has not come, whose shares have not arrived whatever the user types, and in a view without vests.
// No dialog opens, and for a vest that is not due the header says why.
func TestVestingModel_ReleaseRefused(t *testing.T) {
	m, _ := newTestingVesting(t) // on the 2nd of October, before either vest

	_, cmd := m.Update(keyPressed("r"))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "there should be no dialog for a vest that is not due")
	assert.Contains(t, ansi.Strip(m.View().Content), "the vest of 2026-11-15 is not due yet")

	store := new(MockStore)
	store.returns(Portfolio{})
	empty := NewVestingModel(store, DefaultStyle())
	empty.Update(runCmd(t, empty.Init()))

	_, cmd = empty.Update(keyPressed("r"))

	assert.Nil(t, cmd)
	assert.False(t, empty.CapturesKeys(), "there should be no dialog without a vest")
}

// newTestingRelease returns a Vesting view with the release dialog open on the first vest of the
// test portfolio, ten shares that were due five days ago.
func newTestingRelease(t *testing.T) *VestingModel {
	t.Helper()

	m, _ := newTestingVesting(t)
	m.now = func() time.Time { return time.Date(2026, time.November, 20, 9, 0, 0, 0, time.UTC) }
	m.Update(keyPressed("r"))
	require.True(t, m.CapturesKeys())

	return m
}

// TestVestingModel_ReleaseDialogRefuses covers what the release dialog does not take for an answer:
// something that is no number, no shares at all, and more shares than vested. It stays open and says
// why under its field, rather than leaving it to the store to refuse once the dialog is gone.
func TestVestingModel_ReleaseDialogRefuses(t *testing.T) {
	tests := map[string]struct {
		typed string
		error string
	}{
		"not a number":      {typed: "six", error: `"six" is not a number of shares`},
		"no shares":         {typed: "0", error: "a vest must release more than 0 shares"},
		"more than vested":  {typed: "12", error: "the vest has 10 shares, not 12"},
		"a cost without @":  {typed: "6 380.12", error: `"380.12" is not a cost such as @380.12`},
		"a cost of nothing": {typed: "6 @0", error: `"@0" is not a cost such as @380.12`},
		"too much":          {typed: "6 @380.12 net", error: `a release is written as "<shares> [@<cost>]"`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m := newTestingRelease(t)
			m.input.SetValue(tt.typed)

			_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			assert.Nil(t, cmd)
			assert.True(t, m.CapturesKeys(), "the dialog should stay open")
			assert.Contains(t, ansi.Strip(m.View().Content), tt.error)
		})
	}
}

// TestVestingModel_CancelRelease covers leaving the release dialog with esc: it closes, the keys are
// the view's again, and nothing is released.
func TestVestingModel_CancelRelease(t *testing.T) {
	m := newTestingRelease(t)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_, cmd = m.Update(runCmd(t, cmd))

	assert.Nil(t, cmd)
	assert.False(t, m.CapturesKeys(), "the dialog should be closed")
}

// TestVestingModel_RenderReleaseDialog is the frame while a vest is being released: the dialog in
// front of the table with the vest's shares in its field, and its keys in the help lines.
func TestVestingModel_RenderReleaseDialog(t *testing.T) {
	m := newTestingRelease(t)

	golden.RequireEqual(t, ansi.Strip(m.View().Content))
}

// TestVestRow covers how one vest reads as a row of the Vesting table: its day and the grant it
// belongs to, how many shares vest and what they are worth at the latest price, right-aligned like
// the numbers of the Holdings table, and how far off the day is.
func TestVestRow(t *testing.T) {
	today := day("2026-10-02")
	panw := portfolio.Quote{Symbol: "PANW", Price: dec("396.25")}

	tests := map[string]struct {
		vest     grantVest
		quote    portfolio.Quote
		expected table.Row
	}{
		"a vest to come": {
			vest:     grantVest{grant: "Payout", symbol: "PANW", vest: portfolio.Vest{Date: day("2026-11-15"), Shares: dec("10")}},
			quote:    panw,
			expected: table.Row{"2026-11-15", "Payout", "        10", "     $3,962.50", "in 44 days"},
		},
		"a vest pending release": {
			vest:     grantVest{grant: "RSU 2025", symbol: "PANW", vest: portfolio.Vest{Date: day("2026-08-20"), Shares: dec("2.5")}},
			quote:    panw,
			expected: table.Row{"2026-08-20", "RSU 2025", "       2.5", "       $990.63", "pending"},
		},
		"a vest without a quote": {
			vest:     grantVest{grant: "Old plan", symbol: "SAP.DE", vest: portfolio.Vest{Date: day("2027-01-01"), Shares: dec("5")}},
			quote:    portfolio.Quote{},
			expected: table.Row{"2027-01-01", "Old plan", "         5", "             -", "in 2 months"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, vestRow(tt.vest, tt.quote, today))
		})
	}
}

// TestVestingSummary covers the line the Vesting view puts above its table: how many vests are due
// and waiting to be released, which is what there is to do in the view, and how many are still to
// come and when the next of them is.
func TestVestingSummary(t *testing.T) {
	today := day("2026-10-02")
	vest := func(date string) grantVest {
		return grantVest{grant: "Payout", symbol: "PANW", vest: portfolio.Vest{Date: day(date), Shares: dec("10")}}
	}

	tests := map[string]struct {
		vests    []grantVest
		expected string
	}{
		"no vests": {
			vests:    nil,
			expected: "Nothing left to vest",
		},
		"vests to come": {
			vests:    []grantVest{vest("2026-11-15"), vest("2026-12-15"), vest("2027-01-15")},
			expected: "3 vests to come, the next on 2026-11-15",
		},
		"a single vest to come": {
			vests:    []grantVest{vest("2026-11-15")},
			expected: "1 vest to come, the next on 2026-11-15",
		},
		"vests pending release, today's among them": {
			vests:    []grantVest{vest("2026-09-15"), vest("2026-10-02"), vest("2026-11-15")},
			expected: "2 vests pending release · 1 vest to come, the next on 2026-11-15",
		},
		"only a vest pending release": {
			vests:    []grantVest{vest("2026-09-15")},
			expected: "1 vest pending release",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, vestingSummary(tt.vests, today))
		})
	}
}

// TestDueIn covers how far off a vest is, in the words of the last column of the Vesting table. The
// nearer the day, the finer the unit: days up to two months out, then whole months, and whole years
// from two years on, since nobody plans by the day for a vest that is years away. A vest whose day
// has come and which has not been released is pending, as the bank calls it.
func TestDueIn(t *testing.T) {
	today := day("2026-10-02")

	tests := map[string]struct {
		date     string
		expected string
	}{
		"today":                    {date: "2026-10-02", expected: "pending"},
		"in the past":              {date: "2026-08-15", expected: "pending"},
		"tomorrow":                 {date: "2026-10-03", expected: "in 1 day"},
		"in some days":             {date: "2026-11-15", expected: "in 44 days"},
		"the last to read in days": {date: "2026-11-30", expected: "in 59 days"},
		"two months":               {date: "2026-12-02", expected: "in 2 months"},
		"months are rounded down":  {date: "2027-02-20", expected: "in 4 months"},
		"almost two years":         {date: "2028-10-01", expected: "in 23 months"},
		"two years":                {date: "2028-10-02", expected: "in 2 years"},
		"years are rounded down":   {date: "2030-08-20", expected: "in 3 years"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, dueIn(day(tt.date), today))
		})
	}
}
