package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestDemoCmd covers trying folio out without an account of one's own: "folio demo" writes a
// made-up account to a database at the path it is given and says how to open it. It leaves the
// database that --db names alone, which is where the real account is, and does not even create it.
func TestDemoCmd(t *testing.T) {
	demoPath := filepath.Join(t.TempDir(), "try", "demo.db")
	cmd, dbPath := NewTestingCmd(t, "demo", "--seed", "7", demoPath)
	cmd.now = func() time.Time { return time.Date(2026, time.October, 2, 14, 30, 0, 0, time.UTC) }

	var out strings.Builder
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())

	expected := "" +
		"Wrote a demo account to " + demoPath + "\n" +
		"\n" +
		"Open it with:\n" +
		"  folio --db " + demoPath + "\n"
	assert.Equal(t, expected, out.String())

	store, err := portfolio.NewStore(demoPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	grants, err := store.Grants()
	require.NoError(t, err)
	assert.Len(t, grants, 2)

	assert.NoFileExists(t, dbPath, "the demo must not touch the database of the real account")
}

// TestDemoCmd_DefaultPath covers "folio demo" without a path: the demo database goes next to where
// the real one is kept, under another name.
func TestDemoCmd_DefaultPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	cmd, _ := NewTestingCmd(t, "demo")
	require.NoError(t, cmd.Execute())

	assert.FileExists(t, filepath.Join(dataHome, "folio", "demo.db"))
	assert.NoFileExists(t, filepath.Join(dataHome, "folio", "folio.db"))
}

// TestDemoCmd_KeepsWhatIsThere covers a path that has a file already, which may be an account
// someone entered: the demo refuses to write over it, and leaves it as it was.
func TestDemoCmd_KeepsWhatIsThere(t *testing.T) {
	demoPath := filepath.Join(t.TempDir(), "demo.db")
	require.NoError(t, os.WriteFile(demoPath, []byte("an account"), 0o600))

	cmd, _ := NewTestingCmd(t, "demo", demoPath)
	err := cmd.Execute()

	assert.EqualError(t, err, demoPath+" exists already: remove it, or give the demo another path")
	content, readErr := os.ReadFile(demoPath)
	require.NoError(t, readErr)
	assert.Equal(t, "an account", string(content))
}
