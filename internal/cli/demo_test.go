package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/tui"
)

// newDemoCmd returns the folio command set up to run "demo" with args on the 2nd of October 2026.
// In place of the TUI, which needs a terminal, it calls opened with the store the TUI would have
// been given. It also returns the path of the database of the real account, for a test to check
// that the demo left it alone.
func newDemoCmd(t *testing.T, opened func(store tui.Store), args ...string) (*Folio, string) {
	t.Helper()

	cmd, dbPath := NewTestingCmd(t, append([]string{"demo"}, args...)...)
	cmd.now = func() time.Time { return time.Date(2026, time.October, 2, 14, 30, 0, 0, time.UTC) }
	cmd.openTUI = func(_ context.Context, store tui.Store) error {
		opened(store)
		return nil
	}

	return cmd, dbPath
}

// account renders what the account in store holds as text, for two accounts to be compared by.
func account(t *testing.T, store tui.Store) string {
	t.Helper()

	grants, err := store.Grants()
	require.NoError(t, err)
	lots, err := store.Lots()
	require.NoError(t, err)

	var s string
	for _, grant := range grants {
		s += grant.Name + " " + grant.Symbol + " " + grant.Vests[0].Date.Format(time.DateOnly) + "\n"
	}
	for _, lot := range lots {
		s += lot.String() + "\n"
	}

	return s
}

// TestDemoCmd covers trying folio out without an account of one's own: "folio demo" makes up an
// account and opens the TUI on it right away. The account is in a database of its own that is gone
// again once the TUI is closed, and the database of the real account, which --db names, is left
// alone: the demo does not even create it.
func TestDemoCmd(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	var grants int
	cmd, dbPath := newDemoCmd(t, func(store tui.Store) {
		all, err := store.Grants()
		require.NoError(t, err)
		grants = len(all)

		left, err := os.ReadDir(tmp)
		require.NoError(t, err)
		assert.Len(t, left, 1, "the demo database should be there while the TUI is open")
	})

	require.NoError(t, cmd.Execute())

	assert.Equal(t, 2, grants, "the TUI should be opened on the demo account")
	assert.NoFileExists(t, dbPath, "the demo must not touch the database of the real account")

	left, err := os.ReadDir(tmp)
	require.NoError(t, err)
	assert.Empty(t, left, "the demo database should be gone once the TUI is closed")
}

// TestDemoCmd_Seed covers the account the demo makes up: another one every time, unless --seed names
// one, which is then the same every time.
func TestDemoCmd_Seed(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	run := func(args ...string) string {
		var made string
		cmd, _ := newDemoCmd(t, func(store tui.Store) { made = account(t, store) }, args...)
		require.NoError(t, cmd.Execute())
		require.NotEmpty(t, made)

		return made
	}

	seven := run("--seed", "7")
	assert.Equal(t, seven, run("--seed", "7"))
	assert.NotEqual(t, seven, run("--seed", "8"))
}

// TestDemoCmd_Path covers a demo that is meant to be kept: given a path, the demo writes its
// database there and leaves it when the TUI is closed. Run again with that path, it opens what is
// there, changes and all, rather than making up a new account in its place.
func TestDemoCmd_Path(t *testing.T) {
	demoPath := filepath.Join(t.TempDir(), "try", "demo.db")

	var first, second string
	cmd, _ := newDemoCmd(t, func(store tui.Store) { first = account(t, store) }, "--seed", "7", demoPath)
	require.NoError(t, cmd.Execute())
	assert.FileExists(t, demoPath)

	cmd, _ = newDemoCmd(t, func(store tui.Store) { second = account(t, store) }, "--seed", "8", demoPath)
	require.NoError(t, cmd.Execute())

	require.NotEmpty(t, first)
	assert.Equal(t, first, second, "an account that is there should be opened as it is")
}
