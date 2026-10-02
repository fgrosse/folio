package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestAccountHeader covers the two lines above the table of every view, which are where the account
// values live: the total on the first line and the two values it is made of on the second, at the
// right end where the table's values are. What the view itself has to say goes on the left, with a
// status line under it.
func TestAccountHeader(t *testing.T) {
	account := portfolio.Account{Current: dec("3368.13"), Potential: dec("7925")}

	header := accountHeader("8.5 PANW", "PANW $396.25", account, 58, DefaultStyle())

	expected := "" +
		"  8.5 PANW                                 Total: $11,293.13\n" +
		"  PANW $396.25       Current $3,368.13 · Potential $7,925.00"
	assert.Equal(t, expected, ansi.Strip(header))

	// Both lines end in the column the table's last cell does: the indent and the width given.
	for line := range strings.SplitSeq(expected, "\n") {
		assert.Equal(t, 60, lipgloss.Width(line))
	}
}
