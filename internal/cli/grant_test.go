package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestGrantCmd covers recording a grant from the command line: "folio grant" takes a grant spec,
// quoted or as separate arguments, and saves the grant with the vests the spec lays out.
func TestGrantCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "grant", "Acquisition payout:", "10", "PANW", "monthly", "x24", "from", "2026-01-15")
	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	grants, err := store.Grants()
	require.NoError(t, err)
	require.Len(t, grants, 1)
	assert.Equal(t, "Acquisition payout", grants[0].Name)
	assert.Equal(t, "PANW", grants[0].Symbol)
	assert.Len(t, grants[0].Vests, 24)
}

// TestGrantCmd_Vests covers a grant whose schedule no rule lays out: with --vests, the spec is only
// the name and the symbol, and the vests are read from a file that lists them, a day and a number
// of shares to a line. Without vests in the file there is no grant to record.
func TestGrantCmd_Vests(t *testing.T) {
	schedule := filepath.Join(t.TempDir(), "vests.txt")
	require.NoError(t, os.WriteFile(schedule, []byte("2025-08-01 480\n2025-09-01 42\n2025-10-01 41\n"), 0o600))

	cmd, dbPath := NewTestingCmd(t, "grant", "--vests", schedule, "Acquisition payout: panw")
	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	grants, err := store.Grants()
	require.NoError(t, err)
	require.Len(t, grants, 1)
	assert.Equal(t, "Acquisition payout", grants[0].Name)
	assert.Equal(t, "PANW", grants[0].Symbol)
	require.Len(t, grants[0].Vests, 3)
	assert.Equal(t, "480", grants[0].Vests[0].Shares.String())
	assert.Equal(t, "41", grants[0].Vests[2].Shares.String())

	empty := filepath.Join(t.TempDir(), "empty.txt")
	require.NoError(t, os.WriteFile(empty, []byte("# nothing yet\n"), 0o600))

	cmd, _ = NewTestingCmd(t, "grant", "--vests", empty, "Acquisition payout: PANW")
	assert.EqualError(t, cmd.Execute(), "grant has no vests")
}
