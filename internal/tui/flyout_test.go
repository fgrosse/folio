package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

// TestFlyout_Layer covers what a flyout looks like: a box of exactly the size it is asked for,
// whatever it has to say, with its title on top and under it the sections, each a heading over rows
// that have their label on the left and their value on the right.
func TestFlyout_Layer(t *testing.T) {
	f := Flyout{
		Title: "PANW of 2026-01-15",
		Sections: []FlyoutSection{
			{Title: "Lot", Rows: []FlyoutRow{
				{Label: "Acquired", Value: "2026-01-15"},
				{Label: "Shares", Value: "6"},
			}},
			{Title: "Value", Rows: []FlyoutRow{
				{Label: "Price", Value: "$396.25"},
			}},
		},
	}

	layer := f.Layer(30, 12, DefaultStyle())

	expected := strings.Join([]string{
		"╭────────────────────────────╮",
		"│ PANW of 2026-01-15         │",
		"│                            │",
		"│ Lot                        │",
		"│ Acquired        2026-01-15 │",
		"│ Shares                   6 │",
		"│                            │",
		"│ Value                      │",
		"│ Price              $396.25 │",
		"│                            │",
		"│                            │",
		"╰────────────────────────────╯",
	}, "\n")
	assert.Equal(t, expected, ansi.Strip(layer.GetContent()))
	assert.Equal(t, 30, layer.Width())
	assert.Equal(t, 12, layer.Height())
}

// TestFlyout_LayerStylesValues covers a value that should stand out, such as a gain: a row can say
// which style its value is in, and that takes up no room, so the value still ends at the right edge.
func TestFlyout_LayerStylesValues(t *testing.T) {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	f := Flyout{
		Title: "PANW of 2026-01-15",
		Sections: []FlyoutSection{
			{Title: "Value", Rows: []FlyoutRow{
				{Label: "Gain", Value: "$96.78", ValueStyle: green},
			}},
		},
	}

	content := f.Layer(30, 6, DefaultStyle()).GetContent()

	assert.Contains(t, content, green.Render("$96.78"))
	assert.Contains(t, ansi.Strip(content), "│ Gain                $96.78 │")
}

// TestFlyout_LayerCutsOff covers a flyout that has more to say than it has room for, such as a lot
// with a long list of sales in a small window: it stays the size it was asked for, with its frame
// whole, and what does not fit is left out.
func TestFlyout_LayerCutsOff(t *testing.T) {
	f := Flyout{
		Title: "PANW of 2026-01-15",
		Sections: []FlyoutSection{
			{Title: "Lot", Rows: []FlyoutRow{
				{Label: "Acquired", Value: "2026-01-15"},
				{Label: "Shares", Value: "6"},
			}},
		},
	}

	layer := f.Layer(30, 6, DefaultStyle())

	expected := strings.Join([]string{
		"╭────────────────────────────╮",
		"│ PANW of 2026-01-15         │",
		"│                            │",
		"│ Lot                        │",
		"│ Acquired        2026-01-15 │",
		"╰────────────────────────────╯",
	}, "\n")
	assert.Equal(t, expected, ansi.Strip(layer.GetContent()))
}
