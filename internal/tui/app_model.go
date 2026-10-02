// Package tui implements folio's interactive terminal UI, built on Bubble Tea.
package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/fgrosse/folio/internal/portfolio"
)

// tabGap separates one tab from the next in the tab bar. It is the same bullet the help lines
// above the bar use, so the two read as one block.
const tabGap = " • "

// tabBarHeight is the single line the tab bar takes up below the view on display.
const tabBarHeight = 1

// An AppModel shows one of several views at a time and switches between them: the digit keys select
// a view by its number, and tab and shift+tab cycle through them.
type AppModel struct {
	style    Style
	views    []ViewModel
	selected int
}

// A ViewModel is one of the views an AppModel can show: a tea.Model that also has a title, which is
// what the tab bar lists it under.
type ViewModel interface {
	tea.Model
	Title() string
}

// A KeyCapturer is a ViewModel that can take the whole keyboard for itself. While it says it is
// capturing keys - a dialog of its own is open - the AppModel hands it every key press, including
// the tab and the digits that would otherwise switch views: a digit is part of the lot or grant
// being typed. A view that never opens a dialog does not need to implement this.
type KeyCapturer interface {
	CapturesKeys() bool
}

// New returns the app that the folio TUI runs: the views over store, valued at the quotes of quoter
// and all rendered in style. It is the one place that says which views the app has and in what
// order, so that the tab bar and the digit keys that select them follow from a single list.
func New(store Store, quoter portfolio.Quoter, style Style) *AppModel {
	return NewAppModel(style,
		NewHoldingsModel(store, quoter, style),
	)
}

// NewAppModel returns an AppModel over views, showing the first of them. It panics if there is not
// at least one view, which is a programming error rather than anything a user can cause.
func NewAppModel(style Style, views ...ViewModel) *AppModel {
	if len(views) == 0 {
		panic("views must not be empty")
	}

	return &AppModel{
		style: style,
		views: views,
	}
}

// Init implements tea.Model by starting every view, not only the one on display: the view behind it
// has to have loaded by the time it is switched to.
func (m *AppModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, view := range m.views {
		cmds = append(cmds, view.Init())
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model by handling the keys that switch views.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	default:
		return m.updateViews(msg)
	}
}

// handleKeyPress switches views if msg is one of the keys that do, and otherwise leaves the key to
// the view on display. The AppModel stays the model that comes out either way: a view returned here
// would replace the whole app, tab bar and all.
func (m *AppModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if !m.activeViewCapturesKeys() && m.handleViewSwitch(msg) {
		return m, nil
	}

	updated, cmd := m.activeView().Update(msg)
	m.views[m.selected] = asViewModel(updated)

	return m, cmd
}

func (m *AppModel) handleViewSwitch(msg tea.KeyPressMsg) (switched bool) {
	if msg.Code == tea.KeyTab {
		if msg.Mod == tea.ModShift {
			m.cycleView(-1)
		} else {
			m.cycleView(1)
		}

		return true
	}

	n, err := strconv.Atoi(msg.Text)
	if err != nil {
		// ignore keys which are not numbers
		return false
	}

	if n == 0 || n > len(m.views) {
		// ignore numbers that do not relate to a view
		return false
	}

	m.selected = n - 1
	return true
}

// activeViewCapturesKeys reports whether the view on display has claimed the keyboard, which leaves
// the app none of the keys it would otherwise switch views with.
func (m *AppModel) activeViewCapturesKeys() bool {
	capturer, ok := m.activeView().(KeyCapturer)
	return ok && capturer.CapturesKeys()
}

// cycleView moves the selection on by step views, wrapping around at either end.
func (m *AppModel) cycleView(step int) {
	n := len(m.views)
	// Adding n first keeps the left side of the % positive: the remainder in Go carries the
	// sign of the dividend, so stepping back from the first view would otherwise land on -1.
	m.selected = (m.selected + step + n) % n
}

// updateViews handles all tea messages that are not key presses by publishing them to *all* views.
// A view that isn't on display still has to see non-key messages: its width would go stale, and the
// message of a command it started would otherwise arrive at the wrong view.
func (m *AppModel) updateViews(msg tea.Msg) (tea.Model, tea.Cmd) {
	// A view has the window minus the line the tab bar sits on. Told the full height, it would
	// render one line too many and push the tab bar off the bottom of the screen.
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		size.Height -= tabBarHeight
		msg = size
	}

	var cmds []tea.Cmd
	for i, view := range m.views {
		updated, cmd := view.Update(msg)
		m.views[i] = asViewModel(updated)
		cmds = append(cmds, cmd) // tea.Batch ignores nil commands
	}

	return m, tea.Batch(cmds...)
}

// asViewModel is the model a view's Update returned, which has to be a view again: the AppModel
// holds on to it, and a view that swapped itself for another kind of model could no longer be shown
// or switched to.
func asViewModel(model tea.Model) ViewModel {
	view, ok := model.(ViewModel)
	if !ok {
		panic(fmt.Sprintf("a view's Update returned %T, which is not a tui.ViewModel", model))
	}

	return view
}

// View implements tea.Model by rendering the tab bar below the view on display, under its help.
func (m *AppModel) View() tea.View {
	titles := make([]string, len(m.views))
	for i, view := range m.views {
		titles[i] = view.Title()
	}

	// A view ends its frame with a newline of its own, which would leave the bar a blank line
	// adrift from the help above it. Only that one newline goes: any before it end lines the view
	// drew on purpose, such as the empty second help line under an open dialog, and taking those
	// too would make the frame shorter than the window.
	content := strings.TrimSuffix(m.activeView().View().Content, "\n")
	content += "\n"
	content += tabBar(titles, m.selected, m.style)

	return tea.NewView(content)
}

// activeView is the view on display.
func (m *AppModel) activeView() ViewModel {
	return m.views[m.selected]
}

// tabBar renders the line below the help that lists every view, each title after the number key
// that selects it. It is separated and aligned like the help lines above it, so it reads as one
// more of them. The tab at index selected is rendered in the style's TabSelected and every other
// tab in its Tab.
func tabBar(titles []string, selected int, style Style) string {
	tabs := make([]string, len(titles))
	for i, title := range titles {
		tabStyle := style.Tab
		if i == selected {
			tabStyle = style.TabSelected
		}

		tabs[i] = tabStyle.Render(strconv.Itoa(i+1) + " " + title)
	}

	return strings.Join(tabs, tabGap)
}
