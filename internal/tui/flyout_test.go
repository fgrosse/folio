package tui

import (
	"strings"
	"testing"

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
