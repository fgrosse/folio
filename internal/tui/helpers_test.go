package tui

import (
	"testing"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// runCmd executes cmd and returns the message it produced. The timeout keeps a command that never
// returns from hanging the whole test binary instead of failing one test.
func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	require.NotNil(t, cmd, "expected a command to run")

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	select {
	case msg := <-done:
		return msg
	case <-time.After(time.Second):
		t.Fatal("command produced no message within a second")
		return nil
	}
}

func keysPressed(s string) []tea.Msg {
	msgs := make([]tea.Msg, len(s))
	for i, r := range s {
		msgs[i] = tea.KeyPressMsg{Code: r, Text: string(r)}
	}

	return msgs
}

func keyPressed(s string) tea.KeyPressMsg {
	if len(s) != 1 {
		panic("expected a single key")
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

// day parses a calendar day written as YYYY-MM-DD.
func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}

	return t
}

// dec parses a decimal, such as a number of shares or a price, written the way it reads.
func dec(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

// plainRows returns rows with the styling taken out of their cells, such as the color of a gain,
// for a test of what the rows say rather than how they look.
func plainRows(rows []table.Row) []table.Row {
	plain := make([]table.Row, len(rows))
	for i, row := range rows {
		plain[i] = make(table.Row, len(row))
		for j, cell := range row {
			plain[i][j] = ansi.Strip(cell)
		}
	}

	return plain
}
