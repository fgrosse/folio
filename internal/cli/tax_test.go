package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestTaxRateCmd covers setting the rate that vests are taxed at: "folio tax-rate" takes it as a
// percentage and stores it for the whole account.
func TestTaxRateCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "tax-rate", "44.3%")
	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	rate, err := store.TaxRate()
	require.NoError(t, err)
	require.True(t, rate.Valid)
	assert.Equal(t, "44.3", rate.Decimal.String())
}
