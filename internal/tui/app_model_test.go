package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// loadedMsg and tickMsg stand in for the messages of real views, which the AppModel passes on
// without looking at them.
type (
	loadedMsg struct{}
	tickMsg   struct{}
)

// stubView is a tea.Model that stands in for a real view: it has a title and remembers what it was
// updated with, but renders nothing.
type stubView struct {
	title    string
	content  string
	msgs     []tea.Msg
	captures bool    // what CapturesKeys reports, i.e. whether the view has a dialog open
	cmd      tea.Cmd // returned from every Update, so a test can follow it back out
	initCmd  tea.Cmd // returned from Init, for the same reason
}

func (v *stubView) Title() string { return v.title }

func (v *stubView) CapturesKeys() bool { return v.captures }

func (v *stubView) Init() tea.Cmd { return v.initCmd }

// TestAppModel_InitStartsEveryView covers that starting the app starts all of its views, not only
// the one on display: the view behind it has to have loaded by the time it is switched to.
func TestAppModel_InitStartsEveryView(t *testing.T) {
	holdings := &stubView{title: "Holdings", initCmd: func() tea.Msg { return tickMsg{} }}
	vesting := &stubView{title: "Vesting", initCmd: func() tea.Msg { return loadedMsg{} }}
	quiet := &stubView{title: "Quiet"} // a view with nothing to do on startup holds up nothing
	m := NewAppModel(nil, DefaultStyle(), holdings, vesting, quiet)

	cmd := m.Init()
	require.NotNil(t, cmd, "the commands of the views are passed on")

	batch, ok := runCmd(t, cmd).(tea.BatchMsg)
	require.True(t, ok, "Init batches the commands of the views")
	require.Len(t, batch, 2, "the view without a command adds nothing to the batch")

	msgs := []tea.Msg{runCmd(t, batch[0]), runCmd(t, batch[1])}
	assert.ElementsMatch(t, []tea.Msg{tickMsg{}, loadedMsg{}}, msgs)
}

func (v *stubView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	v.msgs = append(v.msgs, msg)
	return v, v.cmd
}

func (v *stubView) View() tea.View { return tea.NewView(v.content) }

// TestAppModel_RendersTabBarBelowActiveView covers the frame the app renders:
// the view on display, then the tab bar, and nothing of the other views.
func TestAppModel_RendersTabBarBelowActiveView(t *testing.T) {
	holdings := &stubView{title: "Holdings", content: "the lots"}
	vesting := &stubView{title: "Vesting", content: "the vests"}
	m := NewAppModel(nil, DefaultStyle(), holdings, vesting)

	assert.Equal(t, "the lots\n1 Holdings • 2 Vesting", ansi.Strip(m.View().Content))

	m = driveApp(t, m, keyPressed("2"))
	assert.Equal(t, "the vests\n1 Holdings • 2 Vesting", ansi.Strip(m.View().Content))
}

// TestAppModel_SwitchView covers the keys that move between views: a digit selects the view with
// that number, and tab cycles forwards while shift+tab cycles backwards, wrapping at either end.
// A digit without a view of its own leaves the selection alone.
func TestAppModel_SwitchView(t *testing.T) {
	tab := tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTab := tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}

	cases := map[string]struct {
		keys     []tea.Msg
		expected int
	}{
		"nothing pressed yet":     {keys: nil, expected: 0},
		"2 selects the second":    {keys: keysPressed("2"), expected: 1},
		"1 selects the first":     {keys: keysPressed("21"), expected: 0},
		"3 has no view":           {keys: keysPressed("3"), expected: 0},
		"0 has no view":           {keys: keysPressed("20"), expected: 1},
		"tab moves on":            {keys: []tea.Msg{tab}, expected: 1},
		"tab wraps around":        {keys: []tea.Msg{tab, tab}, expected: 0},
		"shift+tab moves back":    {keys: []tea.Msg{tab, shiftTab}, expected: 0},
		"shift+tab wraps as well": {keys: []tea.Msg{shiftTab}, expected: 1},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := NewAppModel(nil, DefaultStyle(), &stubView{title: "Holdings"}, &stubView{title: "Vesting"})
			for _, key := range c.keys {
				updated, _ := m.Update(key)
				next, ok := updated.(*AppModel)
				require.True(t, ok, "Update returned %T, which is not a *tui.AppModel", updated)
				m = next
			}

			assert.Equal(t, c.expected, m.selected)
		})
	}
}

// TestAppModel_ForwardsKeysToActiveView covers what happens to a key that does not switch views: it
// goes to the view on display and to no other. The keys that do switch are the AppModel's own and
// are not passed on, or a digit typed into a view would arrive twice over.
func TestAppModel_ForwardsKeysToActiveView(t *testing.T) {
	holdings := &stubView{title: "Holdings"}
	vesting := &stubView{title: "Vesting"}
	m := NewAppModel(nil, DefaultStyle(), holdings, vesting)

	m = driveApp(t, m, keyPressed("a"))
	assert.Equal(t, []tea.Msg{keyPressed("a")}, holdings.msgs, "the view on display gets the key")
	assert.Empty(t, vesting.msgs, "the other view gets nothing")

	driveApp(t, m, keyPressed("2"), keyPressed("e"))
	assert.Equal(t, []tea.Msg{keyPressed("a")}, holdings.msgs, "the view left behind gets nothing more")
	assert.Equal(t, []tea.Msg{keyPressed("e")}, vesting.msgs, "the view switched to gets the key")
}

// TestAppModel_ForwardsOtherMessagesToEveryView covers messages that are not keys: a view that is
// not on display still has to see them, or the size it renders at goes stale and the commands it
// started come back to nobody.
func TestAppModel_ForwardsOtherMessagesToEveryView(t *testing.T) {
	holdings := &stubView{title: "Holdings"}
	vesting := &stubView{title: "Vesting"}
	m := NewAppModel(nil, DefaultStyle(), holdings, vesting)

	driveApp(t, m, tickMsg{})

	assert.Equal(t, []tea.Msg{tickMsg{}}, holdings.msgs)
	assert.Equal(t, []tea.Msg{tickMsg{}}, vesting.msgs)
}

// TestAppModel_ShrinksTheWindowForTheTabBar covers the one message the AppModel does not pass on
// untouched: the window is a line shorter for a view than it is for the app, because the tab bar
// sits on that line. A view that rendered to the full height would push the tab bar off the screen.
func TestAppModel_ShrinksTheWindowForTheTabBar(t *testing.T) {
	holdings := &stubView{title: "Holdings"}
	vesting := &stubView{title: "Vesting"}
	m := NewAppModel(nil, DefaultStyle(), holdings, vesting)

	driveApp(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	expected := []tea.Msg{tea.WindowSizeMsg{Width: 100, Height: 39}}
	assert.Equal(t, expected, holdings.msgs)
	assert.Equal(t, expected, vesting.msgs)
}

// TestAppModel_PassesOnCommands covers that a command a view returns is not dropped on the way out
// of the AppModel's Update.
func TestAppModel_PassesOnCommands(t *testing.T) {
	refresh := func() tea.Msg { return tickMsg{} }
	holdings := &stubView{title: "Holdings", cmd: refresh}
	m := NewAppModel(nil, DefaultStyle(), holdings, &stubView{title: "Vesting"})

	_, cmd := m.Update(keyPressed("a"))
	require.NotNil(t, cmd, "the command of the view on display is passed on")
	assert.IsType(t, tickMsg{}, runCmd(t, cmd))
}

// driveApp feeds msgs through the AppModel's Update in order and returns the model that comes out
// the far end, like drive does for the lots.
func driveApp(t *testing.T, m *AppModel, msgs ...tea.Msg) *AppModel {
	t.Helper()

	for _, msg := range msgs {
		updated, _ := m.Update(msg)
		next, ok := updated.(*AppModel)
		require.True(t, ok, "Update returned %T, which is not a *tui.AppModel", updated)
		m = next
	}

	return m
}

// TestTabBar covers the text of the tab bar: every view's title after the number that selects it,
// bulleted like the help lines above it. Which tab is selected only shows in the styling, so the
// text is the same whichever one it is.
func TestTabBar(t *testing.T) {
	cases := map[string]struct {
		selected int
		expected string
	}{
		"first tab selected": {
			selected: 0,
			expected: "1 Holdings • 2 Vesting",
		},
		"second tab selected": {
			selected: 1,
			expected: "1 Holdings • 2 Vesting",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			bar := tabBar([]string{"Holdings", "Vesting"}, c.selected, DefaultStyle())
			assert.Equal(t, c.expected, ansi.Strip(bar))
		})
	}
}

// TestTabBar_Styled covers which tab stands out: the selected one, number and all, is rendered in
// TabSelected and every other tab in Tab. The styles are the test's own, so that they are certain
// to differ whatever DefaultStyle picks.
func TestTabBar_Styled(t *testing.T) {
	style := Style{
		Tab:         lipgloss.NewStyle().Faint(true),
		TabSelected: lipgloss.NewStyle().Bold(true),
	}

	cases := map[string]struct {
		selected int
		expected string
	}{
		"first tab selected": {
			selected: 0,
			expected: style.TabSelected.Render("1 Holdings") + " • " + style.Tab.Render("2 Vesting"),
		},
		"second tab selected": {
			selected: 1,
			expected: style.Tab.Render("1 Holdings") + " • " + style.TabSelected.Render("2 Vesting"),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, tabBar([]string{"Holdings", "Vesting"}, c.selected, style))
		})
	}
}

// TestAppModel_CapturingViewKeepsTheKeys covers the keys that switch views going to the view on
// display instead, while that view says it is capturing them. That is what a dialog does: a digit
// is typed into the spec of a lot or grant, so it may not be spent on the tab bar. The view keeps the keys only while it says so, so the tab bar works again
// as soon as the dialog closes.
func TestAppModel_CapturingViewKeepsTheKeys(t *testing.T) {
	tab := tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTab := tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	two := keyPressed("2")

	holdings := &stubView{title: "Holdings", captures: true}
	vesting := &stubView{title: "Vesting"}
	m := NewAppModel(nil, DefaultStyle(), holdings, vesting)

	m = driveApp(t, m, tab, shiftTab, two)
	assert.Equal(t, 0, m.selected, "a capturing view is not switched away from")
	assert.Equal(t, []tea.Msg{tab, shiftTab, two}, holdings.msgs, "the keys go to the view itself")
	assert.Empty(t, vesting.msgs, "and to no other view")

	holdings.captures = false
	m = driveApp(t, m, tab)
	assert.Equal(t, 1, m.selected, "once the view lets go, tab switches views again")
	assert.Len(t, holdings.msgs, 3, "and the key that switched is not passed on")
}

// TestNew_Render is the golden of the whole frame the app puts on screen: the Holdings view with the
// tab bar under it, which lists the three views folio has. Each view has a golden of its own, and
// none of them can see what this one is here for - that the app fits a view and the tab bar into
// the window together.
func TestNew_Render(t *testing.T) {
	store := new(MockStore)
	store.returns(testPortfolio())
	store.On("SaveQuote", mock.Anything).Return(nil)

	m := New(store, quotes{"PANW": "396.25"}, DefaultStyle())
	m = driveApp(t, m, tea.WindowSizeMsg{Width: 100, Height: 21})
	m = driveApp(t, m, PortfolioLoadedMsg{portfolio: testPortfolio()})

	frame := ansi.Strip(m.View().Content)
	assert.Equal(t, 21, lipgloss.Height(frame), "the frame should fill the window, and no more")
	golden.RequireEqual(t, frame)
}

// TestNew_TabsAreTheSameHeight covers what makes switching views calm: every view renders a frame of
// the same height, so the tab bar stays on the line it is on and nothing above it jumps.
func TestNew_TabsAreTheSameHeight(t *testing.T) {
	store := new(MockStore)
	store.returns(testPortfolio())

	m := New(store, quotes{}, DefaultStyle())
	m = driveApp(t, m, tea.WindowSizeMsg{Width: 100, Height: 21})
	m = driveApp(t, m, PortfolioLoadedMsg{portfolio: testPortfolio()})

	height := lipgloss.Height(m.View().Content)
	assert.Contains(t, ansi.Strip(m.View().Content), "1 Holdings • 2 Vesting • 3 Grants • 4 Sales")

	for _, key := range []string{"2", "3", "4"} {
		m = driveApp(t, m, keyPressed(key))
		assert.Equal(t, height, lipgloss.Height(m.View().Content), "tab %s should be as tall as the first", key)
	}
}

// TestAppModel_OpensConfig covers the key that opens the configuration: c loads what the keys of
// the configuration are set to from the store, and the dialog opens on those values, in front of
// whichever view is on display. The key is the app's, so no view sees it. A view that is capturing
// the keys keeps this one too: it is a letter like any other in the text being typed there.
func TestAppModel_OpensConfig(t *testing.T) {
	store := new(MockStore)
	store.On("GetConfig", portfolio.TaxRateKey).Return("44.3%", nil)
	store.On("GetConfig", mock.Anything).Return("", portfolio.ErrNotSet)

	holdings := &stubView{title: "Holdings", content: "the lots"}
	m := NewAppModel(store, DefaultStyle(), holdings)

	_, cmd := m.Update(keyPressed("c"))
	loaded := runCmd(t, cmd)
	assert.Equal(t, ConfigLoadedMsg{values: map[string]string{"tax-rate": "44.3%"}}, loaded)
	assert.NotContains(t, ansi.Strip(m.View().Content), "tax-rate", "the dialog waits for its values")

	m = driveApp(t, m, loaded)
	frame := ansi.Strip(m.View().Content)
	assert.Contains(t, frame, "> tax-rate   44.3%")
	assert.Contains(t, frame, "  potential  ‹ gross ›")
	assert.Empty(t, holdings.msgs, "the view sees neither the key nor what was loaded")

	typing := &stubView{title: "Holdings", captures: true}
	m = NewAppModel(store, DefaultStyle(), typing)

	_, cmd = m.Update(keyPressed("c"))
	assert.Nil(t, cmd)
	assert.Equal(t, []tea.Msg{keyPressed("c")}, typing.msgs, "a capturing view gets the key itself")
}

// openConfig returns an app over views whose configuration dialog is open on values.
func openConfig(t *testing.T, store Store, values map[string]string, views ...ViewModel) *AppModel {
	t.Helper()

	m := NewAppModel(store, DefaultStyle(), views...)
	return driveApp(t, m, ConfigLoadedMsg{values: values})
}

// TestAppModel_ConfigKeepsTheKeys covers the keyboard while the configuration is open: every key is
// the dialog's, the ones that switch views and the ones of the view behind it included. Esc closes
// the dialog, and the keys are back where they were.
func TestAppModel_ConfigKeepsTheKeys(t *testing.T) {
	holdings := &stubView{title: "Holdings", content: "the lots"}
	vesting := &stubView{title: "Vesting", content: "the vests"}
	m := openConfig(t, nil, nil, holdings, vesting)

	m = driveApp(t, m, keyPressed("2"), tea.KeyPressMsg{Code: tea.KeyTab}, keyPressed("a"))
	assert.Equal(t, 0, m.selected, "the view behind the dialog stays on display")
	assert.Empty(t, holdings.msgs, "and gets none of the keys")

	m = driveApp(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Contains(t, ansi.Strip(m.View().Content), "> potential", "the dialog gets them")

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = driveApp(t, m, runCmd(t, cmd))
	assert.NotContains(t, ansi.Strip(m.View().Content), "potential", "esc closes the dialog")
	assert.Empty(t, holdings.msgs, "which is nothing a view has to know of")

	m = driveApp(t, m, keyPressed("2"))
	assert.Equal(t, 1, m.selected, "the keys switch views again")
}

// TestAppModel_SetsConfig covers what the app does with a value the dialog asks to be set: it is
// stored, the dialog is given the configuration as the store has it then, and the portfolio is
// loaded again for every view, since what they show depends on the configuration.
func TestAppModel_SetsConfig(t *testing.T) {
	store := new(MockStore)
	store.returnsAccount(testPortfolio())
	store.On("SetConfig", "potential", "net").Return(nil)
	store.On("GetConfig", portfolio.PotentialBasisKey).Return("net", nil)
	store.On("GetConfig", mock.Anything).Return("", portfolio.ErrNotSet)

	holdings := &stubView{title: "Holdings", content: "the lots"}
	m := openConfig(t, store, nil, holdings)

	_, cmd := m.Update(SetConfigMsg{key: "potential", value: "net"})
	saved := runCmd(t, cmd)
	store.AssertCalled(t, "SetConfig", "potential", "net")
	assert.Equal(t, ConfigSavedMsg{values: map[string]string{"potential": "net"}}, saved)

	_, cmd = m.Update(saved)
	assert.Contains(t, ansi.Strip(m.View().Content), "‹ net ›")

	loaded, ok := runCmd(t, cmd).(PortfolioLoadedMsg)
	require.True(t, ok, "the portfolio is loaded again")
	assert.Equal(t, portfolio.Net, loaded.portfolio.PotentialBasis)

	driveApp(t, m, loaded)
	assert.Equal(t, []tea.Msg{loaded}, holdings.msgs, "and every view receives it")
}

// TestAppModel_UnsetsConfig covers what the app does when the dialog asks for a value to be taken
// back: the key is unset in the store, and the dialog and the views are brought up to date as they
// are after a value was set.
func TestAppModel_UnsetsConfig(t *testing.T) {
	store := new(MockStore)
	store.returnsAccount(testPortfolio())
	store.On("UnsetConfig", "tax-rate").Return(nil)
	store.On("GetConfig", mock.Anything).Return("", portfolio.ErrNotSet)

	holdings := &stubView{title: "Holdings", content: "the lots"}
	m := openConfig(t, store, map[string]string{"tax-rate": "44.3%"}, holdings)

	_, cmd := m.Update(UnsetConfigMsg{key: "tax-rate"})
	saved := runCmd(t, cmd)
	store.AssertCalled(t, "UnsetConfig", "tax-rate")
	assert.Equal(t, ConfigSavedMsg{values: map[string]string{}}, saved)

	_, cmd = m.Update(saved)
	assert.Contains(t, ansi.Strip(m.View().Content), "> tax-rate   not set")

	loaded, ok := runCmd(t, cmd).(PortfolioLoadedMsg)
	require.True(t, ok, "the portfolio is loaded again")
	assert.False(t, loaded.portfolio.TaxRate.Valid)
}

// TestAppModel_ConfigErrors covers a store that fails while the configuration is edited. The dialog
// says why, where the change was asked for, and keeps the values it has: one that could not be
// stored is not shown as if it had been. The views are left alone, since nothing changed for them.
// A configuration that cannot be loaded opens the dialog all the same, to say so.
func TestAppModel_ConfigErrors(t *testing.T) {
	store := new(MockStore)
	store.On("SetConfig", "potential", "net").Return(errors.New("disk full"))
	store.On("UnsetConfig", "tax-rate").Return(errors.New("disk full"))
	store.On("GetConfig", portfolio.TaxRateKey).Return("44.3%", nil)
	store.On("GetConfig", mock.Anything).Return("", portfolio.ErrNotSet)

	tests := map[string]struct {
		msg      tea.Msg
		expected string
	}{
		"set":   {msg: SetConfigMsg{key: "potential", value: "net"}, expected: "set potential: disk full"},
		"unset": {msg: UnsetConfigMsg{key: "tax-rate"}, expected: "unset tax-rate: disk full"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m := openConfig(t, store, map[string]string{"tax-rate": "44.3%"}, &stubView{title: "Holdings"})

			_, cmd := m.Update(tt.msg)
			_, cmd = m.Update(runCmd(t, cmd))
			assert.Nil(t, cmd, "nothing changed that the views would have to load")

			frame := ansi.Strip(m.View().Content)
			assert.Contains(t, frame, tt.expected)
			assert.Contains(t, frame, "> tax-rate   44.3%")
			assert.Contains(t, frame, "‹ gross ›")
		})
	}

	broken := new(MockStore)
	broken.On("GetConfig", mock.Anything).Return("", errors.New("disk on fire"))
	m := NewAppModel(broken, DefaultStyle(), &stubView{title: "Holdings"})

	_, cmd := m.Update(keyPressed("c"))
	m = driveApp(t, m, runCmd(t, cmd))
	assert.Contains(t, ansi.Strip(m.View().Content), "get tax-rate: disk on fire")
}
