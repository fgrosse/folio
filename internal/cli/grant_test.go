package cli

import (
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
