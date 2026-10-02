package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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
