package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// A Flyout is a panel of details about whatever is selected in a view, such as a lot of the
// Holdings table: what does not fit into a row. It knows nothing of what it shows, which its parent
// hands it as text, and it holds no state, so the parent builds a new one whenever it renders and
// the details cannot go stale. Unlike a dialog it takes no keys: the view keeps the keyboard, and
// the flyout follows the selection.
type Flyout struct {
	// Title says what the details are of.
	Title string

	Sections []FlyoutSection
}

// A FlyoutSection is a group of rows that belong together, under a heading.
type FlyoutSection struct {
	Title string
	Rows  []FlyoutRow
}

// A FlyoutRow is one detail: what it is on the left, and its value on the right.
type FlyoutRow struct {
	Label string
	Value string
}

// Layer renders the flyout as a compositor layer of exactly width by height cells, border included,
// for the parent to position over its own view. The size is the parent's to say, since it is the
// parent that knows what the flyout has to line up with.
func (f Flyout) Layer(width, height int, style Style) *lipgloss.Layer {
	box := style.Dialog.Width(width).Height(height)
	inner := width - box.GetHorizontalFrameSize()

	lines := []string{style.DialogTitle.Render(f.Title)}
	for _, section := range f.Sections {
		lines = append(lines, "", style.Hint.Render(section.Title))
		for _, row := range section.Rows {
			gap := max(inner-lipgloss.Width(row.Label)-lipgloss.Width(row.Value), 1)
			lines = append(lines, row.Label+strings.Repeat(" ", gap)+row.Value)
		}
	}

	return lipgloss.NewLayer(box.Render(strings.Join(lines, "\n")))
}
